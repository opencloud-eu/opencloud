package groupware

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/cors"
	"github.com/opencloud-eu/opencloud/pkg/jmaptest"
	opencloudmiddleware "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/config"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/logging"
	groupwaremiddleware "github.com/opencloud-eu/opencloud/services/groupware/pkg/middleware"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func init() {
	chi.RegisterMethod("REPORT")
}

type GroupwareTest struct {
	BaseURL string
	Users   []jmaptest.User
}

func newGroupwareTest(t *testing.T) (GroupwareTest, error) {
	require := require.New(t)
	s, err := jmaptest.NewStalwartTest(t)
	t.Cleanup(func() { s.Close() })
	require.NoError(err)

	root := ""
	{
		const charset = "abcdefghijklmnopqrstuvwxyz"
		l := 4 + rand.Intn(27)
		var sb strings.Builder
		sb.Grow(l)
		for range l {
			r := rand.Intn(len(charset))
			sb.WriteByte(charset[r])
		}
		root = sb.String()
	}

	httpPort, err := jmaptest.FreeLocalhostPort()
	require.NoError(err)

	config := &config.Config{
		Service: config.Service{
			Name: "TestGroupware",
		},
		Log: &config.Log{
			Level:  "trace",
			Pretty: true,
			Color:  false,
		},
		Mail: config.Mail{
			Master: config.MailMasterAuth{
				Username: s.MasterUsername,
				Password: s.MasterPassword,
			},
			BaseUrl: s.JmapBaseUrl.String(),
			SessionCache: config.MailSessionCache{
				MaxCapacity: 1,
			},
		},
		HTTP: config.HTTP{
			Addr:     fmt.Sprintf("127.0.0.1:%d", httpPort),
			Insecure: true,
			Root:     "/" + root,
			TLS: shared.HTTPServiceTLS{
				Enabled: false,
			},
			Namespace:             "eu.opencloud.web",
			OpenCloudPublicURL:    fmt.Sprintf("http://127.0.0.1:%d", httpPort),
			SendDurationsResponse: true,
			CORS: config.CORS{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "REPORT"},
				AllowedHeaders:   []string{"Authorization", "Origin", "Content-Type", "Accept", "X-Requested-With", "X-Request-Id", "Trace-Id", "Cache-Control"},
				AllowCredentials: true,
			},
			TraceRequests:            true,
			TraceMaxRequestBodySize:  8 * 1024,
			TraceResponses:           true,
			TraceMaxResponseBodySize: 8 * 1024,
		},
		TokenManager: &config.TokenManager{
			JWTSecret: "some-opencloud-jwt-secret",
		},
	}

	logger := logging.Configure("TestGroupware", config.Log)
	prom := prometheus.NewRegistry()
	m := chi.NewMux()

	gw, err := NewGroupwareUsingJmapClient(s.Client, config, &logger, m, prom)
	require.NoError(err)

	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			username, password, ok := r.BasicAuth()
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if match, ok := structs.First(s.Users, func(u jmaptest.User) bool {
				return u.Name == username
			}); !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			} else if match.Password != password {
				w.WriteHeader(http.StatusUnauthorized)
				return
			} else {
				u := &userv1beta1.User{
					Username: match.Email,
					Mail:     match.Email,
				}
				ctx = revactx.ContextSetUser(ctx, u)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	m.Use(
		opencloudmiddleware.Cors(
			cors.Logger(logger),
			cors.AllowedOrigins(config.HTTP.CORS.AllowedOrigins),
			cors.AllowedMethods(config.HTTP.CORS.AllowedMethods),
			cors.AllowedHeaders(config.HTTP.CORS.AllowedHeaders),
			cors.AllowCredentials(config.HTTP.CORS.AllowCredentials),
		),
		groupwaremiddleware.GroupwareLogger(logger),
		auth,
	)
	m.Route(config.HTTP.Root, gw.Route)

	baseURL := ""
	{
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", httpPort))
		require.NoError(err)
		ts := &httptest.Server{
			Listener: l,
			Config: &http.Server{
				Handler: gw,
			},
		}
		ts.Start()
		t.Cleanup(ts.Close)

		baseURL, err = url.JoinPath(ts.URL, config.HTTP.Root)
		require.NoError(err)
	}

	return GroupwareTest{
		BaseURL: baseURL,
		Users:   s.Users,
	}, nil
}
