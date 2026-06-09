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

package appprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strings"

	appregistry "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	iso6391 "github.com/emvi/iso-639-1"
	"github.com/go-chi/chi/v5"
	"github.com/go-viper/mapstructure/v2"
	ua "github.com/mileusna/useragent"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/global"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
)

func init() {
	global.Register("appprovider", New)
}

// Config holds the config options for the HTTP appprovider service
type Config struct {
	Prefix            string `mapstructure:"prefix"`
	GatewaySvc        string `mapstructure:"gatewaysvc"`
	Insecure          bool   `mapstructure:"insecure"`
	WebBaseURI        string `mapstructure:"webbaseuri"`
	Web               Web    `mapstructure:"web"`
	SecureViewAppAddr string `mapstructure:"secure_view_app_addr"`
}

// Web holds the config options for the URL parameters for Web
type Web struct {
	URLParamsMapping map[string]string `mapstructure:"urlparamsmapping"`
	StaticURLParams  map[string]string `mapstructure:"staticurlparams"`
}

func (c *Config) init() {
	if c.Prefix == "" {
		c.Prefix = "app"
	}
	c.GatewaySvc = sharedconf.GetGatewaySVC(c.GatewaySvc)
}

type svc struct {
	conf   *Config
	router *chi.Mux
}

// New returns a new ocmd object
func New(m map[string]interface{}, log *zerolog.Logger) (global.Service, error) {

	conf := &Config{}
	if err := mapstructure.Decode(m, conf); err != nil {
		return nil, err
	}
	conf.init()

	r := chi.NewRouter()
	s := &svc{
		conf:   conf,
		router: r,
	}

	if err := s.routerInit(); err != nil {
		return nil, err
	}

	_ = chi.Walk(s.router, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Debug().Str("service", "approvider").Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return s, nil
}

const (
	openModeNormal = iota
	openModeWeb
)

func (s *svc) routerInit() error {
	s.router.Get("/list", s.handleList)
	s.router.Post("/new", s.handleNew)
	s.router.Post("/open", s.handleOpen(openModeNormal))
	s.router.Post("/open-with-web", s.handleOpen(openModeWeb))
	return nil
}

// Close performs cleanup.
func (s *svc) Close() error {
	return nil
}

func (s *svc) Prefix() string {
	return s.conf.Prefix
}

func (s *svc) Unprotected() []string {
	return []string{"/list"}
}

func (s *svc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.router.ServeHTTP(w, r)
	})
}

