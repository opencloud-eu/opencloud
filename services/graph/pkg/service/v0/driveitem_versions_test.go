package svc_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/signedurl"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("DriveItemVersions", func() {
	const (
		urlSigningSecret = "url-signing-secret"
		driveID          = "storageid$spaceid"
		itemID           = "storageid$spaceid!nodeid"
		olderKey         = "nodeid.REV.2026-09-01T10:00:00.000000000Z"
		newerKey         = "nodeid.REV.2026-09-05T10:00:00.000000000Z"
	)

	var (
		svc             service.Service
		ctx             context.Context
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher mocks.Publisher
		identityBackend *identitymocks.Backend
		rr              *httptest.ResponseRecorder

		fileInfo *provider.ResourceInfo
		older    *provider.FileVersion
		newer    *provider.FileVersion
		fileTime = time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

		currentUser = &userpb.User{
			Id: &userpb.UserId{
				OpaqueId: "user",
			},
		}
	)

	newRequest := func(method, versionPath, query string) *http.Request {
		r := httptest.NewRequest(method, "/graph/v1.0/drives/"+driveID+"/items/"+itemID+"/versions"+versionPath+query, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("driveID", driveID)
		rctx.URLParams.Add("driveItemID", itemID)
		return r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
	}

	newVersionRequest := func(method, versionID, suffix, query string) *http.Request {
		r := newRequest(method, "/"+versionID+suffix, query)
		chi.RouteContext(r.Context()).URLParams.Add("versionID", versionID)
		return r
	}

	verifiedTarget := func(signed string) *url.URL {
		verifier, err := signedurl.NewJWTSignedURL(signedurl.WithSecret(urlSigningSecret))
		Expect(err).ToNot(HaveOccurred())
		subject, err := verifier.Verify(signed)
		Expect(err).ToNot(HaveOccurred())
		Expect(subject).To(Equal("user"))

		target, err := url.Parse(signed)
		Expect(err).ToNot(HaveOccurred())
		return target
	}

	BeforeEach(func() {
		eventsPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		logger := log.NewLogger()
		identityBackend = &identitymocks.Backend{}
		metrics, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })

		rr = httptest.NewRecorder()
		ctx = context.Background()

		cfg := defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{
			URLSigningSecret: urlSigningSecret,
		}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		var err error
		svc, err = service.NewService(
			service.Config(cfg),
			service.Metrics(metrics),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
			service.WithIdentityBackend(identityBackend),
		)
		Expect(err).ToNot(HaveOccurred())

		fileInfo = &provider.ResourceInfo{
			Type:  provider.ResourceType_RESOURCE_TYPE_FILE,
			Id:    &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "nodeid"},
			Etag:  "etag",
			Size:  300,
			Mtime: utils.TimeToTS(fileTime),
		}
		older = &provider.FileVersion{Key: olderKey, Size: 100, Mtime: uint64(fileTime.Add(-6 * 24 * time.Hour).Unix())}
		newer = &provider.FileVersion{Key: newerKey, Size: 200, Mtime: uint64(fileTime.Add(-2 * 24 * time.Hour).Unix())}

		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
			Status: status.NewOK(ctx),
			Info:   fileInfo,
		}, nil)
		gatewayClient.On("ListFileVersions", mock.Anything, mock.Anything).Return(&provider.ListFileVersionsResponse{
			Status:   status.NewOK(ctx),
			Versions: []*provider.FileVersion{older, newer},
		}, nil)
	})

	Describe("ListDriveItemVersions", func() {
		list := func(r *http.Request) []libregraph.DriveItemVersion {
			svc.ListDriveItemVersions(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := libregraph.CollectionOfDriveItemVersions{}
			Expect(json.Unmarshal(data, &res)).To(Succeed())
			return res.Value
		}

		It("lists the versions newest first", func() {
			versions := list(newRequest(http.MethodGet, "", ""))
			Expect(versions).To(HaveLen(2))

			Expect(versions[0].GetId()).To(Equal(newerKey))
			Expect(versions[0].GetSize()).To(Equal(int64(200)))
			Expect(versions[0].GetLastModifiedDateTime()).To(Equal(fileTime.Add(-2 * 24 * time.Hour)))
			Expect(versions[0].MicrosoftGraphDownloadUrl).To(BeNil())
			Expect(versions[0].LastModifiedBy).To(BeNil())

			Expect(versions[1].GetId()).To(Equal(olderKey))
			Expect(versions[1].GetSize()).To(Equal(int64(100)))
		})

		It("returns an empty list for a file without versions", func() {
			gatewayClient.ExpectedCalls = nil
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{Status: status.NewOK(ctx), Info: fileInfo}, nil)
			gatewayClient.On("ListFileVersions", mock.Anything, mock.Anything).Return(&provider.ListFileVersionsResponse{Status: status.NewOK(ctx)}, nil)

			svc.ListDriveItemVersions(rr, newRequest(http.MethodGet, "", ""))
			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(MatchJSON(`{"value":[]}`))
		})

		It("adds a signed download url for every version when requested via $select", func() {
			versions := list(newRequest(http.MethodGet, "", "?$select=@microsoft.graph.downloadUrl"))
			Expect(versions).To(HaveLen(2))

			target := verifiedTarget(versions[0].GetMicrosoftGraphDownloadUrl())
			Expect(target.Path).To(Equal("/dav/meta/" + itemID + "/v/" + newerKey))
			Expect(verifiedTarget(versions[1].GetMicrosoftGraphDownloadUrl()).Path).To(Equal("/dav/meta/" + itemID + "/v/" + olderKey))
		})

		It("returns 404 for a folder", func() {
			fileInfo.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER

			svc.ListDriveItemVersions(rr, newRequest(http.MethodGet, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "ListFileVersions", mock.Anything, mock.Anything)
		})

		It("returns 404 for an unknown item", func() {
			gatewayClient.ExpectedCalls = nil
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{Status: status.NewNotFound(ctx, "not found")}, nil)

			svc.ListDriveItemVersions(rr, newRequest(http.MethodGet, "", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 404 when the item belongs to another drive", func() {
			r := newRequest(http.MethodGet, "", "")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "storageid$otherspace")
			rctx.URLParams.Add("driveItemID", itemID)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			svc.ListDriveItemVersions(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "Stat", mock.Anything, mock.Anything)
		})
	})

	Describe("GetDriveItemVersion", func() {
		get := func(r *http.Request) libregraph.DriveItemVersion {
			svc.GetDriveItemVersion(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			version := libregraph.DriveItemVersion{}
			Expect(json.Unmarshal(data, &version)).To(Succeed())
			return version
		}

		It("returns the requested version", func() {
			version := get(newVersionRequest(http.MethodGet, olderKey, "", ""))
			Expect(version.GetId()).To(Equal(olderKey))
			Expect(version.GetSize()).To(Equal(int64(100)))
			Expect(version.MicrosoftGraphDownloadUrl).To(BeNil())
		})

		It("adds a signed download url when requested via $select", func() {
			version := get(newVersionRequest(http.MethodGet, olderKey, "", "?$select=@microsoft.graph.downloadUrl"))
			Expect(verifiedTarget(version.GetMicrosoftGraphDownloadUrl()).Path).To(Equal("/dav/meta/" + itemID + "/v/" + olderKey))
		})

		It("describes the file itself as the current version", func() {
			version := get(newVersionRequest(http.MethodGet, "current", "", "?$select=@microsoft.graph.downloadUrl"))
			Expect(version.GetId()).To(Equal("current"))
			Expect(version.GetSize()).To(Equal(int64(300)))
			Expect(version.GetLastModifiedDateTime()).To(Equal(fileTime))
			Expect(verifiedTarget(version.GetMicrosoftGraphDownloadUrl()).Path).To(Equal("/dav/spaces/" + itemID))
			gatewayClient.AssertNotCalled(GinkgoT(), "ListFileVersions", mock.Anything, mock.Anything)
		})

		It("returns 404 for an unknown version", func() {
			svc.GetDriveItemVersion(rr, newVersionRequest(http.MethodGet, "nodeid.REV.unknown", "", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("GetDriveItemVersionContent", func() {
		It("redirects to a signed download url for the version", func() {
			svc.GetDriveItemVersionContent(rr, newVersionRequest(http.MethodGet, olderKey, "/content", ""))
			Expect(rr.Code).To(Equal(http.StatusFound))
			Expect(verifiedTarget(rr.Header().Get("Location")).Path).To(Equal("/dav/meta/" + itemID + "/v/" + olderKey))
		})

		It("redirects to the file itself for the current version", func() {
			svc.GetDriveItemVersionContent(rr, newVersionRequest(http.MethodGet, "current", "/content", ""))
			Expect(rr.Code).To(Equal(http.StatusFound))
			Expect(verifiedTarget(rr.Header().Get("Location")).Path).To(Equal("/dav/spaces/" + itemID))
		})

		It("returns 404 for an unknown version", func() {
			svc.GetDriveItemVersionContent(rr, newVersionRequest(http.MethodGet, "nodeid.REV.unknown", "/content", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("RestoreDriveItemVersion", func() {
		It("restores the version and answers 204", func() {
			gatewayClient.On("RestoreFileVersion", mock.Anything, mock.MatchedBy(func(req *provider.RestoreFileVersionRequest) bool {
				return req.GetKey() == olderKey && req.GetRef().GetResourceId().GetOpaqueId() == "nodeid"
			})).Return(&provider.RestoreFileVersionResponse{Status: status.NewOK(ctx)}, nil)

			svc.RestoreDriveItemVersion(rr, newVersionRequest(http.MethodPost, olderKey, "/restoreVersion", ""))
			Expect(rr.Code).To(Equal(http.StatusNoContent))
		})

		It("returns 423 when the file is locked", func() {
			gatewayClient.On("RestoreFileVersion", mock.Anything, mock.Anything).Return(&provider.RestoreFileVersionResponse{Status: status.NewLocked(ctx, "locked")}, nil)

			svc.RestoreDriveItemVersion(rr, newVersionRequest(http.MethodPost, olderKey, "/restoreVersion", ""))
			Expect(rr.Code).To(Equal(http.StatusLocked))
		})

		It("returns 404 for an unknown version", func() {
			gatewayClient.On("RestoreFileVersion", mock.Anything, mock.Anything).Return(&provider.RestoreFileVersionResponse{Status: status.NewNotFound(ctx, "not found")}, nil)

			svc.RestoreDriveItemVersion(rr, newVersionRequest(http.MethodPost, "nodeid.REV.unknown", "/restoreVersion", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 404 for a folder", func() {
			fileInfo.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER

			svc.RestoreDriveItemVersion(rr, newVersionRequest(http.MethodPost, olderKey, "/restoreVersion", ""))
			Expect(rr.Code).To(Equal(http.StatusNotFound))
			gatewayClient.AssertNotCalled(GinkgoT(), "RestoreFileVersion", mock.Anything, mock.Anything)
		})
	})
})
