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

package spaces

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	providerpb "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	registrypb "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/logger"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	pkgregistry "github.com/opencloud-eu/reva/v2/pkg/storage/registry/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"google.golang.org/grpc"
)

func init() {
	pkgregistry.Register("spaces", NewDefault)
}

type spaceConfig struct {
	// MountPoint determines where a space is mounted. Can be a regex
	// It is used to determine which storage provider is responsible when only a path is given in the request
	MountPoint string `mapstructure:"mount_point"`
	// PathTemplate is used to build the path of an individual space. Layouts can access {{.Space...}} and {{.CurrentUser...}}
	PathTemplate string `mapstructure:"path_template"`
	template     *template.Template
	// filters
	OwnerIsCurrentUser bool   `mapstructure:"owner_is_current_user"`
	ID                 string `mapstructure:"id"`
	// TODO description?
}

// SpacePath generates a layout based on space data.
func (sc *spaceConfig) SpacePath(currentUser *userpb.User, space *providerpb.StorageSpace) (string, error) {
	b := bytes.Buffer{}
	if err := sc.template.Execute(&b, templateData{CurrentUser: currentUser, Space: space}); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Provider holds information on Spaces
type Provider struct {
	// Spaces is a map from space type to space config
	Spaces     map[string]*spaceConfig `mapstructure:"spaces"`
	ProviderID string                  `mapstructure:"providerid"`
}

type templateData struct {
	CurrentUser *userpb.User
	Space       *providerpb.StorageSpace
}

// StorageProviderClient is the interface the spaces registry uses to interact with storage providers
type StorageProviderClient interface {
	ListStorageSpaces(ctx context.Context, in *providerpb.ListStorageSpacesRequest, opts ...grpc.CallOption) (*providerpb.ListStorageSpacesResponse, error)
}

type config struct {
	Providers map[string]*Provider `mapstructure:"providers"`
}

func (c *config) init() {

	if len(c.Providers) == 0 {
		c.Providers = map[string]*Provider{
			sharedconf.GetGatewaySVC(""): {
				Spaces: map[string]*spaceConfig{
					"personal":   {MountPoint: "/users", PathTemplate: "/users/{{.Space.Owner.Id.OpaqueId}}"},
					"project":    {MountPoint: "/projects", PathTemplate: "/projects/{{.Space.Name}}"},
					"virtual":    {MountPoint: "/users/{{.CurrentUser.Id.OpaqueId}}/Shares"},
					"grant":      {MountPoint: "."},
					"mountpoint": {MountPoint: "/users/{{.CurrentUser.Id.OpaqueId}}/Shares", PathTemplate: "/users/{{.CurrentUser.Id.OpaqueId}}/Shares/{{.Space.Name}}"},
					"public":     {MountPoint: "/public"},
				},
			},
		}
	}

	// cleanup space paths
	for _, provider := range c.Providers {
		for _, space := range provider.Spaces {

			if space.MountPoint == "" {
				space.MountPoint = "/"
			}

			// if the path template is not explicitly set use the mount point as path template
			if space.PathTemplate == "" {
				space.PathTemplate = space.MountPoint
			}

			// cleanup path templates
			space.PathTemplate = filepath.Join("/", space.PathTemplate)

			// compile given template tpl
			var err error
			space.template, err = template.New("path_template").Funcs(sprig.TxtFuncMap()).Parse(space.PathTemplate)
			if err != nil {
				logger.New().Fatal().Err(err).Interface("space", space).Msg("error parsing template")
			}
		}

		// TODO connect to provider, (List Spaces,) ListContainerStream
	}
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		return nil, err
	}
	return c, nil
}

// New creates an implementation of the storage.Registry interface that
// uses the available storage spaces from the configured storage providers
func New(m map[string]interface{}, getClientFunc GetSpacesProviderServiceClientFunc) (storage.Registry, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}
	c.init()
	r := &registry{
		c:                              c,
		resources:                      make(map[string][]*registrypb.ProviderInfo),
		resourceNameCache:              make(map[string]string),
		getSpacesProviderServiceClient: getClientFunc,
	}
	return r, nil
}

// NewDefault creates an implementation of the storage.Registry interface that
// uses the available storage spaces from the configured storage providers
func NewDefault(m map[string]interface{}) (storage.Registry, error) {
	getClientFunc := func(addr string) (StorageProviderClient, error) {
		return pool.GetSpacesProviderServiceClient(addr)
	}
	return New(m, getClientFunc)
}

