package svc

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/jellydator/ttlcache/v3"
	"github.com/nats-io/nats.go/jetstream"
	"go-micro.dev/v4/client"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/pkg/keycloak"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
)

// Permissions is the interface used to access the permissions service
type Permissions interface {
	ListPermissions(ctx context.Context, req *settingssvc.ListPermissionsRequest, opts ...client.CallOption) (*settingssvc.ListPermissionsResponse, error)
	GetPermissionByID(ctx context.Context, request *settingssvc.GetPermissionByIDRequest, opts ...client.CallOption) (*settingssvc.GetPermissionByIDResponse, error)
	ListPermissionsByResource(ctx context.Context, in *settingssvc.ListPermissionsByResourceRequest, opts ...client.CallOption) (*settingssvc.ListPermissionsByResourceResponse, error)
}

// HTTPClient is the subset of the http.Client that is being used to interact with the download gateway
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// GetGatewayServiceClientFunc is a callback used to pass in a mock during testing
type GetGatewayServiceClientFunc func() (gateway.GatewayAPIClient, error)

// RoleService is the interface used to access the role service
type RoleService interface {
	ListRoles(ctx context.Context, in *settingssvc.ListBundlesRequest, opts ...client.CallOption) (*settingssvc.ListBundlesResponse, error)
	ListRoleAssignments(ctx context.Context, in *settingssvc.ListRoleAssignmentsRequest, opts ...client.CallOption) (*settingssvc.ListRoleAssignmentsResponse, error)
	ListRoleAssignmentsFiltered(ctx context.Context, in *settingssvc.ListRoleAssignmentsFilteredRequest, opts ...client.CallOption) (*settingssvc.ListRoleAssignmentsResponse, error)
	AssignRoleToUser(ctx context.Context, in *settingssvc.AssignRoleToUserRequest, opts ...client.CallOption) (*settingssvc.AssignRoleToUserResponse, error)
	RemoveRoleFromUser(ctx context.Context, in *settingssvc.RemoveRoleFromUserRequest, opts ...client.CallOption) (*emptypb.Empty, error)
}

// Graph defines implements the business logic for Service.
type Graph struct {
	BaseGraphService
	mux                      *chi.Mux
	identityBackend          identity.Backend
	identityEducationBackend identity.EducationBackend
	roleService              RoleService
	permissionsService       Permissions
	valueService             settingssvc.ValueService
	specialDriveItemsCache   *ttlcache.Cache[string, interface{}]
	eventsPublisher          events.Publisher
	eventsConsumer           events.Consumer
	searchService            searchsvc.SearchProviderService
	keycloakClient           keycloak.Client
	historyClient            ehsvc.EventHistoryService
	traceProvider            trace.TracerProvider
	natskv                   jetstream.KeyValue
}

// ServeHTTP implements the Service interface.
func (g Graph) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// There was a number of issues with the chi router and parameters with
	// slashes/percentage/other characters that didn't get properly escaped.
	// This is a workaround to fix this. Also, we're not the only ones who have
	// tried to fix this, as seen in this issue:
	// https://github.com/go-chi/chi/issues/641#issuecomment-883156692
	r.URL.RawPath = r.URL.EscapedPath()

	g.mux.ServeHTTP(w, r)
}

func (g Graph) publishEvent(ctx context.Context, ev interface{}) {
	if g.eventsPublisher != nil {
		if err := events.Publish(ctx, g.eventsPublisher, ev); err != nil {
			g.logger.Error().
				Err(err).
				Msg("could not publish user created event")
		}
	}
}

func (g Graph) getWebDavBaseURL() (*url.URL, error) {
	webDavBaseURL, err := url.Parse(g.config.Spaces.WebDavBase)
	if err != nil {
		return nil, err
	}
	webDavBaseURL.Path = path.Join(webDavBaseURL.Path, g.config.Spaces.WebDavPath)
	return webDavBaseURL, nil
}

// ListResponse is used for proper marshalling of Graph list responses
type ListResponse struct {
	Value interface{} `json:"value,omitempty"`
}

const (
	// ReadmeSpecialFolderName for the drive specialFolder property
	ReadmeSpecialFolderName = "readme"
	// SpaceImageSpecialFolderName for the drive specialFolder property
	SpaceImageSpecialFolderName = "image"
)

type APIVersion int

const (
	// APIVersion_1 represents the first version of the API.
	APIVersion_1 APIVersion = iota + 1

	// APIVersion_1_Beta_1 refers to the beta version of the API.
	// It is typically used for testing purposes and may have more
	// inconsistencies and bugs than the stable version as it is
	// still in the testing phase, use it with caution.
	APIVersion_1_Beta_1
)

// TODO might be different for /education/users vs /users
func (g Graph) parseMemberRef(ref string) (string, string, error) {
	memberURL, err := url.ParseRequestURI(ref)
	if err != nil {
		return "", "", err
	}
	segments := strings.Split(memberURL.Path, "/")
	if len(segments) < 2 {
		return "", "", errors.New("invalid member reference")
	}
	id := segments[len(segments)-1]
	memberType := segments[len(segments)-2]
	return memberType, id, nil
}

func parseIDParam(r *http.Request, param string) (storageprovider.ResourceId, error) {
	driveID, err := url.PathUnescape(chi.URLParam(r, param))
	if err != nil {
		return storageprovider.ResourceId{}, errorcode.New(errorcode.InvalidRequest, err.Error())
	}

	id, err := storagespace.ParseID(driveID)
	if err != nil {
		return storageprovider.ResourceId{}, errorcode.New(errorcode.InvalidRequest, err.Error())
	}
	return id, nil
}
