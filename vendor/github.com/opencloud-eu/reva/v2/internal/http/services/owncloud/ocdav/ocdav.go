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

package ocdav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/jellydator/ttlcache/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/config"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/global"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
	"github.com/opencloud-eu/reva/v2/pkg/signedurl"
	"github.com/opencloud-eu/reva/v2/pkg/storage/favorite"
	"github.com/opencloud-eu/reva/v2/pkg/storage/favorite/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// name is the Tracer name used to identify this instrumentation library.
const tracerName = "ocdav"

func init() {
	global.Register("ocdav", New)
}

type svc struct {
	c                *config.Config
	webDavHandler    *WebDavHandler
	davHandler       *DavHandler
	favoritesManager favorite.Manager
	client           *http.Client
	gatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	// LockSystem is the lock management system.
	LockSystem          LockSystem
	userIdentifierCache *ttlcache.Cache
	nameValidators      []Validator
	urlSigner           signedurl.Signer
}

func (s *svc) Config() *config.Config {
	return s.c
}

func getFavoritesManager(c *config.Config) (favorite.Manager, error) {
	if f, ok := registry.NewFuncs[c.FavoriteStorageDriver]; ok {
		return f(c.FavoriteStorageDrivers[c.FavoriteStorageDriver])
	}
	return nil, errtypes.NotFound("driver not found: " + c.FavoriteStorageDriver)
}
func getLockSystem(c *config.Config) (LockSystem, error) {
	// TODO in memory implementation
	selector, err := pool.GatewaySelector(c.GatewaySvc)
	if err != nil {
		return nil, err
	}
	return NewCS3LS(selector), nil
}

// New returns a new ocdav service
func New(m map[string]interface{}, log *zerolog.Logger) (global.Service, error) {
	conf := &config.Config{}
	if err := mapstructure.Decode(m, conf); err != nil {
		return nil, err
	}

	conf.Init()

	fm, err := getFavoritesManager(conf)
	if err != nil {
		return nil, err
	}
	ls, err := getLockSystem(conf)
	if err != nil {
		return nil, err
	}

	return NewWith(conf, fm, ls, log, nil)
}

// NewWith returns a new ocdav service
func NewWith(conf *config.Config, fm favorite.Manager, ls LockSystem, _ *zerolog.Logger, selector pool.Selectable[gateway.GatewayAPIClient]) (global.Service, error) {
	// be safe - init the conf again
	conf.Init()

	var signer signedurl.Signer
	if conf.URLSigningSharedSecret != "" {
		var err error
		signer, err = signedurl.NewJWTSignedURL(signedurl.WithSecret(conf.URLSigningSharedSecret))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize URL signer: %w", err)
		}
	}

	s := &svc{
		c:             conf,
		webDavHandler: new(WebDavHandler),
		davHandler:    new(DavHandler),
		client: rhttp.GetHTTPClient(
			rhttp.Timeout(time.Duration(conf.Timeout*int64(time.Second))),
			rhttp.Insecure(conf.Insecure),
		),
		gatewaySelector:     selector,
		favoritesManager:    fm,
		LockSystem:          ls,
		userIdentifierCache: ttlcache.NewCache(),
		nameValidators:      ValidatorsFromConfig(conf),
		urlSigner:           signer,
	}
	_ = s.userIdentifierCache.SetTTL(60 * time.Second)

	// initialize handlers and set default configs
	if err := s.webDavHandler.init(conf.WebdavNamespace, true); err != nil {
		return nil, err
	}
	if err := s.davHandler.init(conf); err != nil {
		return nil, err
	}
	if selector == nil {
		var err error
		s.gatewaySelector, err = pool.GatewaySelector(s.c.GatewaySvc)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *svc) Prefix() string {
	return s.c.Prefix
}

func (s *svc) Close() error {
	return nil
}

func (s *svc) Unprotected() []string {
	return []string{"/status.php", "/status", "/remote.php/dav/public-files/", "/apps/files/", "/index.php/f/", "/index.php/s/", "/remote.php/dav/ocm/", "/dav/ocm/"}
}

