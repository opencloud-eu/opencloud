// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisioncleanup

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/options"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
)

// PolicyConfig holds the strategy specifications a Policy is made of, each as understood by New:
// empty means that the next, less specific one applies, "disabled" that no revisions are deleted.
type PolicyConfig struct {
	// Default is the strategy of the spaces that nothing more specific applies to.
	Default string
	// SpaceTypes are the strategies of the spaces of a type, by type, e.g.
	// storagespace.TypePersonal. An empty strategy, like a missing type, leaves the spaces of
	// that type to Default; "disabled" keeps all their revisions, whatever Default says.
	SpaceTypes map[string]string
	// Forced applies to all spaces, whatever the other settings say, e.g. the strategy given
	// to a command for a single run.
	Forced string
}

// PolicyConfigFromOptions returns the PolicyConfig configured in the driver options: the
// RevisionsCleanupStrategy* settings. It never has a Forced strategy.
func PolicyConfigFromOptions(o *options.Options) PolicyConfig {
	return PolicyConfig{
		Default: o.RevisionsCleanupStrategy,
		SpaceTypes: map[string]string{
			storagespace.TypePersonal: o.RevisionsCleanupStrategyPersonal,
			storagespace.TypeProject:  o.RevisionsCleanupStrategyProject,
		},
	}
}

// StrategySource tells which setting of a Policy the strategy of a file came from.
type StrategySource string

const (
	// SourceForced is the forced strategy, see PolicyConfig.Forced.
	SourceForced StrategySource = "forced"
	// SourceSpaceType is the strategy of the type of the file's space, see
	// PolicyConfig.SpaceTypes.
	SourceSpaceType StrategySource = "space type"
	// SourceDefault is the default strategy, see PolicyConfig.Default.
	SourceDefault StrategySource = "default"
)

// Policy decides which strategy, as a Cleaner, applies to a file. Its settings are layered, the
// most specific first, and the first one that is set applies:
//
//  1. the forced strategy (PolicyConfig.Forced)
//  2. the strategy of the space's type (PolicyConfig.SpaceTypes)
//  3. the default (PolicyConfig.Default)
//
// It decides by the file, which knows its space root, so that more layers can be added: for the
// settings of a space itself, from an attribute of its root, between 1 and 2, and for rules by
// the type or the size of the file.
//
// A Policy is created once, with all its strategies parsed, and can be used by many goroutines
// at the same time. A nil *Policy behaves like one without any strategy, as a safeguard.
type Policy struct {
	forced *Cleaner // nil without a forced strategy
	def    *Cleaner
	// only the types with a strategy of their own
	spaceTypes map[string]*Cleaner
	// the settings, without the empty ones, see String
	config PolicyConfig
}

// NewPolicy creates a Policy from its settings. It returns an error for an invalid strategy,
// naming the setting it belongs to.
func NewPolicy(c PolicyConfig) (*Policy, error) {
	p := &Policy{spaceTypes: map[string]*Cleaner{}}
	var err error
	if p.def, err = New(c.Default); err != nil {
		return nil, fmt.Errorf("invalid default strategy: %w", err)
	}
	p.config.Default = p.def.config
	for typ, spec := range c.SpaceTypes {
		if strings.TrimSpace(spec) == "" {
			continue
		}
		if p.spaceTypes[typ], err = New(spec); err != nil {
			return nil, fmt.Errorf("invalid strategy for %s spaces: %w", typ, err)
		}
		if p.config.SpaceTypes == nil {
			p.config.SpaceTypes = map[string]string{}
		}
		p.config.SpaceTypes[typ] = p.spaceTypes[typ].config
	}
	if strings.TrimSpace(c.Forced) != "" {
		if p.forced, err = New(c.Forced); err != nil {
			return nil, fmt.Errorf("invalid forced strategy: %w", err)
		}
		p.config.Forced = p.forced.config
	}
	return p, nil
}

// Enabled tells whether p may delete revisions in any space, i.e. whether any of its strategies
// is enabled; with a forced strategy, only that one counts. It is false for a nil Policy.
func (p *Policy) Enabled() bool {
	if p == nil {
		return false
	}
	if p.forced != nil {
		return p.forced.Enabled()
	}
	if p.def.Enabled() {
		return true
	}
	for _, c := range p.spaceTypes {
		if c.Enabled() {
			return true
		}
	}
	return false
}

// CleanerFor returns the Cleaner for the file n, and which setting its strategy came from. The
// Cleaner may be disabled, e.g. for a file in a space whose type has the strategy "disabled":
// none of its revisions are deleted then.
//
// It reads the type of the space from the space root of n only when there are strategies per
// type. It returns an error if that fails, or if n or its space root is unknown; the caller must
// not delete anything then, as it doesn't know the strategy that applies.
func (p *Policy) CleanerFor(ctx context.Context, n *node.Node) (*Cleaner, StrategySource, error) {
	if p == nil {
		return nil, SourceDefault, nil
	}
	if p.forced != nil {
		return p.forced, SourceForced, nil
	}
	if len(p.spaceTypes) == 0 {
		return p.def, SourceDefault, nil
	}
	if n == nil || n.SpaceRoot == nil {
		return nil, "", errors.New("the space root of the file is unknown")
	}
	// This reads nothing from disk for a node read with node.ReadNode: its space root has all
	// its attributes cached already, as ReadNode reads its name.
	typ, err := n.SpaceRoot.XattrString(ctx, prefixes.SpaceTypeAttr)
	if err != nil {
		return nil, "", fmt.Errorf("could not read the type of the space: %w", err)
	}
	c, source := p.cleanerForSpaceType(typ)
	return c, source, nil
}

// cleanerForSpaceType is CleanerFor without the forced strategy, for a space of the given type.
func (p *Policy) cleanerForSpaceType(typ string) (*Cleaner, StrategySource) {
	if c, ok := p.spaceTypes[typ]; ok {
		return c, SourceSpaceType
	}
	return p.def, SourceDefault
}

// String describes the settings of p for the log, with the types in a fixed order, e.g.
// "default: 30d; personal: 7d, 5; project: disabled". Two Policies with the same settings
// have the same String.
func (p *Policy) String() string {
	if p == nil {
		return "disabled"
	}
	parts := []string{"default: " + p.config.Default}
	for _, typ := range slices.Sorted(maps.Keys(p.config.SpaceTypes)) {
		parts = append(parts, typ+": "+p.config.SpaceTypes[typ])
	}
	if p.config.Forced != "" {
		parts = append(parts, "forced: "+p.config.Forced)
	}
	return strings.Join(parts, "; ")
}
