// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package micro

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	registrypb "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	oreg "github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/reva/v2/pkg/app"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/rs/zerolog/log"
	mreg "go-micro.dev/v4/registry"
)

type manager struct {
	namespace string
	sync.RWMutex
	cancelFunc context.CancelFunc
	mimeTypes  map[string][]*registrypb.ProviderInfo
	providers  []*registrypb.ProviderInfo
	config     *config
}

// New returns an implementation of the app.Registry interface.
func New(m map[string]interface{}) (app.Registry, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}
	c.init()

	ctx, cancelFunc := context.WithCancel(context.Background())

	newManager := manager{
		namespace:  c.Namespace,
		cancelFunc: cancelFunc,
		config:     c,
	}

	err = newManager.updateProvidersFromMicroRegistry()
	if err != nil {
		if _, ok := err.(errtypes.NotFound); !ok {
			return nil, err
		}
	}

	t := time.NewTicker(time.Second * 30)

	go func() {
		for {
			select {
			case <-t.C:
				log.Debug().Msg("app provider tick, updating local app list")
				err = newManager.updateProvidersFromMicroRegistry()
				if err != nil {
					log.Error().Err(err).Msg("could not update the local provider cache")
					continue
				}
			case <-ctx.Done():
				log.Debug().Msg("app provider stopped")
				t.Stop()
			}
		}
	}()

	return &newManager, nil
}

// AddProvider does not do anything for this registry, it is a placeholder to satisfy the interface
func (m *manager) AddProvider(ctx context.Context, p *registrypb.ProviderInfo) error {
	log := appctx.GetLogger(ctx)

	log.Info().Interface("provider", p).Msg("Tried to register through cs3 api, make sure the provider registers directly through go-micro")

	return nil
}

// FindProvider returns all providers that can provide an app for the given mimeType
func (m *manager) FindProviders(ctx context.Context, mimeType string) ([]*registrypb.ProviderInfo, error) {
	m.RLock()
	defer m.RUnlock()

	if len(m.mimeTypes[mimeType]) < 1 {
		return nil, mreg.ErrNotFound
	}

	return m.mimeTypes[mimeType], nil
}

// GetDefaultProviderForMimeType returns the default provider for the given mimeType
func (m *manager) GetDefaultProviderForMimeType(ctx context.Context, mimeType string) (*registrypb.ProviderInfo, error) {
	m.RLock()
	defer m.RUnlock()

	for _, mt := range m.config.MimeTypes {
		if mt.MimeType != mimeType {
			continue
		}
		for _, p := range m.mimeTypes[mimeType] {
			if p.Name == mt.DefaultApp {
				return p, nil
			}
		}
	}

	return nil, mreg.ErrNotFound
}

// ListProviders lists all registered Providers
func (m *manager) ListProviders(ctx context.Context) ([]*registrypb.ProviderInfo, error) {
	return m.providers, nil
}

// ListSupportedMimeTypes lists all supported mimeTypes
func (m *manager) ListSupportedMimeTypes(ctx context.Context) ([]*registrypb.MimeTypeInfo, error) {
	m.RLock()
	defer m.RUnlock()

	res := make([]*registrypb.MimeTypeInfo, 0, len(m.config.MimeTypes))
	for _, mime := range m.config.MimeTypes {
		res = append(res, &registrypb.MimeTypeInfo{
			MimeType:           mime.MimeType,
			Ext:                mime.Extension,
			Name:               mime.Name,
			Description:        mime.Description,
			Icon:               mime.Icon,
			AppProviders:       m.mimeTypes[mime.MimeType],
			AllowCreation:      mime.AllowCreation,
			DefaultApplication: mime.DefaultApp,
		})
	}
	return res, nil
}

// SetDefaultProviderForMimeType sets the default provider for the given mimeType
func (m *manager) SetDefaultProviderForMimeType(ctx context.Context, mimeType string, p *registrypb.ProviderInfo) error {
	m.Lock()
	defer m.Unlock()
	// NOTE: this is a dirty workaround:

	for _, mt := range m.config.MimeTypes {
		if mt.MimeType == mimeType {
			mt.DefaultApp = p.Name
			return nil
		}
	}

	log.Info().Msgf("default provider for app is not set through the provider, but defined for the app")
	return mreg.ErrNotFound
}

func (m *manager) getProvidersFromMicroRegistry(ctx context.Context) ([]*registrypb.ProviderInfo, error) {
	reg := oreg.GetRegistry()
	services, err := reg.GetService(m.namespace+".api.app-provider", mreg.GetContext(ctx))
	if err != nil {
		log.Warn().Err(err).Msg("getProvidersFromMicroRegistry")
	}

	if len(services) == 0 {
		return nil, errtypes.NotFound("no application provider service registered")
	}
	if len(services) > 1 {
		return nil, errtypes.InternalError("more than one application provider services registered")
	}

	providers := make([]*registrypb.ProviderInfo, 0, len(services[0].Nodes))
	for _, node := range services[0].Nodes {
		p := m.providerFromMetadata(node.Metadata)
		p.Address = node.Address
		providers = append(providers, p)
	}
	return providers, nil
}

func (m *manager) providerFromMetadata(metadata map[string]string) *registrypb.ProviderInfo {
	p := &registrypb.ProviderInfo{
		MimeTypes: splitMimeTypes(metadata[m.namespace+".app-provider.mime_type"]),
		//		Address:     node.Address,
		Name:        metadata[m.namespace+".app-provider.name"],
		Description: metadata[m.namespace+".app-provider.description"],
		Icon:        metadata[m.namespace+".app-provider.icon"],
		DesktopOnly: metadata[m.namespace+".app-provider.desktop_only"] == "true",
		Capability:  registrypb.ProviderInfo_Capability(registrypb.ProviderInfo_Capability_value[metadata[m.namespace+".app-provider.capability"]]),
	}
	if metadata[m.namespace+".app-provider.priority"] != "" {
		p.Opaque = &typesv1beta1.Opaque{Map: map[string]*typesv1beta1.OpaqueEntry{
			"priority": {
				Decoder: "plain",
				Value:   []byte(metadata[m.namespace+".app-provider.priority"]),
			},
		}}
	}
	return p
}

func (m *manager) updateProvidersFromMicroRegistry() error {
	lst, err := m.getProvidersFromMicroRegistry(context.Background())
	ma := map[string][]*registrypb.ProviderInfo{}
	if err != nil {
		return err
	}
	sortByPriority(lst)
	for _, outer := range lst {
		for _, inner := range outer.MimeTypes {
			ma[inner] = append(ma[inner], outer)
		}
	}
	m.Lock()
	defer m.Unlock()
	m.mimeTypes = ma
	m.providers = lst
	return nil
}

func equalsProviderInfo(p1, p2 *registrypb.ProviderInfo) bool {
	sameName := p1.Name == p2.Name
	sameAddress := p1.Address == p2.Address

	if sameName && sameAddress {
		return true
	}
	return false
}

func getPriority(p *registrypb.ProviderInfo) string {
	if p.Opaque != nil && len(p.Opaque.Map) != 0 {
		if priority, ok := p.Opaque.Map["priority"]; ok {
			return string(priority.GetValue())
		}
	}
	return defaultPriority
}

func sortByPriority(providers []*registrypb.ProviderInfo) {
	less := func(i, j int) bool {
		prioI, _ := strconv.ParseInt(getPriority(providers[i]), 10, 64)
		prioJ, _ := strconv.ParseInt(getPriority(providers[j]), 10, 64)
		return prioI < prioJ
	}

	sort.Slice(providers, less)
}
