package theme

import (
	"encoding/json"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	permissionsapi "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/pkg/errors"
	"github.com/spf13/afero"

	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"

	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/pkg/x/path/filepathx"
)

// ServiceOptions defines the options to configure the Service.
type ServiceOptions struct {
	themeFS         *fsx.FallbackFS
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// WithThemeFS sets the theme filesystem.
func (o ServiceOptions) WithThemeFS(fSys *fsx.FallbackFS) ServiceOptions {
	o.themeFS = fSys
	return o
}

// WithGatewaySelector sets the gateway selector.
func (o ServiceOptions) WithGatewaySelector(gws pool.Selectable[gateway.GatewayAPIClient]) ServiceOptions {
	o.gatewaySelector = gws
	return o
}

// validate validates the input parameters.
func (o ServiceOptions) validate() error {
	if o.themeFS == nil {
		return errors.New("themeFS is required")
	}

	if o.gatewaySelector == nil {
		return errors.New("gatewaySelector is required")
	}

	return nil
}

// Service defines the http service.
type Service struct {
	themeFS         *fsx.FallbackFS
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// NewService initializes a new Service.
func NewService(options ServiceOptions) (Service, error) {
	if err := options.validate(); err != nil {
		return Service{}, err
	}

	return Service(options), nil
}

// Get renders the theme, the theme is a merge of the default theme, the base theme, and the branding theme.
func (s Service) Get(w http.ResponseWriter, r *http.Request) {
	// there is no guarantee that the theme exists, its optional; therefore, we ignore the error
	baseTheme, _ := LoadKV(s.themeFS, filepathx.JailJoin(r.PathValue("id"), _themeFileName))

	// there is no guarantee that the theme exists, its optional; therefore, we ignore the error here too
	brandingTheme, _ := LoadKV(s.themeFS, filepathx.JailJoin(_brandingRoot, _themeFileName))

	// merge the themes, the order is important, the last one wins and overrides the previous ones
	// themeDefaults: contains all the default values, this is guaranteed to exist
	// baseTheme: contains the base theme from the theme fs, there is no guarantee that it exists
	// brandingTheme: contains the branding theme from the theme fs, there is no guarantee that it exists
	// mergedTheme = themeDefaults < baseTheme < brandingTheme
	mergedTheme, err := MergeKV(themeDefaults, baseTheme, brandingTheme)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(mergedTheme)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// LogoUpload implements the endpoint to upload a custom logo for the OpenCloud instance.
func (s Service) LogoUpload(w http.ResponseWriter, r *http.Request) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user := revactx.ContextMustGetUser(r.Context())
	rsp, err := gatewayClient.CheckPermission(r.Context(), &permissionsapi.CheckPermissionRequest{
		Permission: "Logo.Write",
		SubjectRef: &permissionsapi.SubjectReference{
			Spec: &permissionsapi.SubjectReference_UserId{
				UserId: user.GetId(),
			},
		},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if rsp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	file, fileHeader, err := r.FormFile("logo")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			w.WriteHeader(http.StatusBadRequest)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	if !isFiletypePermitted(fileHeader.Filename, fileHeader.Header.Get("Content-Type")) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fp := filepathx.JailJoin(_brandingRoot, fileHeader.Filename)
	err = afero.WriteReader(s.themeFS, fp, file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = UpdateKV(s.themeFS, filepathx.JailJoin(_brandingRoot, _themeFileName), KV{
		"common.logo":               filepathx.JailJoin("themes", fp),
		"clients.web.defaults.logo": filepathx.JailJoin("themes", fp),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// LogoReset implements the endpoint to reset the instance logo.
// The config will be changed back to use the embedded logo asset.
func (s Service) LogoReset(w http.ResponseWriter, r *http.Request) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user := revactx.ContextMustGetUser(r.Context())
	rsp, err := gatewayClient.CheckPermission(r.Context(), &permissionsapi.CheckPermissionRequest{
		Permission: "Logo.Write",
		SubjectRef: &permissionsapi.SubjectReference{
			Spec: &permissionsapi.SubjectReference_UserId{
				UserId: user.GetId(),
			},
		},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if rsp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = UpdateKV(s.themeFS, filepathx.JailJoin(_brandingRoot, _themeFileName), KV{
		"common.logo":               nil,
		"clients.web.defaults.logo": nil,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
