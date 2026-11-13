package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userprovider "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("setting the image of a space", func() {
	const (
		imageID  = "storageid$spaceid!imageid"
		readmeID = "storageid$spaceid!readmeid"
	)

	var (
		svc           service.Service
		gatewayClient *cs3mocks.GatewayAPIClient
		cfg           *config.Config
		ctx           context.Context
		rr            *httptest.ResponseRecorder
	)

	newService := func() (service.Service, error) {
		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewaySelector := pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)
		return service.NewService(
			service.Config(cfg),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&mocks.Publisher{}),
			service.PermissionService(&mocks.Permissions{}),
		)
	}

	// sets the maximum size of a space image, as the configuration does
	setMaxSize := func(size string) {
		cfg.Thumbnails.SpaceImageMaxInputImageFileSize = size
		var err error
		svc, err = newService()
		Expect(err).ToNot(HaveOccurred())
	}

	// sends a PATCH of the space with the given file as its special folder
	patchSpecial := func(folder, fileID string) {
		drive := libregraph.NewDriveUpdate()
		drive.SetSpecial([]libregraph.DriveItem{{
			Id:            libregraph.PtrString(fileID),
			SpecialFolder: &libregraph.SpecialFolder{Name: libregraph.PtrString(folder)},
		}})
		body, err := json.Marshal(drive)
		Expect(err).ToNot(HaveOccurred())

		r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/drives/{driveID}/", bytes.NewBuffer(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("driveID", "storageid$spaceid")
		r = r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
		svc.UpdateDrive(rr, r)
	}

	// the stat of the image file answers with the given size
	statReturnsSize := func(size uint64) {
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
			Status: status.NewOK(ctx),
			Info:   &provider.ResourceInfo{Size: size},
		}, nil)
	}

	updateSucceeds := func() {
		gatewayClient.On("UpdateStorageSpace", mock.Anything, mock.Anything).Return(func(_ context.Context, req *provider.UpdateStorageSpaceRequest, _ ...grpc.CallOption) *provider.UpdateStorageSpaceResponse {
			return &provider.UpdateStorageSpaceResponse{
				Status:       status.NewOK(ctx),
				StorageSpace: req.StorageSpace,
			}
		}, nil)
		// the response describes the updated space, with its quota and special folders
		gatewayClient.On("GetQuota", mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
			Status:     status.NewOK(ctx),
			TotalBytes: 500,
		}, nil)
		gatewayClient.On("GetPath", mock.Anything, mock.Anything).Return(&provider.GetPathResponse{
			Status: status.NewOK(ctx),
			Path:   "/.space/file",
		}, nil)
	}

	expectStored := func(folder, fileID string) {
		gatewayClient.AssertCalled(GinkgoT(), "UpdateStorageSpace", mock.Anything, mock.MatchedBy(func(req *provider.UpdateStorageSpaceRequest) bool {
			return utils.ReadPlainFromOpaque(req.GetStorageSpace().GetOpaque(), folder) == fileID
		}))
	}

	expectNothingStored := func() {
		gatewayClient.AssertNotCalled(GinkgoT(), "UpdateStorageSpace", mock.Anything, mock.Anything)
	}

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		ctx = revactx.ContextSetUser(context.Background(), &userprovider.User{
			Id:       &userprovider.UserId{Type: userprovider.UserType_USER_TYPE_PRIMARY, OpaqueId: "testuser"},
			Username: "testuser",
		})
		gatewayClient = &cs3mocks.GatewayAPIClient{}

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // no LDAP in these tests
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		setMaxSize("1KB")
	})

	It("refuses an image above the maximum size with 413, and keeps the previous one", func() {
		updateSucceeds()
		statReturnsSize(1001)

		patchSpecial("image", imageID)

		Expect(rr.Code).To(Equal(http.StatusRequestEntityTooLarge))
		expectNothingStored()
	})

	It("accepts an image of exactly the maximum size", func() {
		updateSucceeds()
		statReturnsSize(1000) // KB is 1000 bytes, KiB would be 1024

		patchSpecial("image", imageID)

		Expect(rr.Code).To(Equal(http.StatusOK))
		expectStored("image", imageID)
	})

	It("accepts an image of any size when the maximum is empty", func() {
		setMaxSize("")
		updateSucceeds()
		statReturnsSize(1 << 40)

		patchSpecial("image", imageID)

		Expect(rr.Code).To(Equal(http.StatusOK))
		expectStored("image", imageID)
	})

	It("doesn't limit the size of the space readme", func() {
		updateSucceeds()
		statReturnsSize(1 << 40)

		patchSpecial("readme", readmeID)

		Expect(rr.Code).To(Equal(http.StatusOK))
		expectStored("readme", readmeID)
	})

	It("passes an empty image id on as before, without looking it up", func() {
		updateSucceeds()

		patchSpecial("image", "")

		Expect(rr.Code).To(Equal(http.StatusOK))
		gatewayClient.AssertNotCalled(GinkgoT(), "Stat", mock.Anything, mock.Anything)
		gatewayClient.AssertCalled(GinkgoT(), "UpdateStorageSpace", mock.Anything, mock.Anything)
	})

	DescribeTable("keeps the previous image when the image file can't be looked up",
		func(code cs3rpc.Code, want int) {
			updateSucceeds()
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
				Status: &cs3rpc.Status{Code: code, Message: "stat failed"},
			}, nil)

			patchSpecial("image", imageID)

			Expect(rr.Code).To(Equal(want))
			expectNothingStored()
		},
		Entry("not found", cs3rpc.Code_CODE_NOT_FOUND, http.StatusNotFound),
		Entry("internal error", cs3rpc.Code_CODE_INTERNAL, http.StatusInternalServerError),
	)

	It("doesn't start with a maximum size that isn't a size", func() {
		cfg.Thumbnails.SpaceImageMaxInputImageFileSize = "fifty megabytes"
		_, err := newService()
		Expect(err).To(MatchError(ContainSubstring("thumbnails.space_image_max_input_image_file_size")))
	})
})