// GetStorageProviderServiceClientFunc is a callback used to pass in a StorageProviderClient during testing
type GetSpacesProviderServiceClientFunc func(addr string) (StorageProviderClient, error)

type registry struct {
	c *config
	// a map of resources to providers
	resources         map[string][]*registrypb.ProviderInfo
	resourceNameCache map[string]string

	getSpacesProviderServiceClient GetSpacesProviderServiceClientFunc
}

// GetProvider return the storage provider for the given spaces according to the rule configuration
func (r *registry) GetProvider(ctx context.Context, space *providerpb.StorageSpace) (*registrypb.ProviderInfo, error) {
	for address, provider := range r.c.Providers {
		for spaceType, sc := range provider.Spaces {
			spacePath := ""
			var err error
			if space.SpaceType != "" && spaceType != space.SpaceType {
				continue
			}
			if space.Owner != nil {
				user := ctxpkg.ContextMustGetUser(ctx)
				spacePath, err = sc.SpacePath(user, space)
				if err != nil {
					continue
				}
				if match, err := regexp.MatchString(sc.MountPoint, spacePath); err != nil || !match {
					continue
				}
			}

			setPath(space, spacePath)

			p := &registrypb.ProviderInfo{
				Address: address,
			}
			validSpaces := []*providerpb.StorageSpace{space}
			if err := setSpaces(p, validSpaces); err != nil {
				appctx.GetLogger(ctx).Debug().Err(err).Interface("provider", provider).Interface("spaces", validSpaces).Msg("marshaling spaces failed, continuing")
				continue
			}
			return p, nil // return the first match we find
		}
	}
	return nil, errtypes.NotFound("no provider found for space")
}

// FIXME the config takes the mount path of a provider as key,
// - it will always be used as the Providerpath
// - if the mount path is a regex, the provider config needs a providerpath config that is used instead of the regex
// - the gateway ALWAYS replaces the mountpath with the spaceid? and builds a relative reference which is forwarded to the responsible provider

// FindProviders will return all providers that need to be queried for a request
// - for an id based or relative request it will return the providers that serve the storage space
// - for a path based request it will return the provider with the most specific mount path, as
//   well as all spaces mounted below the requested path. Stat and ListContainer requests need
//   to take their etag/mtime into account.
// The list of providers also contains the space that should be used as the root for the relative path
//
// Given providers mounted at /home, /personal, /public, /shares, /foo and /foo/sub
// When a stat for / arrives
// Then the gateway needs all providers below /
// -> all providers
//
// When a stat for /home arrives
// Then the gateway needs all providers below /home
// -> only the /home provider
//
// When a stat for /foo arrives
// Then the gateway needs all providers below /foo
// -> the /foo and /foo/sub providers
//
// Given providers mounted at /foo, /foo/sub and /foo/sub/bar
// When a MKCOL for /foo/bif arrives
// Then the ocdav will make a stat for /foo/bif
// Then the gateway only needs the provider /foo
// -> only the /foo provider

// When a MKCOL for /foo/sub/mob arrives
// Then the ocdav will make a stat for /foo/sub/mob
// Then the gateway needs all providers below /foo/sub
// -> only the /foo/sub provider
//
// requested path   provider path
// above   = /foo           <=> /foo/bar        -> stat(spaceid, .)    -> add metadata for /foo/bar
// above   = /foo           <=> /foo/bar/bif    -> stat(spaceid, .)    -> add metadata for /foo/bar
// matches = /foo/bar       <=> /foo/bar        -> list(spaceid, .)
// below   = /foo/bar/bif   <=> /foo/bar        -> list(spaceid, ./bif)
func (r *registry) ListProviders(ctx context.Context, filters map[string]string) ([]*registrypb.ProviderInfo, error) {
	unique, _ := strconv.ParseBool(filters["unique"])
	unrestricted, _ := strconv.ParseBool(filters["unrestricted"])
	mask := filters["mask"]
	switch {
	case filters["space_id"] != "":

		findMountpoint := filters["type"] == "mountpoint"
		findGrant := !findMountpoint && filters["path"] == "" // relative references, by definition, occur in the correct storage, so do not look for grants
		// If opaque_id is empty, we assume that we are looking for a space root
		if filters["opaque_id"] == "" {
			filters["opaque_id"] = filters["space_id"]
		}
		id := storagespace.FormatResourceID(&providerpb.ResourceId{
			StorageId: filters["storage_id"],
			SpaceId:   filters["space_id"],
			OpaqueId:  filters["opaque_id"],
		})

		return r.findProvidersForResource(ctx, id, findMountpoint, findGrant, unrestricted, mask), nil
	case filters["path"] != "":
		return r.findProvidersForAbsolutePathReference(ctx, filters["path"], unique, unrestricted, mask), nil
	case len(filters) == 0:
		// return all providers
		return r.findAllProviders(ctx, mask), nil
	default:
		return r.findProvidersForFilter(ctx, r.buildFilters(filters), unrestricted, mask), nil
	}
}

