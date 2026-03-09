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
	"net/http"
	"path"
	"path/filepath"
	"strings"

	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/config"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

const (
	_trashbinPath = "trash-bin"

	// WwwAuthenticate captures the Www-Authenticate header string.
	WwwAuthenticate = "Www-Authenticate"
)

const (
	ErrListingMembers     = "ERR_LISTING_MEMBERS_NOT_ALLOWED"
	ErrInvalidCredentials = "ERR_INVALID_CREDENTIALS"
	ErrMissingBasicAuth   = "ERR_MISSING_BASIC_AUTH"
	ErrMissingBearerAuth  = "ERR_MISSING_BEARER_AUTH"
	ErrFileNotFoundInRoot = "ERR_FILE_NOT_FOUND_IN_ROOT"
)

// DavHandler routes to the different sub handlers
type DavHandler struct {
	AvatarsHandler      *AvatarsHandler
	FilesHandler        *WebDavHandler
	FilesHomeHandler    *WebDavHandler
	MetaHandler         *MetaHandler
	TrashbinHandler     *TrashbinHandler
	SpacesHandler       *SpacesHandler
	PublicFolderHandler *WebDavHandler
	PublicFileHandler   *PublicFileHandler
	SharesHandler       *WebDavHandler
	OCMSharesHandler    *WebDavHandler
}

func (h *DavHandler) init(c *config.Config) error {
	h.AvatarsHandler = new(AvatarsHandler)
	if err := h.AvatarsHandler.init(c); err != nil {
		return err
	}
	h.FilesHandler = new(WebDavHandler)
	if err := h.FilesHandler.init(c.FilesNamespace, false); err != nil {
		return err
	}
	h.FilesHomeHandler = new(WebDavHandler)
	if err := h.FilesHomeHandler.init(c.WebdavNamespace, true); err != nil {
		return err
	}
	h.MetaHandler = new(MetaHandler)
	if err := h.MetaHandler.init(c); err != nil {
		return err
	}
	h.TrashbinHandler = new(TrashbinHandler)
	if err := h.TrashbinHandler.init(c); err != nil {
		return err
	}

	h.SpacesHandler = new(SpacesHandler)
	if err := h.SpacesHandler.init(c); err != nil {
		return err
	}

	h.PublicFolderHandler = new(WebDavHandler)
	if err := h.PublicFolderHandler.init("public", true); err != nil { // jail public file requests to /public/ prefix
		return err
	}

	h.PublicFileHandler = new(PublicFileHandler)
	if err := h.PublicFileHandler.init("public"); err != nil { // jail public file requests to /public/ prefix
		return err
	}

	h.OCMSharesHandler = new(WebDavHandler)
	if err := h.OCMSharesHandler.init(c.OCMNamespace, true); err != nil {
		return err
	}

	return nil
}

func isOwner(userIDorName string, user *userv1beta1.User) bool {
	return userIDorName != "" && (userIDorName == user.Id.OpaqueId || strings.EqualFold(userIDorName, user.Username))
}