func (s *svc) handleNew(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := pool.GetGatewayServiceClient(s.conf.GatewaySvc)
	if err != nil {
		writeError(w, r, appErrorServerError, "error getting grpc gateway client", err)
		return
	}

	err = r.ParseForm()
	if err != nil {
		writeError(w, r, appErrorInvalidParameter, "parameters could not be parsed", nil)
	}

	if r.Form.Get("template") != "" {
		// TODO in the future we want to create a file out of the given template
		writeError(w, r, appErrorUnimplemented, "template is not implemented", nil)
		return
	}

	parentContainerIDStr := r.Form.Get("parent_container_id")
	if parentContainerIDStr == "" {
		writeError(w, r, appErrorInvalidParameter, "missing parent container ID", nil)
		return
	}

	parentContainerID, err := storagespace.ParseID(parentContainerIDStr)
	if err != nil {
		writeError(w, r, appErrorInvalidParameter, "invalid parent container ID", nil)
		return
	}

	filename := r.Form.Get("filename")
	if filename == "" {
		writeError(w, r, appErrorInvalidParameter, "missing filename", nil)
		return
	}

	dirPart, filePart := path.Split(filename)
	if dirPart != "" || filePart != filename {
		writeError(w, r, appErrorInvalidParameter, "the filename must not contain a path segment", nil)
		return
	}

	statParentContainerReq := &provider.StatRequest{
		Ref: &provider.Reference{
			ResourceId: &parentContainerID,
		},
	}
	parentContainer, err := client.Stat(ctx, statParentContainerReq)
	if err != nil {
		writeError(w, r, appErrorServerError, "error sending a grpc stat request", err)
		return
	}

	if parentContainer.Status.Code != rpc.Code_CODE_OK {
		writeError(w, r, appErrorNotFound, "the parent container is not accessible or does not exist", err)
		return
	}

	if parentContainer.Info.Type != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		writeError(w, r, appErrorInvalidParameter, "the parent container id does not point to a container", nil)
		return
	}

	fileRef := &provider.Reference{
		ResourceId: &parentContainerID,
		Path:       utils.MakeRelativePath(filename),
	}

	statFileReq := &provider.StatRequest{
		Ref: fileRef,
	}
	statFileRes, err := client.Stat(ctx, statFileReq)
	if err != nil {
		writeError(w, r, appErrorServerError, "failed to stat the file", err)
		return
	}

	if statFileRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
		if statFileRes.Status.Code == rpc.Code_CODE_OK {
			writeError(w, r, appErrorAlreadyExists, "the file already exists", nil)
			return
		}
		writeError(w, r, appErrorServerError, "statting the file returned unexpected status code", err)
		return
	}

	touchFileReq := &provider.TouchFileRequest{
		Ref: fileRef,
	}

	touchRes, err := client.TouchFile(ctx, touchFileReq)
	if err != nil {
		writeError(w, r, appErrorServerError, "error sending a grpc touchfile request", err)
		return
	}

	if touchRes.Status.Code != rpc.Code_CODE_OK {
		if touchRes.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
			writeError(w, r, appErrorPermissionDenied, "permission denied to create the file", nil)
			return
		}
		writeError(w, r, appErrorServerError, "touching the file failed", nil)
		return
	}

	// Stat the newly created file
	statRes, err := client.Stat(ctx, statFileReq)
	if err != nil {
		writeError(w, r, appErrorServerError, "statting the created file failed", err)
		return
	}

	if statRes.Status.Code != rpc.Code_CODE_OK {
		writeError(w, r, appErrorServerError, "statting the created file failed", nil)
		return
	}

	if statRes.Info.Type != provider.ResourceType_RESOURCE_TYPE_FILE {
		writeError(w, r, appErrorInvalidParameter, "the given file id does not point to a file", nil)
		return
	}
	fileid := storagespace.FormatResourceID(statRes.Info.Id)

	js, err := json.Marshal(
		map[string]interface{}{
			"file_id": fileid,
		},
	)
	if err != nil {
		writeError(w, r, appErrorServerError, "error marshalling JSON response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(js); err != nil {
		writeError(w, r, appErrorServerError, "error writing JSON response", err)
		return
	}
}

func (s *svc) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	client, err := pool.GetGatewayServiceClient(s.conf.GatewaySvc)
	if err != nil {
		writeError(w, r, appErrorServerError, "error getting grpc gateway client", err)
		return
	}

	listRes, err := client.ListSupportedMimeTypes(ctx, &appregistry.ListSupportedMimeTypesRequest{})
	if err != nil {
		writeError(w, r, appErrorServerError, "error listing supported mime types", err)
		return
	}
	if listRes.Status.Code != rpc.Code_CODE_OK {
		writeError(w, r, appErrorServerError, "error listing supported mime types", nil)
		return
	}

	res := buildApps(listRes.MimeTypes, r.UserAgent(), s.conf.SecureViewAppAddr)

	js, err := json.Marshal(map[string]interface{}{"mime-types": res})
	if err != nil {
		writeError(w, r, appErrorServerError, "error marshalling JSON response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(js); err != nil {
		writeError(w, r, appErrorServerError, "error writing JSON response", err)
		return
	}
}