func (r *registry) buildFilters(filterMap map[string]string) []*providerpb.ListStorageSpacesRequest_Filter {
	filters := []*providerpb.ListStorageSpacesRequest_Filter{}
	for k, f := range filterMap {
		switch k {
		case "space_id":
			filters = append(filters, &providerpb.ListStorageSpacesRequest_Filter{
				Type: providerpb.ListStorageSpacesRequest_Filter_TYPE_ID,
				Term: &providerpb.ListStorageSpacesRequest_Filter_Id{
					Id: &providerpb.StorageSpaceId{
						OpaqueId: f,
					},
				},
			})
		case "space_type":
			filters = append(filters, &providerpb.ListStorageSpacesRequest_Filter{
				Type: providerpb.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE,
				Term: &providerpb.ListStorageSpacesRequest_Filter_SpaceType{
					SpaceType: f,
				},
			})
		}
	}
	if filterMap["user_id"] != "" {
		filters = append(filters, &providerpb.ListStorageSpacesRequest_Filter{
			Type: providerpb.ListStorageSpacesRequest_Filter_TYPE_USER,
			Term: &providerpb.ListStorageSpacesRequest_Filter_User{
				User: &userpb.UserId{
					Idp:      filterMap["user_idp"],
					OpaqueId: filterMap["user_id"],
				},
			},
		})
	}
	if filterMap["owner_id"] != "" && filterMap["owner_idp"] != "" {
		filters = append(filters, &providerpb.ListStorageSpacesRequest_Filter{
			Type: providerpb.ListStorageSpacesRequest_Filter_TYPE_OWNER,
			Term: &providerpb.ListStorageSpacesRequest_Filter_Owner{
				Owner: &userpb.UserId{
					Idp:      filterMap["owner_idp"],
					OpaqueId: filterMap["owner_id"],
				},
			},
		})
	}
	return filters
}