// Handler handles requests
func (h *DavHandler) Handler(s *svc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := appctx.GetLogger(ctx)

		// if there is no file in the request url we assume the request url is: "/remote.php/dav/files"
		// https://github.com/owncloud/core/blob/18475dac812064b21dabcc50f25ef3ffe55691a5/tests/acceptance/features/apiWebdavOperations/propfind.feature
		if r.URL.Path == "/files" {
			log.Debug().Str("path", r.URL.Path).Msg("method not allowed")
			contextUser, ok := ctxpkg.ContextGetUser(ctx)
			if ok {
				r.URL.Path = path.Join(r.URL.Path, contextUser.Username)
			}

			if r.Header.Get(net.HeaderDepth) == "" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				b, err := errors.Marshal(http.StatusMethodNotAllowed, "Listing members of this collection is disabled", "", ErrListingMembers)
				if err != nil {
					log.Error().Msgf("error marshaling xml response: %s", b)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				_, err = w.Write(b)
				if err != nil {
					log.Error().Msgf("error writing xml response: %s", b)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				return
			}
		}

		var head string
		head, r.URL.Path = router.ShiftPath(r.URL.Path)

		switch head {
		case "avatars":
			h.AvatarsHandler.Handler(s).ServeHTTP(w, r)
		case "files":
			var requestUserID string
			var oldPath = r.URL.Path

			// detect and check current user in URL
			requestUserID, r.URL.Path = router.ShiftPath(r.URL.Path)

			// note: some requests like OPTIONS don't forward the user
			contextUser, ok := ctxpkg.ContextGetUser(ctx)
			if ok && isOwner(requestUserID, contextUser) {
				// use home storage handler when user was detected
				base := path.Join(ctx.Value(net.CtxKeyBaseURI).(string), "files", requestUserID)
				ctx := context.WithValue(ctx, net.CtxKeyBaseURI, base)
				r = r.WithContext(ctx)

				h.FilesHomeHandler.Handler(s).ServeHTTP(w, r)
			} else {
				r.URL.Path = oldPath
				base := path.Join(ctx.Value(net.CtxKeyBaseURI).(string), "files")
				ctx := context.WithValue(ctx, net.CtxKeyBaseURI, base)
				r = r.WithContext(ctx)

				h.FilesHandler.Handler(s).ServeHTTP(w, r)
			}
		case "meta":
			base := path.Join(ctx.Value(net.CtxKeyBaseURI).(string), "meta")
			ctx = context.WithValue(ctx, net.CtxKeyBaseURI, base)
			r = r.WithContext(ctx)
			h.MetaHandler.Handler(s).ServeHTTP(w, r)
		case "ocm":
			base := path.Join(ctx.Value(net.CtxKeyBaseURI).(string), "ocm")
			ctx := context.WithValue(ctx, net.CtxKeyBaseURI, base)
			c, err := s.gatewaySelector.Next()
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			// OC10 and Nextcloud (OCM 1.0) are using basic auth for carrying the
			// ocm share id.
			var ocmshare, sharedSecret string
			username, _, ok := r.BasicAuth()
			if ok {
				// OCM 1.0
				ocmshare = username
				sharedSecret = username
				r.URL.Path = filepath.Join("/", ocmshare, r.URL.Path)
			} else {
				ocmshare, _ = router.ShiftPath(r.URL.Path)
				sharedSecret = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			}

			authRes, err := handleOCMAuth(ctx, c, ocmshare, sharedSecret)
			switch {
			case err != nil:
				log.Error().Err(err).Msg("error during ocm authentication")
				w.WriteHeader(http.StatusInternalServerError)
				return
			case authRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED:
				log.Debug().Str("ocmshare", ocmshare).Msg("permission denied")
				fallthrough
			case authRes.Status.Code == rpc.Code_CODE_UNAUTHENTICATED:
				log.Debug().Str("ocmshare", ocmshare).Msg("unauthorized")
				w.WriteHeader(http.StatusUnauthorized)
				return
			case authRes.Status.Code == rpc.Code_CODE_NOT_FOUND:
				log.Debug().Str("ocmshare", ocmshare).Msg("not found")
				w.WriteHeader(http.StatusNotFound)
				return
			case authRes.Status.Code != rpc.Code_CODE_OK:
				log.Error().Str("ocmshare", ocmshare).Interface("status", authRes.Status).Msg("grpc auth request failed")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			ctx = ctxpkg.ContextSetToken(ctx, authRes.Token)
			ctx = ctxpkg.ContextSetUser(ctx, authRes.User)
			ctx = metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, authRes.Token)

			log.Debug().Str("ocmshare", ocmshare).Interface("user", authRes.User).Msg("OCM user authenticated")

			r = r.WithContext(ctx)
			h.OCMSharesHandler.Handler(s).ServeHTTP(w, r)
		case "trash-bin":
			base := path.Join(ctx.Value(net.CtxKeyBaseURI).(string), "trash-bin")
			ctx := context.WithValue(ctx, net.CtxKeyBaseURI, base)
			r = r.WithContext(ctx)
			h.TrashbinHandler.Handler(s).ServeHTTP(w, r)
		case "spaces":
			base := path.Join(ctx.Value(net.CtxKeyBaseURI).(string), "spaces")
			ctx := context.WithValue(ctx, net.CtxKeyBaseURI, base)
			r = r.WithContext(ctx)
			h.SpacesHandler.Handler(s, h.TrashbinHandler).ServeHTTP(w, r)
		case "public-files":
			base := path.Join(ctx.Value(net.CtxKeyBaseURI).(string), "public-files")
			ctx = context.WithValue(ctx, net.CtxKeyBaseURI, base)

			var res *gatewayv1beta1.AuthenticateResponse
			token, _ := router.ShiftPath(r.URL.Path)
			var hasValidBasicAuthHeader bool
			var pass string
			var err error
			// If user is authenticated
			_, userExists := ctxpkg.ContextGetUser(ctx)
			if userExists {
				client, err := s.gatewaySelector.Next()
				if err != nil {
					log.Error().Err(err).Msg("error sending grpc stat request")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				psRes, err := client.GetPublicShare(ctx, &link.GetPublicShareRequest{
					Ref: &link.PublicShareReference{
						Spec: &link.PublicShareReference_Token{
							Token: token,
						},
					}})
				if err != nil && !strings.Contains(err.Error(), "core access token not found") {
					log.Error().Err(err).Msg("error sending grpc stat request")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// If the link is internal then 307 redirect
				if psRes.Status.Code == rpc.Code_CODE_OK && grants.PermissionsEqual(psRes.Share.Permissions.GetPermissions(), &provider.ResourcePermissions{}) {
					if psRes.GetShare().GetResourceId() != nil {
						rUrl := path.Join("/dav/spaces", storagespace.FormatResourceID(psRes.GetShare().GetResourceId()))
						http.Redirect(w, r, rUrl, http.StatusTemporaryRedirect)
						return
					}
					log.Debug().Str("token", token).Interface("status", psRes.Status).Msg("resource id not found")
					w.WriteHeader(http.StatusNotFound)
					return
				}
			}

			if _, pass, hasValidBasicAuthHeader = r.BasicAuth(); hasValidBasicAuthHeader {
				res, err = handleBasicAuth(r.Context(), s.gatewaySelector, token, pass)
			} else {
				q := r.URL.Query()
				sig := q.Get("signature")
				expiration := q.Get("expiration")
				// We restrict the pre-signed urls to downloads.
				if sig != "" && expiration != "" && (r.Method != http.MethodGet && r.Method != http.MethodHead) {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				res, err = handleSignatureAuth(r.Context(), s.gatewaySelector, token, sig, expiration)
			}

			switch {
			case err != nil:
				w.WriteHeader(http.StatusInternalServerError)
				return
			case res.Status.Code == rpc.Code_CODE_PERMISSION_DENIED:
				fallthrough
			case res.Status.Code == rpc.Code_CODE_UNAUTHENTICATED:
				w.WriteHeader(http.StatusUnauthorized)
				if hasValidBasicAuthHeader {
					b, err := errors.Marshal(http.StatusUnauthorized, "Username or password was incorrect", "", ErrInvalidCredentials)
					errors.HandleWebdavError(log, w, b, err)
					return
				}
				b, err := errors.Marshal(http.StatusUnauthorized, "No 'Authorization: Basic' header found", "", ErrMissingBasicAuth)
				errors.HandleWebdavError(log, w, b, err)
				return
			case res.Status.Code == rpc.Code_CODE_NOT_FOUND:
				w.WriteHeader(http.StatusNotFound)
				return
			case res.Status.Code != rpc.Code_CODE_OK:
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if userExists {
				// Build new context without an authenticated user.
				// the public link should be resolved by the 'publicshares' authenticated user
				baseURI := ctx.Value(net.CtxKeyBaseURI).(string)
				logger := appctx.GetLogger(ctx)
				span := trace.SpanFromContext(ctx)
				span.End()
				ctx = trace.ContextWithSpan(context.Background(), span)
				ctx = appctx.WithLogger(ctx, logger)
				ctx = context.WithValue(ctx, net.CtxKeyBaseURI, baseURI)
			}
			ctx = ctxpkg.ContextSetToken(ctx, res.Token)
			ctx = ctxpkg.ContextSetUser(ctx, res.User)
			ctx = metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, res.Token)

			r = r.WithContext(ctx)

			// the public share manager knew the token, but does the referenced target still exist?
			sRes, err := getTokenStatInfo(ctx, s.gatewaySelector, token)
			switch {
			case err != nil:
				log.Error().Err(err).Msg("error sending grpc stat request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			case sRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED:
				fallthrough
			case sRes.Status.Code == rpc.Code_CODE_OK && grants.PermissionsEqual(sRes.GetInfo().GetPermissionSet(), &provider.ResourcePermissions{}):
				// If the link is internal
				if !userExists {
					w.Header().Add(WwwAuthenticate, fmt.Sprintf("Bearer realm=\"%s\", charset=\"UTF-8\"", r.Host))
					w.WriteHeader(http.StatusUnauthorized)
					b, err := errors.Marshal(http.StatusUnauthorized, "No 'Authorization: Bearer' header found", "", ErrMissingBearerAuth)
					errors.HandleWebdavError(log, w, b, err)
					return
				}
				fallthrough
			case sRes.Status.Code == rpc.Code_CODE_NOT_FOUND:
				log.Debug().Str("token", token).Interface("status", res.Status).Msg("Resource not found")
				w.WriteHeader(http.StatusNotFound) // log the difference
				return
			case sRes.Status.Code == rpc.Code_CODE_UNAUTHENTICATED:
				log.Debug().Str("token", token).Interface("status", res.Status).Msg("unauthorized")
				w.WriteHeader(http.StatusUnauthorized)
				return
			case sRes.Status.Code != rpc.Code_CODE_OK:
				log.Error().Str("token", token).Interface("status", res.Status).Msg("grpc stat request failed")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Debug().Interface("statInfo", sRes.Info).Msg("Stat info from public link token path")

			ctx := ContextWithTokenStatInfo(ctx, sRes.Info)
			r = r.WithContext(ctx)
			if sRes.Info.Type != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
				h.PublicFileHandler.Handler(s).ServeHTTP(w, r)
			} else {
				h.PublicFolderHandler.Handler(s).ServeHTTP(w, r)
			}

		default:
			w.WriteHeader(http.StatusNotFound)
			b, err := errors.Marshal(http.StatusNotFound, "File not found in root", "", ErrFileNotFoundInRoot)
			errors.HandleWebdavError(log, w, b, err)
		}
	})
}

func getTokenStatInfo(ctx context.Context, selector pool.Selectable[gatewayv1beta1.GatewayAPIClient], token string) (*provider.StatResponse, error) {
	client, err := selector.Next()
	if err != nil {
		return nil, err
	}

	return client.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: utils.PublicStorageProviderID,
			SpaceId:   utils.PublicStorageSpaceID,
			OpaqueId:  token,
		},
	}})
}

