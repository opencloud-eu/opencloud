// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
)

// Policy state bitfield constants for the unified policyState atomic.Int32.
// Each policy carries a single atomic.Int32 that encodes both runtime state
// (from DiscoveryUpdate) and operator overrides (from OPENSEARCH_GO_POLICY_*
// environment variables) as independent bit flags.
//
// Resolution priority in IsEnabled():
//
//	psEnvDisabled -> false  (highest: operator says off)
//	psEnabled     -> true   (runtime: has connections)
//	default       -> false
//
// psEnvEnabled is set when OPENSEARCH_GO_POLICY_*=true so observers can note
// the origin of the value (vs defaults or user supplied configuration code).
const (
	psEnabled     int32 = 1 << 0 // Runtime: policy has connections / is active
	psDisabled    int32 = 1 << 1 // Runtime: policy has no connections / is inactive
	psEnvEnabled  int32 = 1 << 2 // Env override: OPENSEARCH_GO_POLICY_*=true
	psEnvDisabled int32 = 1 << 3 // Env override: OPENSEARCH_GO_POLICY_*=false

	psDynamicMask int32 = psEnabled | psDisabled
	psEnvMask     int32 = psEnvEnabled | psEnvDisabled
)

// psIsEnabled resolves the policyState bitfield into a boolean, allowing
// environment variable overrides to influence the result. See
// [parsePolicyOverrides] and the policyState bit constants above for details.
func psIsEnabled(state int32) bool {
	// Env-set override takes precedence over runtime state.
	if state&psEnvDisabled != 0 {
		return false
	}
	// Otherwise honor the runtime bit set by DiscoveryUpdate.
	return state&psEnabled != 0
}

// psSetEnabled atomically updates the dynamic enabled/disabled bits
// without clobbering the env override bits.
func psSetEnabled(s *atomic.Int32, enabled bool) {
	for {
		old := s.Load()
		next := old &^ psDynamicMask
		if enabled {
			next |= psEnabled
		} else {
			next |= psDisabled
		}
		if s.CompareAndSwap(old, next) {
			return
		}
	}
}

// psSetEnvOverride atomically sets the env override bits without
// clobbering the dynamic state bits. Called once at startup.
func psSetEnvOverride(s *atomic.Int32, enabled bool) {
	for {
		old := s.Load()
		next := old &^ psEnvMask
		if enabled {
			next |= psEnvEnabled
		} else {
			next |= psEnvDisabled
		}
		if s.CompareAndSwap(old, next) {
			return
		}
	}
}

const policyTypeNameUnknown = "unknown"

// Policy type name constants used by policyTypeName() methods and
// the policyTypeNames slice for env-var override matching.
const (
	policyTypeNameChain          = "chain"
	policyTypeNameMux            = "mux"
	policyTypeNameIfEnabled      = "ifenabled"
	policyTypeNameRouter         = "router"
	policyTypeNameRole           = "role"
	policyTypeNameRoundRobin     = "roundrobin"
	policyTypeNameCoordinator    = "coordinator"
	policyTypeNameNull           = "null"
	policyTypeNameIndexRouter    = "index_router"
	policyTypeNameDocumentRouter = "document_router"
)

// Human-readable action words used in policy-override debug logging.
const (
	overrideActionEnabled  = "enabled"
	overrideActionDisabled = "disabled"
)

// policySortKey returns a structural identity string for a policy that
// encodes its type name plus distinguishing configuration and children.
// Used as a deterministic sort key when sibling policies share the same
// policyTypeName (e.g., multiple poolRouter instances under
// a MuxPolicy).
//
// Examples:
//
//	RolePolicy("data")          -> "role:data"
//	poolRouter       -> "router:write(role:ingest)"
//	IfEnabledPolicy             -> "ifenabled(role:coordinating_only,null)"
//	PolicyChain                 -> "chain(ifenabled(...),roundrobin)"
func policySortKey(p Policy) string {
	switch v := p.(type) {
	case *RolePolicy:
		return "role:" + v.requiredRoleKey
	case *poolRouter:
		return "router:" + v.poolName + "(" + policySortKey(v.inner) + ")"
	case *IfEnabledPolicy:
		return "ifenabled(" + policySortKey(v.truePolicy) + "," + policySortKey(v.falsePolicy) + ")"
	case *PolicyChain:
		parts := make([]string, len(v.policies))
		for i, child := range v.policies {
			parts[i] = policySortKey(child)
		}
		return "chain(" + strings.Join(parts, ",") + ")"
	case *MuxPolicy:
		// MuxPolicy children are in a map; sort their keys for determinism.
		children := v.childPolicies() // already sorted
		parts := make([]string, len(children))
		for i, child := range children {
			parts[i] = policySortKey(child)
		}
		return "mux(" + strings.Join(parts, ",") + ")"
	default:
		if typed, ok := p.(policyTyped); ok {
			return typed.policyTypeName()
		}
		return policyTypeNameUnknown
	}
}

