package service

import (
	"context"
	"errors"
	"net/url"
	"path"
	"strconv"
	"strings"

	appproviderv1beta1 "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/helpers"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/wopisrc"
)

// NewHandler creates a new grpc service implementing the OpenInApp interface
func NewHandler(opts ...Option) (*Service, func(), error) {
	teardown := func() {
		/* this is required as a argument for the return value to satisfy the interface */
		/* in case you are wondering about the necessity of this comment, sonarcloud is asking for it */
	}
	options := newOptions(opts...)

	gatewaySelector := options.GatewaySelector
	var err error
	if gatewaySelector == nil {
		gatewaySelector, err = pool.GatewaySelector(options.Config.CS3Api.Gateway.Name)
		if err != nil {
			return nil, teardown, err
		}
	}

	if options.AppURLs == nil {
		return nil, teardown, errors.New("AppURLs option is required")
	}

	return &Service{
		id:              options.Config.GRPC.Namespace + "." + options.Config.Service.Name + "." + options.Config.App.Name,
		appURLs:         options.AppURLs,
		logger:          options.Logger,
		config:          options.Config,
		gatewaySelector: gatewaySelector,
		store:           options.Store,
	}, teardown, nil
}

// Service implements the OpenInApp interface
type Service struct {
	id              string
	appURLs         *helpers.AppURLs
	logger          log.Logger
	config          *config.Config
	gatewaySelector pool.Selectable[gatewayv1beta1.GatewayAPIClient]
	store           microstore.Store
}

// OpenInApp will implement the OpenInApp interface of the app provider
func (s *Service) OpenInApp(
	ctx context.Context,
	req *appproviderv1beta1.OpenInAppRequest,
) (*appproviderv1beta1.OpenInAppResponse, error) {

	// get the current user
	var user *userv1beta1.User = nil
	meReq := &gatewayv1beta1.WhoAmIRequest{
		Token: req.GetAccessToken(),
	}
	gwc, err := s.gatewaySelector.Next()
	if err != nil {
		s.logger.Error().Err(err).Msg("OpenInApp: could not select a gateway client")
		return nil, err
	}

	meResp, err := gwc.WhoAmI(ctx, meReq)
	if err == nil {
		if meResp.GetStatus().GetCode() == rpcv1beta1.Code_CODE_OK {
			user = meResp.GetUser()
		}
	}

	// required for the response, it will be used also for logs
	providerFileRef := providerv1beta1.Reference{
		ResourceId: req.GetResourceInfo().GetId(),
		Path:       ".",
	}

	logger := s.logger.With().
		Str("FileReference", providerFileRef.String()).
		Str("ViewMode", req.GetViewMode().String()).
		Str("Requester", user.GetId().String()).
		Logger()

	// get the file extension to use the right wopi app url
	fileExt := path.Ext(req.GetResourceInfo().GetPath())

	// get the appURL we need to use
	appURL := s.getAppUrl(fileExt, req.GetViewMode())
	if appURL == "" {
		logger.Error().Msg("OpenInApp: neither edit nor view app URL found")
		return nil, errors.New("neither edit nor view app URL found")
	}

	// append the parameters we need
	appURL, err = s.addQueryToURL(appURL, req)
	if err != nil {
		logger.Error().Err(err).Msg("OpenInApp: error parsing appUrl")
		return &appproviderv1beta1.OpenInAppResponse{
			Status: &rpcv1beta1.Status{
				Code:    rpcv1beta1.Code_CODE_INVALID_ARGUMENT,
				Message: "OpenInApp: error parsing appUrl",
			},
		}, nil
	}

	// create the wopiContext and generate the token
	wopiContext := middleware.WopiContext{
		AccessToken:   req.GetAccessToken(), // it will be encrypted
		ViewOnlyToken: utils.ReadPlainFromOpaque(req.GetOpaque(), "viewOnlyToken"),
		FileReference: &providerFileRef,
		ViewMode:      req.GetViewMode(),
	}

	if templateID := utils.ReadPlainFromOpaque(req.GetOpaque(), "template"); templateID != "" {
		// we can ignore the error here, as we are sure that the templateID is not empty
		templateRes, _ := storagespace.ParseID(templateID)
		// we need to have at least both opaqueID and spaceID set
		if templateRes.GetOpaqueId() == "" || templateRes.GetSpaceId() == "" {
			logger.Error().Err(err).Msg("OpenInApp: error parsing templateID")
			return &appproviderv1beta1.OpenInAppResponse{
				Status: &rpcv1beta1.Status{
					Code:    rpcv1beta1.Code_CODE_INVALID_ARGUMENT,
					Message: "OpenInApp: error parsing templateID",
				},
			}, nil
		}
		wopiContext.TemplateReference = &providerv1beta1.Reference{
			ResourceId: &templateRes,
			Path:       ".",
		}
	}

	accessToken, accessExpiration, err := middleware.GenerateWopiToken(wopiContext, s.config, s.store)
	if err != nil {
		logger.Error().Err(err).Msg("OpenInApp: error generating the token")
		return &appproviderv1beta1.OpenInAppResponse{
			Status: &rpcv1beta1.Status{Code: rpcv1beta1.Code_CODE_INTERNAL},
		}, err
	}

	logger.Debug().Msg("OpenInApp: success")

	return &appproviderv1beta1.OpenInAppResponse{
		Status: &rpcv1beta1.Status{Code: rpcv1beta1.Code_CODE_OK},
		AppUrl: &appproviderv1beta1.OpenInAppURL{
			AppUrl: appURL,
			Method: "POST",
			FormParameters: map[string]string{
				// these parameters will be passed to the web server by the app provider application
				"access_token": accessToken,
				// milliseconds since Jan 1, 1970 UTC as required in https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/concepts#access_token_ttl
				//"access_token_ttl": strconv.FormatInt(claims.ExpiresAt.UnixMilli(), 10),
				"access_token_ttl": strconv.FormatInt(accessExpiration, 10),
			},
		},
	}, nil
}