func handleBasicAuth(ctx context.Context, selector pool.Selectable[gatewayv1beta1.GatewayAPIClient], token, pw string) (*gatewayv1beta1.AuthenticateResponse, error) {
	c, err := selector.Next()
	if err != nil {
		return nil, err
	}
	authenticateRequest := gatewayv1beta1.AuthenticateRequest{
		Type:         "publicshares",
		ClientId:     token,
		ClientSecret: "password|" + pw,
	}

	return c.Authenticate(ctx, &authenticateRequest)
}

func handleSignatureAuth(ctx context.Context, selector pool.Selectable[gatewayv1beta1.GatewayAPIClient], token, sig, expiration string) (*gatewayv1beta1.AuthenticateResponse, error) {
	c, err := selector.Next()
	if err != nil {
		return nil, err
	}
	authenticateRequest := gatewayv1beta1.AuthenticateRequest{
		Type:         "publicshares",
		ClientId:     token,
		ClientSecret: "signature|" + sig + "|" + expiration,
	}

	return c.Authenticate(ctx, &authenticateRequest)
}

func handleOCMAuth(ctx context.Context, c gatewayv1beta1.GatewayAPIClient, ocmshare, sharedSecret string) (*gatewayv1beta1.AuthenticateResponse, error) {
	return c.Authenticate(ctx, &gatewayv1beta1.AuthenticateRequest{
		Type:         "ocmshares",
		ClientId:     ocmshare,
		ClientSecret: sharedSecret,
	})
}