// policyTyped is implemented by policies that expose a type name for
// environment variable matching and path construction.
type policyTyped interface {
	policyTypeName() string
}

// policyOverrider is implemented by policies that support environment
// variable overrides via OPENSEARCH_GO_POLICY_* variables.
type policyOverrider interface {
	setEnvOverride(enabled bool)
}

// policyOverride represents a parsed OPENSEARCH_GO_POLICY_<TYPE> env var.
type policyOverride struct {
	typeName string        // e.g., "role"
	envKey   string        // e.g., "OPENSEARCH_GO_POLICY_ROLE"
	applyAll *bool         // non-nil when the entire value is a bool
	matchers []pathMatcher // parsed from comma-separated path=bool items
}

// pathMatcher matches policy paths using regex or string prefix.
type pathMatcher struct {
	raw     string         // original item string for debug logging
	pattern string         // path portion (left of last '=')
	enable  bool           // action: true=enable, false=disable
	regex   *regexp.Regexp // non-nil if pattern compiled as valid regex
}

// policyTypeNames is the canonical list of policy type names used for
// environment variable lookup. Each entry corresponds to a policy struct
// that implements policyTyped.
//
//nolint:gochecknoglobals // Package-level constant list used by parsePolicyOverrides.
var policyTypeNames = []string{
	policyTypeNameChain,
	policyTypeNameMux,
	policyTypeNameIfEnabled,
	policyTypeNameRouter,
	policyTypeNameRole,
	policyTypeNameRoundRobin,
	policyTypeNameCoordinator,
	policyTypeNameNull,
	policyTypeNameIndexRouter,
	policyTypeNameDocumentRouter,
}

// parsePolicyOverrides reads OPENSEARCH_GO_POLICY_* environment variables
// and returns parsed overrides. Only variables that are set AND non-empty
// are processed (os.LookupEnv + non-empty check).
//
// Parsing priority for each env var value:
//  1. strconv.ParseBool(value) -- if parseable, applies to ALL instances of that type
//  2. Comma-separated items, each "path=<bool>":
//     a. Split on last '=', try ParseBool(right) -- if valid, left is path pattern
//     b. If right side isn't valid bool: treat entire item as regex then prefix
func parsePolicyOverrides() []policyOverride {
	var overrides []policyOverride

	for _, name := range policyTypeNames {
		envKey := "OPENSEARCH_GO_POLICY_" + strings.ToUpper(name)
		envVal, ok := os.LookupEnv(envKey)
		if !ok || envVal == "" {
			continue
		}

		override := policyOverride{
			typeName: name,
			envKey:   envKey,
		}

		// Priority 1: Try parsing the entire value as a bool.
		if b, err := strconv.ParseBool(envVal); err == nil {
			override.applyAll = &b
			overrides = append(overrides, override)
			continue
		}

		// Priority 2: Comma-separated path matchers.
		for item := range strings.SplitSeq(envVal, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}

			m := pathMatcher{raw: item}

			// Split on last '=' to separate path from bool value.
			if lastEq := strings.LastIndex(item, "="); lastEq >= 0 {
				path := item[:lastEq]
				val := item[lastEq+1:]

				if b, err := strconv.ParseBool(val); err == nil {
					m.pattern = path
					m.enable = b

					// Try compiling path as regex.
					if re, err := regexp.Compile(path); err == nil {
						m.regex = re
					}
					// If regex fails, pattern is used as string prefix match.

					override.matchers = append(override.matchers, m)
					continue
				}
			}

			// Fallback: no valid path=bool format.
			// Treat entire item as regex, then prefix. Matches disable.
			m.pattern = item
			m.enable = false
			if re, err := regexp.Compile(item); err == nil {
				m.regex = re
			}
			override.matchers = append(override.matchers, m)
		}

		if override.applyAll != nil || len(override.matchers) > 0 {
			overrides = append(overrides, override)
		}
	}

	return overrides
}