func (s *svc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := appctx.GetLogger(ctx)

		// TODO(jfd): do we need this?
		// fake litmus testing for empty namespace: see https://github.com/golang/net/blob/e514e69ffb8bc3c76a71ae40de0118d794855992/webdav/litmus_test_server.go#L58-L89
		if r.Header.Get(net.HeaderLitmus) == "props: 3 (propfind_invalid2)" {
			http.Error(w, "400 Bad Request", http.StatusBadRequest)
			return
		}

		// to build correct href prop urls we need to keep track of the base path
		// always starts with /
		base := path.Join("/", s.Prefix())

		var head string
		head, r.URL.Path = router.ShiftPath(r.URL.Path)
		log.Debug().Str("method", r.Method).Str("head", head).Str("tail", r.URL.Path).Msg("http routing")
		switch head {
		case "status.php", "status":
			s.doStatus(w, r)
			return
		case "remote.php":
			// skip optional "remote.php"
			head, r.URL.Path = router.ShiftPath(r.URL.Path)

			// yet, add it to baseURI
			base = path.Join(base, "remote.php")
		case "apps":
			head, r.URL.Path = router.ShiftPath(r.URL.Path)
			if head == "files" {
				s.handleLegacyPath(w, r)
				return
			}
		case "index.php":
			head, r.URL.Path = router.ShiftPath(r.URL.Path)
			if head == "s" {
				token := r.URL.Path
				rURL := s.c.PublicURL + path.Join(head, token)

				http.Redirect(w, r, rURL, http.StatusMovedPermanently)
				return
			}
		}
		switch head {
		// the old `/webdav` endpoint uses remote.php/webdav/$path
		case "webdav":
			// for oc we need to prepend /home as the path that will be passed to the home storage provider
			// will not contain the username
			base = path.Join(base, "webdav")
			ctx := context.WithValue(ctx, net.CtxKeyBaseURI, base)
			r = r.WithContext(ctx)
			s.webDavHandler.Handler(s).ServeHTTP(w, r)
			return
		case "dav":
			// cern uses /dav/files/$namespace -> /$namespace/...
			// oc uses /dav/files/$user -> /$home/$user/...
			// for oc we need to prepend the path to user homes
			// or we take the path starting at /dav and allow rewriting it?
			base = path.Join(base, "dav")
			ctx := context.WithValue(ctx, net.CtxKeyBaseURI, base)
			r = r.WithContext(ctx)
			s.davHandler.Handler(s).ServeHTTP(w, r)
			return
		}
		log.Warn().Msg("resource not found")
		w.WriteHeader(http.StatusNotFound)
	})
}

func (s *svc) ApplyLayout(ctx context.Context, ns string, useLoggedInUserNS bool, requestPath string) (string, string, error) {
	// If useLoggedInUserNS is false, that implies that the request is coming from
	// the FilesHandler method invoked by a /dav/files/fileOwner where fileOwner
	// is not the same as the logged in user. In that case, we'll treat fileOwner
	// as the username whose files are to be accessed and use that in the
	// namespace template.
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok || !useLoggedInUserNS {
		var requestUsernameOrID string
		requestUsernameOrID, requestPath = router.ShiftPath(requestPath)

		// Check if this is a Userid
		client, err := s.gatewaySelector.Next()
		if err != nil {
			return "", "", err
		}

		userRes, err := client.GetUser(ctx, &userpb.GetUserRequest{
			UserId: &userpb.UserId{OpaqueId: requestUsernameOrID},
		})
		if err != nil {
			return "", "", err
		}

		// If it's not a userid try if it is a user name
		if userRes.Status.Code != rpc.Code_CODE_OK {
			res, err := client.GetUserByClaim(ctx, &userpb.GetUserByClaimRequest{
				Claim: "username",
				Value: requestUsernameOrID,
			})
			if err != nil {
				return "", "", err
			}
			userRes.Status = res.Status
			userRes.User = res.User
		}

		// If still didn't find a user, fallback
		if userRes.Status.Code != rpc.Code_CODE_OK {
			userRes.User = &userpb.User{
				Username: requestUsernameOrID,
				Id:       &userpb.UserId{OpaqueId: requestUsernameOrID},
			}
		}

		u = userRes.User
	}

	return templates.WithUser(u, ns), requestPath, nil
}

func authContextForUser(client gateway.GatewayAPIClient, userID *userpb.UserId, machineAuthAPIKey string) (context.Context, error) {
	if machineAuthAPIKey == "" {
		return nil, errtypes.NotSupported("machine auth not configured")
	}
	// Get auth
	granteeCtx := ctxpkg.ContextSetUser(context.Background(), &userpb.User{Id: userID})

	authRes, err := client.Authenticate(granteeCtx, &gateway.AuthenticateRequest{
		Type:         "machine",
		ClientId:     "userid:" + userID.OpaqueId,
		ClientSecret: machineAuthAPIKey,
	})
	if err != nil {
		return nil, err
	}
	if authRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil, errtypes.NewErrtypeFromStatus(authRes.Status)
	}
	granteeCtx = metadata.AppendToOutgoingContext(granteeCtx, ctxpkg.TokenHeader, authRes.Token)
	return granteeCtx, nil
}

