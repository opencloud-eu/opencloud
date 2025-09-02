package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

// CreateHome provides a middleware which sends a CreateHome request to the reva gateway
func CreateHome(optionSetters ...Option) func(next http.Handler) http.Handler {
	options := newOptions(optionSetters...)
	logger := options.Logger
	tracer := getTraceProvider(options).Tracer("proxy.middleware.create_home")

	return func(next http.Handler) http.Handler {
		return &createHome{
			next:                next,
			logger:              logger,
			tracer:              tracer,
			revaGatewaySelector: options.RevaGatewaySelector,
			roleQuotas:          options.RoleQuotas,
		}
	}
}

type createHome struct {
	next                http.Handler
	logger              log.Logger
	tracer              trace.Tracer
	revaGatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	roleQuotas          map[string]uint64
}

func (m createHome) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, span := m.tracer.Start(req.Context(), fmt.Sprintf("%s %s", req.Method, req.URL.Path), trace.WithSpanKind(trace.SpanKindServer))
	req = req.WithContext(ctx)
	defer span.End()
	if !m.shouldServe(req) {
		span.End()
		m.next.ServeHTTP(w, req)
		return
	}

	token := req.Header.Get(revactx.TokenHeader)

	// we need to pass the token to authenticate the CreateHome request.
	//ctx := tokenpkg.ContextSetToken(r.Context(), token)
	ctx = metadata.AppendToOutgoingContext(req.Context(), revactx.TokenHeader, token)

	createHomeReq := &provider.CreateHomeRequest{}
	u, ok := revactx.ContextGetUser(ctx)
	if ok {
		roleIDs, err := m.getUserRoles(u)
		if err != nil {
			m.logger.Error().Err(err).Str("userid", u.Id.OpaqueId).Msg("failed to get roles for user")
			errorcode.GeneralException.Render(w, req, http.StatusInternalServerError, "Unauthorized")
			return
		}
		if limit, hasLimit := m.checkRoleQuotaLimit(roleIDs); hasLimit {
			createHomeReq.Opaque = utils.AppendPlainToOpaque(nil, "quota", strconv.FormatUint(limit, 10))
		}
	}

	client, err := m.revaGatewaySelector.Next()
	if err != nil {
		m.logger.Err(err).Msg("error selecting next gateway client")
	} else {
		createHomeRes, err := client.CreateHome(ctx, createHomeReq)
		if err != nil {
			m.logger.Err(err).Msg("error calling CreateHome")
		} else if createHomeRes.Status.Code != rpc.Code_CODE_OK {
			err := status.NewErrorFromCode(createHomeRes.Status.Code, "gateway")
			if createHomeRes.Status.Code != rpc.Code_CODE_ALREADY_EXISTS {
				m.logger.Err(err).Msg("error when calling Createhome")
			}
		}
	}
	span.End()
	m.next.ServeHTTP(w, req)
}

func (m createHome) shouldServe(req *http.Request) bool {
	return req.Header.Get(revactx.TokenHeader) != ""
}

func (m createHome) getUserRoles(user *userv1beta1.User) ([]string, error) {
	var roleIDs []string
	if err := utils.ReadJSONFromOpaque(user.Opaque, "roles", &roleIDs); err != nil {
		return nil, err
	}

	tmp := make(map[string]struct{})
	for _, id := range roleIDs {
		tmp[id] = struct{}{}
	}

	dedup := make([]string, 0, len(tmp))
	for k := range tmp {
		dedup = append(dedup, k)
	}
	return dedup, nil
}

func (m createHome) checkRoleQuotaLimit(roleIDs []string) (uint64, bool) {
	if len(roleIDs) == 0 {
		return 0, false
	}
	id := roleIDs[0] // At the moment a user can only have one role.
	quota, ok := m.roleQuotas[id]
	return quota, ok
}