// buildPolicyPaths walks a policy tree and assigns a dot-delimited path
// to each policy node. Paths use the format:
//
//	chain[0].ifenabled[0].mux[0].role[0]
//
// Sibling indices are per-type within each parent level.
func buildPolicyPaths(root Policy) map[Policy]string {
	paths := make(map[Policy]string)
	buildPolicyPathsRecursive(root, "", paths, make(map[string]int))
	return paths
}

func buildPolicyPathsRecursive(p Policy, parentPath string, paths map[Policy]string, siblingCounts map[string]int) {
	typeName := policyTypeNameUnknown
	if typed, ok := p.(policyTyped); ok {
		typeName = typed.policyTypeName()
	}

	idx := siblingCounts[typeName]
	siblingCounts[typeName] = idx + 1

	var path string
	if parentPath == "" {
		path = fmt.Sprintf("%s[%d]", typeName, idx)
	} else {
		path = fmt.Sprintf("%s.%s[%d]", parentPath, typeName, idx)
	}
	paths[p] = path

	if walker, ok := p.(policyTreeWalker); ok {
		children := walker.childPolicies()

		// Sort children by structural identity for deterministic path assignment.
		slices.SortFunc(children, func(a, b Policy) int {
			return strings.Compare(policySortKey(a), policySortKey(b))
		})

		childCounts := make(map[string]int)
		for _, child := range children {
			buildPolicyPathsRecursive(child, path, paths, childCounts)
		}
	}
}

// applyPolicyOverrides walks the policy tree and applies parsed overrides
// by calling setEnvOverride on matching policies.
func applyPolicyOverrides(root Policy, overrides []policyOverride) {
	if len(overrides) == 0 {
		return
	}

	paths := buildPolicyPaths(root)
	applyPolicyOverridesRecursive(root, overrides, paths)
}

func applyPolicyOverridesRecursive(p Policy, overrides []policyOverride, paths map[Policy]string) {
	path := paths[p]

	typeName := ""
	if typed, ok := p.(policyTyped); ok {
		typeName = typed.policyTypeName()
	}

	for _, override := range overrides {
		if override.typeName != typeName {
			continue
		}

		if override.applyAll != nil {
			// Bool case: applies to every instance of this type.
			overrider, ok := p.(policyOverrider)
			if !ok {
				continue
			}
			overrider.setEnvOverride(*override.applyAll)
			if dl := loadDebugLogger(); dl != nil {
				action := overrideActionEnabled
				if !*override.applyAll {
					action = overrideActionDisabled
				}
				dl.Logf("Policy override: %s %s at path %q (env: %s)\n",
					action, typeName, path, override.envKey)
			}
			continue
		}

		// Path matcher case.
		for _, m := range override.matchers {
			var matched bool
			if m.regex != nil {
				matched = m.regex.MatchString(path)
			} else {
				matched = strings.HasPrefix(path, m.pattern)
			}

			if !matched {
				continue
			}

			overrider, ok := p.(policyOverrider)
			if !ok {
				break
			}

			overrider.setEnvOverride(m.enable)
			if dl := loadDebugLogger(); dl != nil {
				action := overrideActionEnabled
				if !m.enable {
					action = overrideActionDisabled
				}
				dl.Logf("Policy override: %s %s at path %q (env: %s, matcher: %q)\n",
					action, typeName, path, override.envKey, m.raw)
			}
			break // First match wins for this override
		}
	}

	// Recurse into children regardless of this node's state,
	// so that child nodes are also marked for tree-dump accuracy.
	if walker, ok := p.(policyTreeWalker); ok {
		for _, child := range walker.childPolicies() {
			applyPolicyOverridesRecursive(child, overrides, paths)
		}
	}
}