func (s *svc) sspReferenceIsChildOf(ctx context.Context, selector pool.Selectable[gateway.GatewayAPIClient], child, parent *provider.Reference) (bool, error) {
	client, err := selector.Next()
	if err != nil {
		return false, err
	}
	parentStatRes, err := client.Stat(ctx, &provider.StatRequest{Ref: parent})
	if err != nil {
		return false, err
	}
	if parentStatRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return false, errtypes.NewErrtypeFromStatus(parentStatRes.GetStatus())
	}
	parentAuthCtx, err := authContextForUser(client, parentStatRes.GetInfo().GetOwner(), s.c.MachineAuthAPIKey)
	if err != nil {
		return false, err
	}
	parentPathRes, err := client.GetPath(parentAuthCtx, &provider.GetPathRequest{ResourceId: parentStatRes.GetInfo().GetId()})
	if err != nil {
		return false, err
	}

	childStatRes, err := client.Stat(ctx, &provider.StatRequest{Ref: child})
	if err != nil {
		return false, err
	}
	if childStatRes.GetStatus().GetCode() == rpc.Code_CODE_NOT_FOUND && utils.IsRelativeReference(child) && child.Path != "." {
		childParentRef := &provider.Reference{
			ResourceId: child.ResourceId,
			Path:       utils.MakeRelativePath(path.Dir(child.Path)),
		}
		childStatRes, err = client.Stat(ctx, &provider.StatRequest{Ref: childParentRef})
		if err != nil {
			return false, err
		}
	}
	if childStatRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return false, errtypes.NewErrtypeFromStatus(parentStatRes.Status)
	}
	// TODO: this should use service accounts https://github.com/owncloud/ocis/issues/7597
	childAuthCtx, err := authContextForUser(client, childStatRes.GetInfo().GetOwner(), s.c.MachineAuthAPIKey)
	if err != nil {
		return false, err
	}
	childPathRes, err := client.GetPath(childAuthCtx, &provider.GetPathRequest{ResourceId: childStatRes.GetInfo().GetId()})
	if err != nil {
		return false, err
	}

	cp := childPathRes.Path + "/"
	pp := parentPathRes.Path + "/"
	return strings.HasPrefix(cp, pp), nil
}

func (s *svc) referenceIsChildOf(ctx context.Context, selector pool.Selectable[gateway.GatewayAPIClient], child, parent *provider.Reference) (bool, error) {
	if child.ResourceId.SpaceId != parent.ResourceId.SpaceId {
		return false, nil // Not on the same storage -> not a child
	}

	if utils.ResourceIDEqual(child.ResourceId, parent.ResourceId) {
		return strings.HasPrefix(child.Path, parent.Path+"/"), nil // Relative to the same resource -> compare paths
	}

	if child.ResourceId.SpaceId == utils.ShareStorageSpaceID || parent.ResourceId.SpaceId == utils.ShareStorageSpaceID {
		// the sharesstorageprovider needs some special handling
		return s.sspReferenceIsChildOf(ctx, selector, child, parent)
	}

	client, err := selector.Next()
	if err != nil {
		return false, err
	}

	// the references are on the same storage but relative to different resources
	// -> we need to get the path for both resources
	childPathRes, err := client.GetPath(ctx, &provider.GetPathRequest{ResourceId: child.ResourceId})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
			return false, nil // the storage provider doesn't support GetPath() -> rely on it taking care of recursion issues
		}
		return false, err
	}
	parentPathRes, err := client.GetPath(ctx, &provider.GetPathRequest{ResourceId: parent.ResourceId})
	if err != nil {
		return false, err
	}

	cp := path.Join(childPathRes.Path, child.Path) + "/"
	pp := path.Join(parentPathRes.Path, parent.Path) + "/"
	return strings.HasPrefix(cp, pp), nil
}

// filename returns the base filename from a path and replaces any slashes with an empty string
func filename(p string) string {
	return strings.Trim(path.Base(p), "/")
}

// isBodyEmpty returns true when the Body of the request is Empty
func isBodyEmpty(r *http.Request) bool {
	if r.Body != nil && r.Body != http.NoBody {
		buf := make([]byte, 0)
		_, err := r.Body.Read(buf)
		if err != io.EOF {
			// We currently do not support extended mkcol https://datatracker.ietf.org/doc/rfc5689/
			// TODO let clients send a body with properties to set on the new resource
			return false
		}
	}
	return true
}