func (r *registry) findProvidersForFilter(ctx context.Context, filters []*providerpb.ListStorageSpacesRequest_Filter, unrestricted bool, _ string) []*registrypb.ProviderInfo {

	var requestedSpaceType string
	for _, f := range filters {
		if f.Type == providerpb.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE {
			requestedSpaceType = f.GetSpaceType()
		}
	}

	currentUser := ctxpkg.ContextMustGetUser(ctx)
	providerInfos := []*registrypb.ProviderInfo{}
	for address, provider := range r.c.Providers {

		// when a specific space type is requested we may skip this provider altogether if it is not configured for that type
		// we have to ignore a space type filter with +grant or +mountpoint type because they can live on any provider
		if requestedSpaceType != "" && !strings.HasPrefix(requestedSpaceType, "+") {
			found := false
			for spaceType := range provider.Spaces {
				if spaceType == requestedSpaceType {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		p := &registrypb.ProviderInfo{
			Address: address,
		}
		spaces, err := r.findStorageSpaceOnProvider(ctx, address, filters, unrestricted)
		if err != nil {
			appctx.GetLogger(ctx).Debug().Err(err).Interface("provider", provider).Msg("findStorageSpaceOnProvider by id failed, continuing")
			continue
		}

		validSpaces := []*providerpb.StorageSpace{}
		if len(spaces) > 0 {
			for _, space := range spaces {
				var sc *spaceConfig
				var ok bool
				var spacePath string
				// filter unconfigured space types
				if sc, ok = provider.Spaces[space.SpaceType]; !ok {
					continue
				}
				spacePath, err = sc.SpacePath(currentUser, space)
				if err != nil {
					appctx.GetLogger(ctx).Error().Err(err).Interface("provider", provider).Interface("space", space).Msg("failed to execute template, continuing")
					continue
				}

				setPath(space, spacePath)
				validSpaces = append(validSpaces, space)
			}

			if len(validSpaces) > 0 {
				if err := setSpaces(p, validSpaces); err != nil {
					appctx.GetLogger(ctx).Debug().Err(err).Interface("provider", provider).Interface("spaces", validSpaces).Msg("marshaling spaces failed, continuing")
					continue
				}
				providerInfos = append(providerInfos, p)
			}
		}
	}
	return providerInfos
}

// findProvidersForResource looks up storage providers based on a resource id
// for the root of a space the res.SpaceId is the same as the res.OpaqueId
// for share spaces the res.SpaceId tells the registry the spaceid and res.OpaqueId is a node in that space
func (r *registry) findProvidersForResource(ctx context.Context, id string, findMoundpoint, findGrant, unrestricted bool, mask string) []*registrypb.ProviderInfo {
	currentUser := ctxpkg.ContextMustGetUser(ctx)
	providerInfos := []*registrypb.ProviderInfo{}
	rid, err := storagespace.ParseID(id)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("splitting spaceid failed")
		return nil
	}

	for address, provider := range r.c.Providers {
		p := &registrypb.ProviderInfo{
			Address:    address,
			ProviderId: rid.StorageId,
		}
		// try to find provider based on storageproviderid prefix if only root is requested
		if provider.ProviderID != "" && rid.StorageId != "" && mask == "root" {
			match, err := regexp.MatchString("^"+provider.ProviderID+"$", rid.StorageId)
			if err != nil || !match {
				continue
			}
			// construct space based on configured properties without actually making a ListStorageSpaces call
			space := &providerpb.StorageSpace{
				Id:   &providerpb.StorageSpaceId{OpaqueId: id},
				Root: &rid,
			}
			// this is a request for requests by id
			// setPath(space, provider.Path) // hmm not enough info to build a path.... the space alias is no longer known here we would need to query the provider

			validSpaces := []*providerpb.StorageSpace{space}
			if err := setSpaces(p, validSpaces); err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Interface("provider", provider).Interface("spaces", validSpaces).Msg("marshaling spaces failed, continuing")
				return nil
			}
			providerInfos = append(providerInfos, p)
			return providerInfos
		}
		if provider.ProviderID != "" && rid.StorageId != "" {
			match, err := regexp.MatchString("^"+provider.ProviderID+"$", rid.StorageId)
			if err != nil || !match {
				// skip mismatching storageproviders
				continue
			}
		}
		filters := []*providerpb.ListStorageSpacesRequest_Filter{{
			Type: providerpb.ListStorageSpacesRequest_Filter_TYPE_ID,
			Term: &providerpb.ListStorageSpacesRequest_Filter_Id{
				Id: &providerpb.StorageSpaceId{
					OpaqueId: id,
				},
			},
		}}
		if findMoundpoint {
			// when listing by id return also grants and mountpoints
			filters = append(filters, &providerpb.ListStorageSpacesRequest_Filter{
				Type: providerpb.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE,
				Term: &providerpb.ListStorageSpacesRequest_Filter_SpaceType{
					SpaceType: "+mountpoint",
				},
			})
		}
		if findGrant {
			// when listing by id return also grants and mountpoints
			filters = append(filters, &providerpb.ListStorageSpacesRequest_Filter{
				Type: providerpb.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE,
				Term: &providerpb.ListStorageSpacesRequest_Filter_SpaceType{
					SpaceType: "+grant",
				},
			})
		}
		spaces, err := r.findStorageSpaceOnProvider(ctx, address, filters, unrestricted)
		if err != nil {
			appctx.GetLogger(ctx).Debug().Err(err).Interface("provider", provider).Msg("findStorageSpaceOnProvider by id failed, continuing")
			continue
		}

		switch len(spaces) {
		case 0:
			// nothing to do, will continue with next provider
		case 1:
			space := spaces[0]

			var sc *spaceConfig
			var ok bool
			var spacePath string

			if space.SpaceType == "grant" {
				spacePath = "." // a . indicates a grant, the gateway will do a findMountpoint for it
			} else {
				if findMoundpoint && space.SpaceType != "mountpoint" {
					continue
				}
				// filter unwanted space types. type mountpoint is not explicitly configured but requested by the gateway
				if sc, ok = provider.Spaces[space.SpaceType]; !ok && space.SpaceType != "mountpoint" {
					continue
				}

				spacePath, err = sc.SpacePath(currentUser, space)
				if err != nil {
					appctx.GetLogger(ctx).Error().Err(err).Interface("provider", provider).Interface("space", space).Msg("failed to execute template, continuing")
					continue
				}

				setPath(space, spacePath)
			}
			validSpaces := []*providerpb.StorageSpace{space}
			if err := setSpaces(p, validSpaces); err != nil {
				appctx.GetLogger(ctx).Debug().Err(err).Interface("provider", provider).Interface("spaces", validSpaces).Msg("marshaling spaces failed, continuing")
				continue
			}
			// we can stop after we found the first space
			// TODO to improve lookup time the registry could cache which provider last was responsible for a space? could be invalidated by simple ttl? would that work for shares?
			// return []*registrypb.ProviderInfo{p}
			providerInfos = append(providerInfos, p) // hm we need to query all providers ... or the id based lookup might only see the spaces storage provider
		default:
			// there should not be multiple spaces with the same id per provider
			appctx.GetLogger(ctx).Error().Err(err).Interface("provider", provider).Interface("spaces", spaces).Msg("multiple spaces returned, ignoring")
		}
	}
	return providerInfos
}

// findProvidersForAbsolutePathReference takes a path and returns the storage provider with the longest matching path prefix
// FIXME use regex to return the correct provider when multiple are configured
func (r *registry) findProvidersForAbsolutePathReference(ctx context.Context, requestedPath string, unique, unrestricted bool, _ string) []*registrypb.ProviderInfo {
	currentUser := ctxpkg.ContextMustGetUser(ctx)

	pathSegments := strings.Split(strings.TrimPrefix(requestedPath, string(os.PathSeparator)), string(os.PathSeparator))
	deepestMountPath := ""
	var deepestMountSpace *providerpb.StorageSpace
	var deepestMountPathProvider *registrypb.ProviderInfo
	providers := map[string]*registrypb.ProviderInfo{}
	for address, provider := range r.c.Providers {
		p := &registrypb.ProviderInfo{
			Opaque:  &typesv1beta1.Opaque{Map: map[string]*typesv1beta1.OpaqueEntry{}},
			Address: address,
		}
		var spaces []*providerpb.StorageSpace
		var err error

		// check if any space in the provider has a valid mountpoint
		containsRelatedSpace := false

	spaceLoop:
		for _, space := range provider.Spaces {
			spacePath, _ := space.SpacePath(currentUser, nil)
			spacePathSegments := strings.Split(strings.TrimPrefix(spacePath, string(os.PathSeparator)), string(os.PathSeparator))

			for i, segment := range spacePathSegments {
				if i >= len(pathSegments) {
					break
				}
				if pathSegments[i] != segment {
					if segment != "" && !strings.Contains(segment, "{{") {
						// Mount path points elsewhere -> irrelevant
						continue spaceLoop
					}
					// Encountered a template which couldn't be filled -> potentially relevant
					break
				}
			}

			containsRelatedSpace = true
			break
		}

		if !containsRelatedSpace {
			continue
		}

		// when listing paths also return mountpoints
		filters := []*providerpb.ListStorageSpacesRequest_Filter{
			{
				Type: providerpb.ListStorageSpacesRequest_Filter_TYPE_PATH,
				Term: &providerpb.ListStorageSpacesRequest_Filter_Path{
					Path: strings.TrimPrefix(requestedPath, p.ProviderPath), // FIXME this no longer has an effect as the p.Providerpath is always empty
				},
			},
			{
				Type: providerpb.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE,
				Term: &providerpb.ListStorageSpacesRequest_Filter_SpaceType{
					SpaceType: "+mountpoint",
				},
			},
		}

		spaces, err = r.findStorageSpaceOnProvider(ctx, p.Address, filters, unrestricted)
		if err != nil {
			appctx.GetLogger(ctx).Debug().Err(err).Interface("provider", provider).Msg("findStorageSpaceOnProvider failed, continuing")
			continue
		}

		validSpaces := []*providerpb.StorageSpace{}
		for _, space := range spaces {
			var sc *spaceConfig
			var ok bool

			if space.SpaceType == "grant" {
				setPath(space, ".") // a . indicates a grant, the gateway will do a findMountpoint for it
				validSpaces = append(validSpaces, space)
				continue
			}

			// filter unwanted space types. type mountpoint is not explicitly configured but requested by the gateway
			if sc, ok = provider.Spaces[space.SpaceType]; !ok {
				continue
			}
			spacePath, err := sc.SpacePath(currentUser, space)
			if err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Interface("provider", provider).Interface("space", space).Msg("failed to execute template, continuing")
				continue
			}
			setPath(space, spacePath)

			// determine deepest mount point
			switch {
			case spacePath == requestedPath && unique:
				validSpaces = append(validSpaces, space)

				deepestMountPath = spacePath
				deepestMountSpace = space
				deepestMountPathProvider = p

			case !unique && isSubpath(spacePath, requestedPath):
				// and add all providers below and exactly matching the path
				// requested /foo, mountPath /foo/sub
				validSpaces = append(validSpaces, space)
				if len(spacePath) > len(deepestMountPath) {
					deepestMountPath = spacePath
					deepestMountSpace = space
					deepestMountPathProvider = p
				}

			case isSubpath(requestedPath, spacePath) && len(spacePath) > len(deepestMountPath):
				// eg. three providers: /foo, /foo/sub, /foo/sub/bar
				// requested /foo/sub/mob
				deepestMountPath = spacePath
				deepestMountSpace = space
				deepestMountPathProvider = p
			}
		}

		if len(validSpaces) > 0 {
			if err := setSpaces(p, validSpaces); err != nil {
				appctx.GetLogger(ctx).Debug().Err(err).Interface("provider", provider).Msg("marshaling spaces failed, continuing")
				continue
			}
			providers[p.Address] = p
		}
	}

	if deepestMountPathProvider != nil {
		if _, ok := providers[deepestMountPathProvider.Address]; !ok {
			if err := setSpaces(deepestMountPathProvider, []*providerpb.StorageSpace{deepestMountSpace}); err == nil {
				providers[deepestMountPathProvider.Address] = deepestMountPathProvider
			} else {
				appctx.GetLogger(ctx).Debug().Err(err).Interface("provider", deepestMountPathProvider).Interface("space", deepestMountSpace).Msg("marshaling space failed, continuing")
			}
		}
	}

	providerInfos := []*registrypb.ProviderInfo{}
	for _, p := range providers {
		providerInfos = append(providerInfos, p)
	}
	return providerInfos
}

