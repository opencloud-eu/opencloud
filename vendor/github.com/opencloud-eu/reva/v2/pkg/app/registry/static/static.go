// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

package static

import (
	"container/heap"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	registrypb "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/app"
	"github.com/opencloud-eu/reva/v2/pkg/app/registry/registry"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map"
)

func init() {
	registry.Register("static", New)
}

const defaultPriority = 0

type mimeTypeConfig struct {
	MimeType      string `mapstructure:"mime_type"`
	Extension     string `mapstructure:"extension"`
	Name          string `mapstructure:"name"`
	Description   string `mapstructure:"description"`
	Icon          string `mapstructure:"icon"`
	DefaultApp    string `mapstructure:"default_app"`
	AllowCreation bool   `mapstructure:"allow_creation"`
	apps          providerHeap
}

type config struct {
	Providers []*registrypb.ProviderInfo `mapstructure:"providers"`
	MimeTypes []*mimeTypeConfig          `mapstructure:"mime_types"`
}

func (c *config) init() {
	if len(c.Providers) == 0 {
		c.Providers = []*registrypb.ProviderInfo{}
	}
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		return nil, err
	}
	return c, nil
}

type manager struct {
	providers map[string]*registrypb.ProviderInfo
	mimetypes *orderedmap.OrderedMap // map[string]*mimeTypeConfig  ->  map the mime type to the addresses of the corresponding providers
	sync.RWMutex
}

// New returns an implementation of the app.Registry interface.
func New(m map[string]interface{}) (app.Registry, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}
	c.init()

	mimetypes := orderedmap.New()

	for _, mime := range c.MimeTypes {
		mimetypes.Set(mime.MimeType, mime)
	}

	providerMap := make(map[string]*registrypb.ProviderInfo)
	for _, p := range c.Providers {
		providerMap[p.Address] = p
	}

	// register providers configured manually from the config
	// (different from the others that are registering themselves -
	// dinamically added invoking the AddProvider function)
	for _, p := range c.Providers {
		if p != nil {
			for _, m := range p.MimeTypes {
				if v, ok := mimetypes.Get(m); ok {
					mtc := v.(*mimeTypeConfig)
					registerProvider(p, mtc)
				} else {
					return nil, errtypes.NotFound(fmt.Sprintf("mimetype %s not found in the configuration", m))
				}
			}
		}
	}

	newManager := manager{
		providers: providerMap,
		mimetypes: mimetypes,
	}
	return &newManager, nil
}

// remove a provider from the provider list in a mime type
// it's a no-op if the provider is not in the list of providers in the mime type
func unregisterProvider(p *registrypb.ProviderInfo, mime *mimeTypeConfig) {
	if index, in := getIndex(mime.apps, p); in {
		// remove the provider from the list
		heap.Remove(&mime.apps, index)
	}
}

func registerProvider(p *registrypb.ProviderInfo, mime *mimeTypeConfig) {
	// the app provider could be previously registered to the same mime type list
	// so we will remove it
	unregisterProvider(p, mime)

	heap.Push(&mime.apps, providerWithPriority{
		provider: p,
		priority: getPriority(p),
	})
}

func getPriority(p *registrypb.ProviderInfo) uint64 {
	if p.Opaque != nil && len(p.Opaque.Map) != 0 {
		if priority, ok := p.Opaque.Map["priority"]; ok {
			if pr, err := strconv.ParseUint(string(priority.GetValue()), 10, 64); err == nil {
				return pr
			}
		}
	}
	return defaultPriority
}

func (m *manager) FindProviders(ctx context.Context, mimeType string) ([]*registrypb.ProviderInfo, error) {
	// find longest match
	var match string

	m.RLock()
	defer m.RUnlock()

	for pair := m.mimetypes.Oldest(); pair != nil; pair = pair.Next() {
		prefix := pair.Key.(string)
		if strings.HasPrefix(mimeType, prefix) && len(prefix) > len(match) {
			match = prefix
		}
	}

	if match == "" {
		return nil, errtypes.NotFound("application provider not found for mime type " + mimeType)
	}

	mimeInterface, _ := m.mimetypes.Get(match)
	mimeMatch := mimeInterface.(*mimeTypeConfig)
	var providers = make([]*registrypb.ProviderInfo, 0, len(mimeMatch.apps))
	for _, p := range mimeMatch.apps {
		providers = append(providers, m.providers[p.provider.Address])
	}
	return providers, nil
}

func (m *manager) AddProvider(ctx context.Context, p *registrypb.ProviderInfo) error {
	m.Lock()
	defer m.Unlock()

	// check if the provider was already registered
	// if it's the case, we have to unregister it
	// from all the old mime types
	if oldP, ok := m.providers[p.Address]; ok {
		oldMimeTypes := oldP.MimeTypes
		for _, mimeName := range oldMimeTypes {
			mimeIf, ok := m.mimetypes.Get(mimeName)
			if !ok {
				continue
			}
			mime := mimeIf.(*mimeTypeConfig)
			unregisterProvider(p, mime)
		}
	}

	m.providers[p.Address] = p

	for _, mime := range p.MimeTypes {
		if mimeTypeInterface, ok := m.mimetypes.Get(mime); ok {
			mimeType := mimeTypeInterface.(*mimeTypeConfig)
			registerProvider(p, mimeType)
		} else {
			// the mime type should be already registered as config in the AppRegistry
			// we will create a new entry fot the mimetype, but leaving a warning for
			// future log inspection for weird behaviour
			// log.Warn().Msgf("config for mimetype '%s' not found while adding a new AppProvider", m)
			m.mimetypes.Set(mime, dummyMimeType(mime, []*registrypb.ProviderInfo{p}))
		}
	}
	return nil
}

