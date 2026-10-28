/*
 * Copyright 2017-2019 Kopano and its licensors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package scopes

import (
	"os"
	"sort"
	"strings"

	"github.com/libregraph/oidc-go"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	konnect "github.com/libregraph/lico"
)

const (
	scopeAliasBasic = "basic"
	scopeUnknown    = "unknown"
)

const (
	priorityBasic         = 20
	priorityOfflineAccess = 10
)

var defaultScopesMap = map[string]string{
	oidc.ScopeOpenID:  scopeAliasBasic,
	oidc.ScopeEmail:   scopeAliasBasic,
	oidc.ScopeProfile: scopeAliasBasic,

	konnect.ScopeNumericID:    scopeAliasBasic,
	konnect.ScopeUniqueUserID: scopeAliasBasic,
	konnect.ScopeRawSubject:   scopeAliasBasic,
}

var defaultScopesDefinitionMap = map[string]*Definition{
	scopeAliasBasic: &Definition{
		ID:       "scope_alias_basic",
		Priority: priorityBasic,
	},
	oidc.ScopeOfflineAccess: &Definition{
		ID:       "scope_offline_access",
		Priority: priorityOfflineAccess,
	},
}

// Scopes contain collections for scope related meta data
type Scopes struct {
	Mapping     map[string]string      `json:"mapping" yaml:"mapping"`
	Definitions map[string]*Definition `json:"definitions" yaml:"scopes"`
}

// NewScopesFromIDs creates a new scopes meta data collection from the provided
// scopes IDs optionally also adding definitions from a parent.
func NewScopesFromIDs(scopes map[string]bool, parent *Scopes) *Scopes {
	mapping := make(map[string]string)
	definitions := make(map[string]*Definition)

	for scope, enabled := range scopes {
		if !enabled {
			continue
		}

		alias := scope
		if mapped, ok := parent.Mapping[scope]; ok {
			alias = mapped
			mapping[scope] = mapped
		} else if mapped, ok := defaultScopesMap[scope]; ok {
			alias = mapped
			mapping[scope] = mapped
		}

		if definition, ok := parent.Definitions[alias]; ok {
			definitions[alias] = definition
		} else if definition, ok := defaultScopesDefinitionMap[alias]; ok {
			definitions[alias] = definition
		}
	}

	return &Scopes{
		Mapping:     mapping,
		Definitions: definitions,
	}
}

// NewScopesFromFile loads scope definitions from a file.
func NewScopesFromFile(scopesConfFilepath string, logger logrus.FieldLogger) (*Scopes, error) {
	scopes := &Scopes{}

	if scopesConfFilepath != "" {
		logger.Debugf("parsing scopes conf from %v", scopesConfFilepath)
		confFile, err := os.ReadFile(scopesConfFilepath)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(confFile, scopes)
		if err != nil {
			return nil, err
		}

		for id, definition := range scopes.Definitions {
			// A blank translation would win over the plain description and
			// render an empty row, and an empty map would be marshalled into
			// every hello response, so both are dropped here rather than
			// guarded against per request.
			for locale, text := range definition.Descriptions {
				if strings.TrimSpace(text) == "" {
					delete(definition.Descriptions, locale)
				}
			}
			if len(definition.Descriptions) == 0 {
				definition.Descriptions = nil
			}

			locales := make([]string, 0, len(definition.Descriptions))
			for locale := range definition.Descriptions {
				locales = append(locales, locale)
			}
			sort.Strings(locales)

			fields := logrus.Fields{
				"id":       id,
				"priority": definition.Priority,
				// Yaml is parsed leniently, so a mistyped translations key is
				// silently ignored. This is where an operator sees what landed.
				"translations": strings.Join(locales, ","),
			}

			logger.WithFields(fields).Debugln("registered scope")
		}

		for id, mapped := range scopes.Mapping {
			fields := logrus.Fields{
				"id": id,
				"to": mapped,
			}

			logger.WithFields(fields).Debugln("registered scope mapping")
		}
	}

	if scopes.Mapping == nil {
		scopes.Mapping = make(map[string]string)
	}
	if scopes.Definitions == nil {
		scopes.Definitions = make(map[string]*Definition)
	}

	return scopes, nil
}

// Extend adds the provided scope mappings and definitions to the accociated
// scopes mappings and definitions with replacing already existing. If scopes is
// nil, Extends is a no-op.
func (s *Scopes) Extend(scopes *Scopes) error {
	if scopes == nil {
		return nil
	}

	for scope, definition := range scopes.Definitions {
		s.Definitions[scope] = definition
	}
	for mapped, mapping := range scopes.Mapping {
		s.Mapping[mapped] = mapping
	}

	return nil
}
