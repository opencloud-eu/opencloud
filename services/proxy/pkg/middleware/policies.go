package middleware

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	pMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/policies/v0"
	pService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/policies/v0"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/net"
)

type (
	// RequestDenied struct for OdataErrorMain
	RequestDenied struct {
		Error RequestDeniedError `json:"error"`
	}

	// RequestDeniedError struct for RequestDenied
	RequestDeniedError struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		// The structure of this object is service-specific
		Innererror map[string]interface{} `json:"innererror,omitempty"`
	}
)

const DeniedMessage = "Operation denied due to security policies"

// Policies verifies if a request is granted or not.
func Policies(qs string, opts ...Option) func(next http.Handler) http.Handler {
	options := newOptions(opts...)
	logger := options.Logger
	tracer := getTraceProvider(options).Tracer("proxy.middleware.policies")
	gatewaySelector := options.RevaGatewaySelector
	policiesProviderClient := options.PoliciesProviderService

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path), trace.WithSpanKind(trace.SpanKindServer))
			r = r.WithContext(ctx)
			defer span.End()
			if qs == "" {
				span.End()
				next.ServeHTTP(w, r)
				return
			}

			req := &pService.EvaluateRequest{
				Query: qs,
				Environment: &pMessage.Environment{
					Request: &pMessage.Request{
						Method: r.Method,
						Path:   r.URL.Path,
					},
					Stage: pMessage.Stage_STAGE_HTTP,
				},
			}

			resource := &pMessage.Resource{}

			// tus
			meta := tusd.ParseMetadataHeader(r.Header.Get(net.HeaderUploadMetadata))
			resource.Name = meta["filename"]

			// name is part of the request path
			if resource.Name == "" && filepath.Ext(r.URL.Path) != "" {
				resource.Name = filepath.Base(r.URL.Path)
			}

			// no resource info in path, stat the resource and try to obtain the file information.
			// this should only be used as last bastion, every request goes through the proxy and doing stats is expensive!
			// needed for:
			// - if a single resource is shared -> the url only contains the resourceID (spaceRef)
			if resource.Name == "" && filepath.Ext(r.URL.Path) == "" && r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/remote.php/dav/spaces") {
				client, err := gatewaySelector.Next()
				if err != nil {
					logger.Err(err).Msg("error selecting next gateway client")
					RenderError(w, r, req, http.StatusForbidden, DeniedMessage)
					return
				}

				resourceID, err := storagespace.ParseID(strings.TrimPrefix(r.URL.Path, "/remote.php/dav/spaces/"))
				if err != nil {
					logger.Debug().Err(err).Msg("error parsing the resourceId")
					RenderError(w, r, req, http.StatusForbidden, DeniedMessage)
					return
				}

				if resourceID.StorageId == "" && resourceID.SpaceId == utils.ShareStorageSpaceID {
					resourceID.StorageId = utils.ShareStorageProviderID
				}

				token := r.Header.Get(revactx.TokenHeader)
				ctx := metadata.AppendToOutgoingContext(r.Context(), revactx.TokenHeader, token)
				sRes, err := client.Stat(ctx, &provider.StatRequest{
					Ref: &provider.Reference{
						ResourceId: &resourceID,
					},
				})

				resource.Name = sRes.GetInfo().GetName()
			}

			req.Environment.Resource = resource

			if user, ok := revactx.ContextGetUser(r.Context()); ok {
				req.Environment.User = &pMessage.User{
					Id: &pMessage.User_ID{
						OpaqueId: user.GetId().GetOpaqueId(),
					},
					Username:    user.GetUsername(),
					Mail:        user.GetMail(),
					DisplayName: user.GetDisplayName(),
					Groups:      user.GetGroups(),
				}
			}

			rsp, err := policiesProviderClient.Evaluate(r.Context(), req)
			if err != nil {
				logger.Err(err).Msg("error evaluating request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if !rsp.Result {
				RenderError(w, r, req, http.StatusForbidden, DeniedMessage)
				return
			}
			span.End()
			next.ServeHTTP(w, r)
		})
	}
}

// RenderError writes a Policies ErrorObject to the response writer
func RenderError(w http.ResponseWriter, r *http.Request, evaluateReq *pService.EvaluateRequest, status int, msg string) {
	filename := evaluateReq.Environment.GetResource().GetName()
	if filename == "" {
		filename = path.Base(evaluateReq.Environment.GetRequest().GetPath())
	}

	innererror := map[string]interface{}{
		"date": time.Now().UTC().Format(time.RFC3339),
	}

	innererror["request-id"] = middleware.GetReqID(r.Context())
	innererror["method"] = evaluateReq.Environment.GetRequest().GetMethod()
	innererror["filename"] = filename
	innererror["path"] = evaluateReq.Environment.GetRequest().GetPath()

	resp := &RequestDenied{
		Error: RequestDeniedError{
			Code:       "deniedByPolicy",
			Message:    msg,
			Innererror: innererror,
		},
	}
	render.Status(r, status)
	render.JSON(w, r, resp)
}
