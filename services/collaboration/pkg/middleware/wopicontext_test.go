package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"

	appprovider "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/helpers"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/wopisrc"
	"github.com/opencloud-eu/reva/v2/pkg/token"
	rjwt "github.com/opencloud-eu/reva/v2/pkg/token/manager/jwt"
)

var _ = Describe("Wopi Context Middleware", func() {
	var (
		cfg     *config.Config
		ctx     context.Context
		mw      http.Handler
		rid     *providerv1beta1.ResourceId
		tknMngr token.Manager
		user    *userv1beta1.User
		src     *url.URL
	)

	BeforeEach(func() {
		var err error
		cfg = &config.Config{
			TokenManager: &config.TokenManager{JWTSecret: "jwtSecret"},
			Wopi: config.Wopi{
				Secret:  "wopiSecret",
				WopiSrc: "https://localhost:9300",
			},
		}

		ctx = context.Background()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mw = middleware.WopiContextAuthMiddleware(cfg, nil, next)

		tknMngr, err = rjwt.New(map[string]any{
			"secret":  cfg.TokenManager.JWTSecret,
			"expires": int64(24 * 60 * 60),
		})
		Expect(err).ToNot(HaveOccurred())

		user = &userv1beta1.User{
			Id: &userv1beta1.UserId{
				Idp:      "example.com",
				OpaqueId: "12345",
				Type:     userv1beta1.UserType_USER_TYPE_PRIMARY,
			},
			Username: "admin",
			Mail:     "admin@example.com",
		}

		rid = &providerv1beta1.ResourceId{
			StorageId: "storageID",
			OpaqueId:  "opaqueID",
			SpaceId:   "spaceID",
		}

		src, err = url.Parse(cfg.Wopi.WopiSrc)
		src.Path = path.Join("wopi", "files", helpers.HashResourceId(rid))
		Expect(err).ToNot(HaveOccurred())
	})
	It("Should not authorize with empty access token", func() {
		req := httptest.NewRequest("GET", src.String(), nil).WithContext(ctx)
		resp := httptest.NewRecorder()

		mw.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusUnauthorized))
	})
	It("Should not authorize with malformed access token", func() {
		req := httptest.NewRequest("GET", src.String(), nil).WithContext(ctx)
		q := req.URL.Query()
		q.Add("access_token", "token")
		req.URL.RawQuery = q.Encode()

		resp := httptest.NewRecorder()

		mw.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusUnauthorized))
	})
	It("Should not authorize when fileID mismatches", func() {
		req := httptest.NewRequest("GET", src.String(), nil).WithContext(ctx)
		// create request with different fileID in the wopi context
		token, err := tknMngr.MintToken(ctx, user, nil)
		Expect(err).ToNot(HaveOccurred())
		wopiContext := middleware.WopiContext{
			AccessToken: token,
			ViewMode:    appprovider.ViewMode_VIEW_MODE_READ_WRITE,
			FileReference: &providerv1beta1.Reference{
				ResourceId: &providerv1beta1.ResourceId{
					StorageId: "storageID",
					OpaqueId:  "opaqueID2",
					SpaceId:   "spaceID",
				},
				Path: ".",
			},
		}
		wopiToken, ttl, err := middleware.GenerateWopiToken(wopiContext, cfg, nil)
		q := req.URL.Query()
		q.Add("access_token", wopiToken)
		q.Add("access_token_ttl", strconv.FormatInt(ttl, 10))
		req.URL.RawQuery = q.Encode()
		resp := httptest.NewRecorder()

		mw.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusUnauthorized))
	})
	It("Should not authorize with wrong wopi secret", func() {
		src.Path = path.Join("wopi", "files", helpers.HashResourceId(rid))
		req := httptest.NewRequest("GET", src.String(), nil).WithContext(ctx)
		token, err := tknMngr.MintToken(ctx, user, nil)
		Expect(err).ToNot(HaveOccurred())

		wopiContext := middleware.WopiContext{
			AccessToken: token,
		}
		// use wrong wopi secret when generating the wopi token
		wopiToken, ttl, err := middleware.GenerateWopiToken(wopiContext, &config.Config{Wopi: config.Wopi{
			Secret: "wrongSecret",
		}}, nil)
		q := req.URL.Query()
		q.Add("access_token", wopiToken)
		q.Add("access_token_ttl", strconv.FormatInt(ttl, 10))
		req.URL.RawQuery = q.Encode()
		resp := httptest.NewRecorder()

		mw.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusUnauthorized))
	})
	It("Should authorize successful", func() {
		req := httptest.NewRequest("GET", src.String(), nil).WithContext(ctx)
		token, err := tknMngr.MintToken(ctx, user, nil)
		Expect(err).ToNot(HaveOccurred())

		wopiContext := middleware.WopiContext{
			AccessToken: token,
			ViewMode:    appprovider.ViewMode_VIEW_MODE_READ_WRITE,
			FileReference: &providerv1beta1.Reference{
				ResourceId: rid,
				Path:       ".",
			},
		}
		wopiToken, ttl, err := middleware.GenerateWopiToken(wopiContext, cfg, nil)
		q := req.URL.Query()
		q.Add("access_token", wopiToken)
		q.Add("access_token_ttl", strconv.FormatInt(ttl, 10))
		req.URL.RawQuery = q.Encode()
		resp := httptest.NewRecorder()

		mw.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusOK))
	})
	It("Should authorize successful with template reference", func() {
		req := httptest.NewRequest("GET", src.String(), nil).WithContext(ctx)
		token, err := tknMngr.MintToken(ctx, user, nil)
		Expect(err).ToNot(HaveOccurred())

		wopiContext := middleware.WopiContext{
			AccessToken: token,
			ViewMode:    appprovider.ViewMode_VIEW_MODE_READ_WRITE,
			TemplateReference: &providerv1beta1.Reference{
				ResourceId: rid,
				Path:       ".",
			},
			FileReference: &providerv1beta1.Reference{
				ResourceId: &providerv1beta1.ResourceId{
					StorageId: "storageID",
					OpaqueId:  "opaqueID2",
					SpaceId:   "spaceID",
				},
			},
		}
		wopiToken, ttl, err := middleware.GenerateWopiToken(wopiContext, cfg, nil)
		q := req.URL.Query()
		q.Add("access_token", wopiToken)
		q.Add("access_token_ttl", strconv.FormatInt(ttl, 10))
		req.URL.RawQuery = q.Encode()
		resp := httptest.NewRecorder()

		mw.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusOK))
	})
	It("Should not authorize when no reference matches", func() {
		req := httptest.NewRequest("GET", src.String(), nil).WithContext(ctx)
		token, err := tknMngr.MintToken(ctx, user, nil)
		Expect(err).ToNot(HaveOccurred())

		wopiContext := middleware.WopiContext{
			AccessToken: token,
			ViewMode:    appprovider.ViewMode_VIEW_MODE_READ_WRITE,
			TemplateReference: &providerv1beta1.Reference{
				ResourceId: &providerv1beta1.ResourceId{
					StorageId: "storageID",
					OpaqueId:  "opaqueID3",
					SpaceId:   "spaceID",
				},
				Path: ".",
			},
			FileReference: &providerv1beta1.Reference{
				ResourceId: &providerv1beta1.ResourceId{
					StorageId: "storageID",
					OpaqueId:  "opaqueID2",
					SpaceId:   "spaceID",
				},
			},
		}
		wopiToken, ttl, err := middleware.GenerateWopiToken(wopiContext, cfg, nil)
		q := req.URL.Query()
		q.Add("access_token", wopiToken)
		q.Add("access_token_ttl", strconv.FormatInt(ttl, 10))
		req.URL.RawQuery = q.Encode()
		resp := httptest.NewRecorder()

		mw.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusUnauthorized))
	})
	It("Should not authorize with proxy when fileID mismatches", func() {
		cfg.Wopi.ProxySecret = "proxySecret"
		cfg.Wopi.ProxyURL = "https://proxy"
		src, err := wopisrc.GenerateWopiSrc(helpers.HashResourceId(rid), cfg)
		Expect(err).ToNot(HaveOccurred())

		req := httptest.NewRequest("GET", src.String(), nil).WithContext(ctx)
		token, err := tknMngr.MintToken(ctx, user, nil)
		Expect(err).ToNot(HaveOccurred())
		wopiContext := middleware.WopiContext{
			AccessToken: token,
			ViewMode:    appprovider.ViewMode_VIEW_MODE_READ_WRITE,
			FileReference: &providerv1beta1.Reference{
				ResourceId: &providerv1beta1.ResourceId{
					StorageId: "storageID",
					OpaqueId:  "opaqueID3",
					SpaceId:   "spaceID",
				},
				Path: ".",
			},
		}
		wopiToken, ttl, err := middleware.GenerateWopiToken(wopiContext, cfg, nil)
		q := req.URL.Query()
		q.Add("access_token", wopiToken)
		q.Add("access_token_ttl", strconv.FormatInt(ttl, 10))
		req.URL.RawQuery = q.Encode()

		resp := httptest.NewRecorder()
		mw.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusUnauthorized))
	})
	It("Should authorize successful with proxy", func() {
		cfg.Wopi.ProxySecret = "proxySecret"
		cfg.Wopi.ProxyURL = "https://proxy"
		src, err := wopisrc.GenerateWopiSrc(helpers.HashResourceId(rid), cfg)
		Expect(err).ToNot(HaveOccurred())

		req := httptest.NewRequest("GET", src.String(), nil).WithContext(ctx)
		token, err := tknMngr.MintToken(ctx, user, nil)
		Expect(err).ToNot(HaveOccurred())
		wopiContext := middleware.WopiContext{
			AccessToken: token,
			ViewMode:    appprovider.ViewMode_VIEW_MODE_READ_WRITE,
			FileReference: &providerv1beta1.Reference{
				ResourceId: rid,
				Path:       ".",
			},
		}
		wopiToken, ttl, err := middleware.GenerateWopiToken(wopiContext, cfg, nil)
		q := req.URL.Query()
		q.Add("access_token", wopiToken)
		q.Add("access_token_ttl", strconv.FormatInt(ttl, 10))
		req.URL.RawQuery = q.Encode()

		resp := httptest.NewRecorder()
		mw.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusOK))
	})
})