func (m *manager) ListProviders(ctx context.Context) ([]*registrypb.ProviderInfo, error) {
	m.RLock()
	defer m.RUnlock()

	providers := make([]*registrypb.ProviderInfo, 0, len(m.providers))
	for _, p := range m.providers {
		providers = append(providers, p)
	}
	return providers, nil
}

func (m *manager) ListSupportedMimeTypes(ctx context.Context) ([]*registrypb.MimeTypeInfo, error) {
	m.RLock()
	defer m.RUnlock()

	res := make([]*registrypb.MimeTypeInfo, 0, m.mimetypes.Len())

	for pair := m.mimetypes.Oldest(); pair != nil; pair = pair.Next() {

		mime := pair.Value.(*mimeTypeConfig)

		res = append(res, &registrypb.MimeTypeInfo{
			MimeType:           mime.MimeType,
			Ext:                mime.Extension,
			Name:               mime.Name,
			Description:        mime.Description,
			Icon:               mime.Icon,
			AppProviders:       mime.apps.getOrderedProviderByPriority(),
			AllowCreation:      mime.AllowCreation,
			DefaultApplication: mime.DefaultApp,
		})

	}

	return res, nil
}

func (h providerHeap) getOrderedProviderByPriority() []*registrypb.ProviderInfo {
	providers := make([]*registrypb.ProviderInfo, 0, h.Len())
	for _, pp := range h {
		providers = append(providers, pp.provider)
	}
	return providers
}

func getIndex(h providerHeap, s *registrypb.ProviderInfo) (int, bool) {
	for i, e := range h {
		if equalsProviderInfo(e.provider, s) {
			return i, true
		}
	}
	return -1, false
}

func (m *manager) SetDefaultProviderForMimeType(ctx context.Context, mimeType string, p *registrypb.ProviderInfo) error {
	m.Lock()
	defer m.Unlock()

	mimeInterface, ok := m.mimetypes.Get(mimeType)
	if ok {
		mime := mimeInterface.(*mimeTypeConfig)
		mime.DefaultApp = p.Address

		registerProvider(p, mime)
	} else {
		// the mime type should be already registered as config in the AppRegistry
		// we will create a new entry fot the mimetype, but leaving a warning for
		// future log inspection for weird behaviour
		log.Warn().Msgf("config for mimetype '%s' not found while setting a new default AppProvider", mimeType)
		m.mimetypes.Set(mimeType, dummyMimeType(mimeType, []*registrypb.ProviderInfo{p}))
	}
	return nil
}

func dummyMimeType(m string, apps []*registrypb.ProviderInfo) *mimeTypeConfig {
	appsHeap := providerHeap{}
	for _, p := range apps {
		heap.Push(&appsHeap, providerWithPriority{
			provider: p,
			priority: getPriority(p),
		})
	}

	return &mimeTypeConfig{
		MimeType: m,
		apps:     appsHeap,
		//Extension: "", // there is no meaningful general extension, so omit it
		//Name:        "", // there is no meaningful general name, so omit it
		//Description: "", // there is no meaningful general description, so omit it
	}
}

func (m *manager) GetDefaultProviderForMimeType(ctx context.Context, mimeType string) (*registrypb.ProviderInfo, error) {
	m.RLock()
	defer m.RUnlock()

	mimeInterface, ok := m.mimetypes.Get(mimeType)
	if ok {
		mime := mimeInterface.(*mimeTypeConfig)
		// default by provider address
		if p, ok := m.providers[mime.DefaultApp]; ok {
			return p, nil
		}

		// default by provider name
		for _, p := range m.providers {
			if p.Name == mime.DefaultApp {
				return p, nil
			}
		}
	}

	return nil, errtypes.NotFound("default application provider not set for mime type " + mimeType)
}

func equalsProviderInfo(p1, p2 *registrypb.ProviderInfo) bool {
	return p1.Name == p2.Name
}

// check that all providers in the two lists are equals
func providersEquals(l1, l2 []*registrypb.ProviderInfo) bool {
	if len(l1) != len(l2) {
		return false
	}

	for i := 0; i < len(l1); i++ {
		if !equalsProviderInfo(l1[i], l2[i]) {
			return false
		}
	}
	return true
}

type providerWithPriority struct {
	provider *registrypb.ProviderInfo
	priority uint64
}

type providerHeap []providerWithPriority

func (h providerHeap) Len() int {
	return len(h)
}

func (h providerHeap) Less(i, j int) bool {
	return h[i].priority > h[j].priority
}

func (h providerHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *providerHeap) Push(x interface{}) {
	*h = append(*h, x.(providerWithPriority))
}

func (h *providerHeap) Pop() interface{} {
	last := len(*h) - 1
	x := (*h)[last]
	*h = (*h)[:last]
	return x
}