func (s *svc) handleOpen(openMode int) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		client, err := pool.GetGatewayServiceClient(s.conf.GatewaySvc)
		if err != nil {
			writeError(w, r, appErrorServerError, "Internal error with the gateway, please try again later", err)
			return
		}

		err = r.ParseForm()
		if err != nil {
			writeError(w, r, appErrorInvalidParameter, "parameters could not be parsed", nil)
		}

		lang := r.Form.Get("lang")
		parts := strings.Split(lang, "_")
		if lang != "" && !iso6391.ValidCode(parts[0]) {
			writeError(w, r, appErrorInvalidParameter, "lang parameter does not contain a valid ISO 639-1 language code in the language tag", nil)
			return
		}

		fileID := r.Form.Get("file_id")

		if fileID == "" {
			writeError(w, r, appErrorInvalidParameter, "missing file ID", nil)
			return
		}

		resourceID, err := storagespace.ParseID(fileID)
		if err != nil {
			writeError(w, r, appErrorInvalidParameter, "invalid file ID", nil)
			return
		}

		fileRef := &provider.Reference{
			ResourceId: &resourceID,
			Path:       ".",
		}

		statRes, err := client.Stat(ctx, &provider.StatRequest{Ref: fileRef})
		if err != nil {
			writeError(w, r, appErrorServerError, "Internal error accessing the file, please try again later", err)
			return
		}

		if status := utils.ReadPlainFromOpaque(statRes.GetInfo().GetOpaque(), "status"); status == "processing" {
			writeError(w, r, appErrorTooEarly, "The requested file is not yet available, please try again later", nil)
			return
		}

		viewMode, err := getViewModeFromPublicScope(ctx)
		if err != nil {
			writeError(w, r, appErrorPermissionDenied, "permission denied to open the application", err)
			return
		}

		if viewMode == gateway.OpenInAppRequest_VIEW_MODE_INVALID {
			// we have no publicshare Role in the token scope
			// do a stat request to assemble the permissions for this user
			statRes, err := client.Stat(ctx, &provider.StatRequest{Ref: fileRef})
			if err != nil {
				writeError(w, r, appErrorServerError, "Internal error accessing the file, please try again later", err)
				return
			}

			if statRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
				writeError(w, r, appErrorNotFound, "file does not exist", nil)
				return
			} else if statRes.Status.Code != rpc.Code_CODE_OK {
				writeError(w, r, appErrorServerError, "failed to stat the file", nil)
				return
			}

			if statRes.Info.Type != provider.ResourceType_RESOURCE_TYPE_FILE {
				writeError(w, r, appErrorInvalidParameter, "the given file id does not point to a file", nil)
				return
			}

			// Calculate the view mode from the resource permissions
			viewMode = getViewMode(statRes.Info, r.Form.Get("view_mode"))
			if viewMode == gateway.OpenInAppRequest_VIEW_MODE_INVALID {
				writeError(w, r, appErrorInvalidParameter, "invalid view mode", err)
				return
			}
		}

		openReq := gateway.OpenInAppRequest{
			Ref:      fileRef,
			ViewMode: viewMode,
			App:      r.Form.Get("app_name"),
			Opaque:   utils.AppendPlainToOpaque(nil, "lang", lang),
		}

		templateID := r.Form.Get("template_id")
		if templateID != "" {
			openReq.Opaque = utils.AppendPlainToOpaque(openReq.Opaque, "template", templateID)
		}
		openRes, err := client.OpenInApp(ctx, &openReq)
		if err != nil {
			writeError(w, r, appErrorServerError,
				"Error contacting the requested application, please use a different one or try again later", err)
			return
		}
		if openRes.Status.Code != rpc.Code_CODE_OK {
			if openRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
				writeError(w, r, appErrorNotFound, openRes.Status.Message, nil)
				return
			}
			writeError(w, r, appErrorServerError, openRes.Status.Message,
				status.NewErrorFromCode(openRes.Status.Code, "error calling OpenInApp"))
			return
		}

		var payload interface{}

		switch openMode {
		case openModeNormal:
			payload = openRes.AppUrl

		case openModeWeb:
			payload, err = newOpenInWebResponse(s.conf.WebBaseURI, s.conf.Web.URLParamsMapping, s.conf.Web.StaticURLParams, fileID, r.Form.Get("app_name"), r.Form.Get("view_mode"))
			if err != nil {
				writeError(w, r, appErrorServerError, "Internal error",
					errors.Wrap(err, "error building OpenInWeb response"))
				return
			}

		default:
			writeError(w, r, appErrorServerError, "Internal error with the open mode",
				errors.New("unknown open mode"))
			return

		}

		js, err := json.Marshal(payload)
		if err != nil {
			writeError(w, r, appErrorServerError, "Internal error with JSON payload",
				errors.Wrap(err, "error marshalling JSON response"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err = w.Write(js); err != nil {
			writeError(w, r, appErrorServerError, "Internal error with JSON payload",
				errors.Wrap(err, "error writing JSON response"))
			return
		}
	}
}

type openInWebResponse struct {
	URI string `json:"uri"`
}

func newOpenInWebResponse(baseURI string, params, staticParams map[string]string, fileID, appName, viewMode string) (openInWebResponse, error) {

	uri, err := url.Parse(baseURI)
	if err != nil {
		return openInWebResponse{}, err
	}

	query := uri.Query()

	for key, val := range params {

		switch val {
		case "fileid":
			if fileID != "" {
				query.Add(key, fileID)
			}
		case "appname":
			if appName != "" {
				query.Add(key, appName)
			}
		case "viewmode":
			if viewMode != "" {
				query.Add(key, viewMode)
			}
		default:
			return openInWebResponse{}, errors.New("unknown parameter mapper")
		}

	}

	for key, val := range staticParams {
		query.Add(key, val)
	}

	uri.RawQuery = query.Encode()

	return openInWebResponse{URI: uri.String()}, nil
}

// MimeTypeInfo wraps the appregistry.MimeTypeInfo to change the app providers to ProviderInfos with a secure view flag
type MimeTypeInfo struct {
	appregistry.MimeTypeInfo
	AppProviders []*ProviderInfo `json:"app_providers"`
}

// ProviderInfo wraps the appregistry.ProviderInfo to add a secure view flag
type ProviderInfo struct {
	appregistry.ProviderInfo
	// TODO make this part of the CS3 provider info
	SecureView bool   `json:"secure_view"`
	TargetExt  string `json:"target_ext,omitempty"`
}

// buildApps rewrites the mime type info to only include apps that
// * have a name
// * can be called by the user agent, eg Desktop-only
//
// it also
// * wraps the provider info to be able to add a secure view flag
// * adds a secure view flag if the address matches the secure view app address and
// * removes the address from the provider info to not expose internal addresses
func buildApps(mimeTypes []*appregistry.MimeTypeInfo, userAgent, secureViewAppAddr string) []*MimeTypeInfo {
	ua := ua.Parse(userAgent)
	res := []*MimeTypeInfo{}
	for _, m := range mimeTypes {
		apps := []*ProviderInfo{}
		for _, p := range m.AppProviders {
			ep := &ProviderInfo{}
			proto.Merge(&ep.ProviderInfo, p)
			if p.Address == secureViewAppAddr {
				ep.SecureView = true
			}
			p.Address = "" // address is internal only and not needed in the client
			// apps are called by name, so if it has no name it cannot be called and should not be advertised
			// also filter Desktop-only apps if ua is not Desktop
			if p.Name != "" && (ua.Desktop || !p.DesktopOnly) {
				apps = append(apps, ep)
			}
		}
		if len(apps) > 0 {
			mt := &MimeTypeInfo{}
			addTemplateInfo(m, apps)
			proto.Merge(&mt.MimeTypeInfo, m)
			mt.AppProviders = apps
			res = append(res, mt)
		}
	}
	return res
}

func getViewMode(res *provider.ResourceInfo, vm string) gateway.OpenInAppRequest_ViewMode {
	if vm != "" {
		return utils.GetViewMode(vm)
	}

	var viewMode gateway.OpenInAppRequest_ViewMode
	canEdit := res.PermissionSet.InitiateFileUpload
	canView := res.PermissionSet.InitiateFileDownload

	switch {
	case canEdit && canView:
		viewMode = gateway.OpenInAppRequest_VIEW_MODE_READ_WRITE
	case canView:
		viewMode = gateway.OpenInAppRequest_VIEW_MODE_READ_ONLY
	default:
		viewMode = gateway.OpenInAppRequest_VIEW_MODE_INVALID
	}
	return viewMode
}

// try to get the view mode from a publicshare scope
func getViewModeFromPublicScope(ctx context.Context) (gateway.OpenInAppRequest_ViewMode, error) {
	scopes, ok := ctxpkg.ContextGetScopes(ctx)
	if ok {
		for key, scope := range scopes {
			if strings.HasPrefix(key, "publicshare:") {
				switch scope.GetRole() {
				case providerv1beta1.Role_ROLE_VIEWER:
					return gateway.OpenInAppRequest_VIEW_MODE_VIEW_ONLY, nil
				case providerv1beta1.Role_ROLE_EDITOR:
					return gateway.OpenInAppRequest_VIEW_MODE_READ_WRITE, nil
				default:
					return gateway.OpenInAppRequest_VIEW_MODE_INVALID, errors.New("invalid view mode in publicshare scope")
				}
			}
		}
	}
	return gateway.OpenInAppRequest_VIEW_MODE_INVALID, nil
}