// getAppUrl will get the appURL that should be used based on the extension
// and the provided view mode.
// "view" urls will be chosen first, then if the view mode is "read/write",
// "edit" urls will be prioritized. Note that "view" url might be returned for
// "read/write" view mode if no "edit" url is found.
func (s *Service) getAppUrl(fileExt string, viewMode appproviderv1beta1.ViewMode) string {
	// prioritize view action if possible
	appURL := s.appURLs.GetAppURLFor("view", fileExt)

	if strings.ToLower(s.config.App.Product) == "collabora" {
		// collabora provides only one action per extension. usual options
		// are "view" (checked above), "edit" or "view_comment" (this last one
		// is exclusive of collabora)
		if appURL == "" {
			if editURL := s.appURLs.GetAppURLFor("edit", fileExt); editURL != "" {
				return editURL
			}
			if commentURL := s.appURLs.GetAppURLFor("view_comment", fileExt); commentURL != "" {
				return commentURL
			}
		}
	} else {
		// If not collabora, there might be an edit action for the extension.
		// If read/write mode has been requested, prioritize edit action.
		if viewMode == appproviderv1beta1.ViewMode_VIEW_MODE_READ_WRITE {
			if editAppURL := s.appURLs.GetAppURLFor("edit", fileExt); editAppURL != "" {
				appURL = editAppURL
			}
		}
	}

	return appURL
}

// addQueryToURL will add specific query parameters to the baseURL. These
// parameters are:
// * "WOPISrc" pointing to the requested resource in the OpenInAppRequest
// * "dchat" to disable the chat, based on configuration
// * "lang" (WOPI app dependent) with the language in the request. "lang"
// for collabora, "ui" for onlyoffice and "UI_LLCC" for the rest
func (s *Service) addQueryToURL(baseURL string, req *appproviderv1beta1.OpenInAppRequest) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// build a urlsafe and stable file reference that can be used for proxy routing,
	// so that all sessions on one file end on the same office server
	fileRef := helpers.HashResourceId(req.GetResourceInfo().GetId())

	wopiSrcURL, err := wopisrc.GenerateWopiSrc(fileRef, s.config)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Add("WOPISrc", wopiSrcURL.String())

	if s.config.Wopi.DisableChat {
		q.Add("dchat", "1")
	}

	lang := utils.ReadPlainFromOpaque(req.GetOpaque(), "lang")

	// @TODO: this is a temporary solution until we figure out how to send these from oc web
	switch lang {
	case "bg":
		lang = "bg-BG"
	case "cs":
		lang = "cs-CZ"
	case "de":
		lang = "de-DE"
	case "en":
		lang = "en-GB"
	case "es":
		lang = "es-ES"
	case "fr":
		lang = "fr-FR"
	case "gl":
		lang = "gl-ES"
	case "it":
		lang = "it-IT"
	case "nl":
		lang = "nl-NL"
	case "ko":
		lang = "ko-KR"
	case "sq":
		lang = "sq-AL"
	case "sv":
		lang = "sv-SE"
	case "tr":
		lang = "tr-TR"
	case "zh":
		lang = "zh-CN"
	}

	if lang != "" {
		switch strings.ToLower(s.config.App.Product) {
		case "collabora":
			q.Add("lang", lang)
		case "onlyoffice":
			q.Add("ui", lang)
		default:
			q.Add("UI_LLCC", lang)
		}
	}

	if strings.ToLower(s.config.App.Product) == "collabora" {
		q.Add("closebutton", "false")
	}

	qs := q.Encode()
	u.RawQuery = qs

	return u.String(), nil
}
