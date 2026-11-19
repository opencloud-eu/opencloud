/*
 * Copyright 2026 Kopano and its licensors
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

package resources

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// RegistryData is the base structure of the resources section of our client
// registry configuration file. All other sections of that file are parsed by
// their own registry and are ignored here.
type RegistryData struct {
	Resources []*Resource `yaml:"resources,flow"`
}

// Resource is one protected resource, an MCP server, as registered in the
// resources section of the client registration configuration file.
type Resource struct {
	ID     string   `yaml:"id"`
	Name   string   `yaml:"name"`
	Scopes []string `yaml:"scopes,flow"`

	// ImplicitScopes are granted for this resource whether a client asked for
	// them or not. They are meant for the scopes which are not a permission of
	// the resource but change what a token carries, and a client has no reason
	// to know about.
	ImplicitScopes []string `yaml:"implicit_scopes,flow"`

	scopes         map[string]bool // built at load
	implicitScopes map[string]bool // built at load
}

// AllowsScope returns true when the provided scope is one the resource accepts.
func (r *Resource) AllowsScope(scope string) bool {
	return r.scopes[scope] || r.implicitScopes[scope]
}

// ApplyImplicitScopes adds the implicit scopes of the associated resource to
// the provided scopes.
func (r *Resource) ApplyImplicitScopes(scopes map[string]bool) {
	for scope := range r.implicitScopes {
		scopes[scope] = true
	}
}

// Registry implements the registry for registered protected resources.
type Registry struct {
	mutex sync.RWMutex

	resources map[string]*Resource
	ordered   []*Resource

	logger logrus.FieldLogger
}

// NewRegistry creates a new resource Registry with the provided parameters. An
// empty registrationConfFilepath yields an empty registry.
func NewRegistry(ctx context.Context, registrationConfFilepath string, logger logrus.FieldLogger) (*Registry, error) {
	registryData := &RegistryData{}

	if registrationConfFilepath != "" {
		logger.Debugf("parsing resources registration conf from %v", registrationConfFilepath)
		registryFile, err := os.ReadFile(registrationConfFilepath)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(registryFile, registryData)
		if err != nil {
			return nil, err
		}
	}

	r := &Registry{
		resources: make(map[string]*Resource),

		logger: logger,
	}

	for _, resource := range registryData.Resources {
		if err := r.register(resource); err != nil {
			return nil, err
		}

		fields := logrus.Fields{
			"id":     resource.ID,
			"name":   resource.Name,
			"scopes": resource.Scopes,
		}
		logger.WithFields(fields).Debugln("registered resource")
	}

	return r, nil
}

// register validates the provided resource registration, normalizes its
// identifier and adds it to the associated registry. Returns error otherwise.
func (r *Registry) register(resource *Resource) error {
	if resource.ID == "" {
		return fmt.Errorf("resource entry without id")
	}

	id, err := NormalizeResourceID(resource.ID)
	if err != nil {
		return fmt.Errorf("invalid resource id %v: %w", resource.ID, err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.resources[id]; ok {
		return fmt.Errorf("duplicate resource id: %v", id)
	}

	// Write back the normalized value, it is what ends up in token audiences.
	resource.ID = id
	resource.scopes = make(map[string]bool, len(resource.Scopes))
	for _, scope := range resource.Scopes {
		resource.scopes[scope] = true
	}
	resource.implicitScopes = make(map[string]bool, len(resource.ImplicitScopes))
	for _, scope := range resource.ImplicitScopes {
		resource.implicitScopes[scope] = true
	}

	r.resources[id] = resource
	r.ordered = append(r.ordered, resource)

	return nil
}

// Get returns the registered resource for the provided resource identifier,
// which is normalized before lookup.
func (r *Registry) Get(resourceID string) (*Resource, bool) {
	id, err := NormalizeResourceID(resourceID)
	if err != nil {
		return nil, false
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	resource, ok := r.resources[id]
	return resource, ok
}

// Empty returns true when no resource is registered.
func (r *Registry) Empty() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.resources) == 0
}

// ScopesSupported returns the union of the scopes, implicit ones included, of
// all registered resources.
func (r *Registry) ScopesSupported() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var scopes []string
	seen := make(map[string]bool)
	for _, resource := range r.ordered {
		for _, scope := range resource.allScopes() {
			if seen[scope] {
				continue
			}
			seen[scope] = true
			scopes = append(scopes, scope)
		}
	}

	return scopes
}

// ValidateScopes returns an error when a registered resource names a scope
// which is not in the provided list of supported scopes. An undeclared scope
// would be dropped when authorizing a request, so a resource asking for one is
// a configuration error.
func (r *Registry) ValidateScopes(supported []string) error {
	supportedSet := make(map[string]bool, len(supported))
	for _, scope := range supported {
		supportedSet[scope] = true
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, resource := range r.ordered {
		for _, scope := range resource.allScopes() {
			if !supportedSet[scope] {
				return fmt.Errorf("resource %v declares unsupported scope: %v", resource.ID, scope)
			}
		}
	}

	return nil
}

// allScopes returns the accepted and the implicit scopes of the associated
// resource, in registration order.
func (r *Resource) allScopes() []string {
	return append(append([]string{}, r.Scopes...), r.ImplicitScopes...)
}

// NormalizeResourceID validates and normalizes a resource identifier as
// defined by RFC 8707 section 2 and the MCP canonical server URI rules,
// returning an error when the value cannot be one.
func NormalizeResourceID(resourceID string) (string, error) {
	if resourceID == "" {
		return "", fmt.Errorf("is empty")
	}

	u, err := url.Parse(resourceID)
	if err != nil {
		return "", fmt.Errorf("is not a valid URI: %w", err)
	}
	if !u.IsAbs() {
		return "", fmt.Errorf("is not an absolute URI")
	}
	// Check the raw value as well, since an empty fragment or query is not
	// visible in the parsed result but still makes the identifier non
	// canonical.
	if u.Fragment != "" || strings.Contains(resourceID, "#") {
		return "", fmt.Errorf("must not contain a fragment")
	}
	if u.RawQuery != "" || u.ForceQuery || strings.Contains(resourceID, "?") {
		return "", fmt.Errorf("must not contain a query")
	}
	if u.User != nil {
		return "", fmt.Errorf("must not contain user information")
	}
	if u.Host == "" {
		return "", fmt.Errorf("must contain a host")
	}

	// The spec says servers SHOULD accept uppercase scheme and host, so
	// normalize both to lower case. The port is part of the host and consists
	// of digits only, thus it is kept as is.
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Host)

	switch scheme {
	case "https":
		// Always allowed.
	case "http":
		if !isLoopbackHost(u.Hostname()) {
			return "", fmt.Errorf("scheme http is only allowed for loopback hosts")
		}
	default:
		return "", fmt.Errorf("scheme %v is not allowed", scheme)
	}

	// Use the escaped path so percent encoding is kept as provided, and strip
	// a single trailing slash so that a resource with and without it is the
	// same resource.
	path := strings.TrimSuffix(u.EscapedPath(), "/")

	return scheme + "://" + host + path, nil
}

// isLoopbackHost returns true when the provided host name, without port, is a
// loopback host as used when running a resource locally for development.
func isLoopbackHost(hostname string) bool {
	switch strings.ToLower(hostname) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}