// findAllProviders returns a list of all storage providers
// This is a dumb call that does not call ListStorageSpaces() on the providers: ListStorageSpaces() in the gateway can cache that better.
func (r *registry) findAllProviders(ctx context.Context, _ string) []*registrypb.ProviderInfo {
	pis := make([]*registrypb.ProviderInfo, 0, len(r.c.Providers))
	for address := range r.c.Providers {
		pis = append(pis, &registrypb.ProviderInfo{
			Address: address,
		})
	}
	return pis
}

func setPath(space *providerpb.StorageSpace, path string) {
	if space.Opaque == nil {
		space.Opaque = &typesv1beta1.Opaque{}
	}
	if space.Opaque.Map == nil {
		space.Opaque.Map = map[string]*typesv1beta1.OpaqueEntry{}
	}
	if _, ok := space.Opaque.Map["path"]; !ok {
		space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "path", path)
	}
}
func setSpaces(providerInfo *registrypb.ProviderInfo, spaces []*providerpb.StorageSpace) error {
	if providerInfo.Opaque == nil {
		providerInfo.Opaque = &typesv1beta1.Opaque{}
	}
	if providerInfo.Opaque.Map == nil {
		providerInfo.Opaque.Map = map[string]*typesv1beta1.OpaqueEntry{}
	}
	spacesBytes, err := json.Marshal(spaces)
	if err != nil {
		return err
	}
	providerInfo.Opaque.Map["spaces"] = &typesv1beta1.OpaqueEntry{
		Decoder: "json",
		Value:   spacesBytes,
	}
	return nil
}

func (r *registry) findStorageSpaceOnProvider(ctx context.Context, addr string, filters []*providerpb.ListStorageSpacesRequest_Filter, unrestricted bool) ([]*providerpb.StorageSpace, error) {
	c, err := r.getSpacesProviderServiceClient(addr)
	if err != nil {
		return nil, err
	}
	req := &providerpb.ListStorageSpacesRequest{
		Opaque: &typesv1beta1.Opaque{
			Map: map[string]*typesv1beta1.OpaqueEntry{
				"unrestricted": {
					Decoder: "plain",
					Value:   []byte(strconv.FormatBool(unrestricted)),
				},
			},
		},
		Filters: filters,
	}

	res, err := c.ListStorageSpaces(ctx, req)
	if err != nil {
		// ignore errors
		return nil, nil
	}
	return res.StorageSpaces, nil
}

// isSubpath determines if `p` is a subpath of `path`
func isSubpath(p string, path string) bool {
	if p == path {
		return true
	}

	r, err := filepath.Rel(path, p)
	if err != nil {
		return false
	}

	return r != ".." && !strings.HasPrefix(r, "../")
}
