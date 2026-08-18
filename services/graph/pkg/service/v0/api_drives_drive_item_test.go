package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaborationv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/mock"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	graphm "github.com/opencloud-eu/opencloud/services/graph/pkg/middleware"
	svc "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("DrivesDriveItemService", func() {
	var (
		drivesDriveItemService svc.DrivesDriveItemService
		gatewayClient          *cs3mocks.GatewayAPIClient
		gatewaySelector        *mocks.Selectable[gateway.GatewayAPIClient]
	)

	BeforeEach(func() {
		logger := log.NewLogger()
		gatewayClient = cs3mocks.NewGatewayAPIClient(GinkgoT())

		gatewaySelector = mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
		gatewaySelector.EXPECT().Next().Return(gatewayClient, nil).Maybe()

		service, err := svc.NewDrivesDriveItemService(logger, gatewaySelector)
		Expect(err).ToNot(HaveOccurred())
		drivesDriveItemService = service
	})

	failOnFailingGatewayClientRotation := func(f func() error) {
		It("fails if obtaining the next gateway client fails", func() {
			someErr := errors.New("some error")
			gatewaySelector.EXPECT().Next().Unset()
			gatewaySelector.EXPECT().Next().Return(gatewayClient, someErr).Times(1)
			Expect(f()).To(MatchError(someErr))
		})
	}

	var _ = Describe("GetSharesForResource", func() {
		failOnFailingGatewayClientRotation(func() error {
			_, err := drivesDriveItemService.GetSharesForResource(context.Background(), nil, nil)
			return err
		})

		It("uses the correct filters to list received shares", func() {
			resourceID := &storageprovider.ResourceId{
				StorageId: "1",
				OpaqueId:  "2",
				SpaceId:   "3",
			}
			state := collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED

			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *collaborationv1beta1.ListReceivedSharesRequest, option ...grpc.CallOption) (*collaborationv1beta1.ListReceivedSharesResponse, error) {
					Expect(request.Filters).To(HaveLen(2))
					Expect(request.Filters[0].Term.(*collaborationv1beta1.Filter_ResourceId).ResourceId).To(Equal(resourceID))
					Expect(request.Filters[1].Term.(*collaborationv1beta1.Filter_State).State).To(Equal(state))
					return nil, nil
				}).
				Once()

			_, _ = drivesDriveItemService.GetSharesForResource(context.Background(), resourceID, []*collaborationv1beta1.Filter{
				{
					Type: collaborationv1beta1.Filter_TYPE_STATE,
					Term: &collaborationv1beta1.Filter_State{
						State: state,
					},
				},
			})
		})

		It("fails on ancestor error", func() {
			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				Return(nil, someErr).
				Once()

			_, err := drivesDriveItemService.GetSharesForResource(context.Background(), &storageprovider.ResourceId{}, []*collaborationv1beta1.Filter{})
			Expect(err).To(MatchError(someErr))
		})

		It("fails if no shares are found", func() {
			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				Return(nil, nil).
				Once()

			_, err := drivesDriveItemService.GetSharesForResource(context.Background(), &storageprovider.ResourceId{}, []*collaborationv1beta1.Filter{})
			Expect(err).To(MatchError(svc.ErrNoShares))
		})

		It("successfully returns shares", func() {
			givenShares := []*collaborationv1beta1.ReceivedShare{
				{State: collaborationv1beta1.ShareState_SHARE_STATE_PENDING},
				{State: collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED},
				{State: collaborationv1beta1.ShareState_SHARE_STATE_REJECTED},
			}
			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				Return(&collaborationv1beta1.ListReceivedSharesResponse{
					Status: status.NewOK(context.Background()),
					Shares: givenShares,
				}, nil).
				Once()

			shares, err := drivesDriveItemService.GetSharesForResource(context.Background(), &storageprovider.ResourceId{}, []*collaborationv1beta1.Filter{})
			Expect(err).To(BeNil())
			Expect(shares).To(Equal(givenShares))
		})
	})

	var _ = Describe("GetShare", func() {
		failOnFailingGatewayClientRotation(func() error {
			_, err := drivesDriveItemService.GetShare(context.Background(), nil)
			return err
		})

		It("fails if share lookup reports an error", func() {
			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				GetReceivedShare(context.Background(), mock.Anything, mock.Anything).
				Return(nil, someErr).
				Once()

			_, err := drivesDriveItemService.GetShare(context.Background(), &collaborationv1beta1.ShareId{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, someErr.Error()).WithOrigin(errorcode.ErrorOriginCS3)))
		})

		It("fails if share lookup does not report an error but the status is off", func() {
			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				GetReceivedShare(context.Background(), mock.Anything, mock.Anything).
				Return(&collaborationv1beta1.GetReceivedShareResponse{
					Status: status.NewInvalid(context.Background(), someErr.Error()),
				}, nil).
				Once()

			_, err := drivesDriveItemService.GetShare(context.Background(), &collaborationv1beta1.ShareId{})
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, someErr.Error()).WithOrigin(errorcode.ErrorOriginCS3)))
		})

		It("successfully returns a share", func() {
			gatewayClient.
				EXPECT().
				GetReceivedShare(context.Background(), mock.Anything, mock.Anything).
				Return(&collaborationv1beta1.GetReceivedShareResponse{
					Status: status.NewOK(context.Background()),
					Share: &collaborationv1beta1.ReceivedShare{
						Share: &collaborationv1beta1.Share{
							Id: &collaborationv1beta1.ShareId{
								OpaqueId: "123",
							},
						},
					},
				}, nil).
				Once()

			share, err := drivesDriveItemService.GetShare(context.Background(), &collaborationv1beta1.ShareId{})
			Expect(err).To(BeNil())
			Expect(share.GetShare().GetId().GetOpaqueId()).To(Equal("123"))
		})
	})

	var _ = Describe("UpdateShare", func() {
		failOnFailingGatewayClientRotation(func() error {
			_, err := drivesDriveItemService.UpdateShare(context.Background(), nil, nil)
			return err
		})

		It("fails without an updater", func() {
			_, err := drivesDriveItemService.UpdateShare(context.Background(), &collaborationv1beta1.ReceivedShare{}, nil)
			Expect(err).To(MatchError(svc.ErrNoUpdater))
		})

		It("fails without updates", func() {
			_, err := drivesDriveItemService.UpdateShare(context.Background(), &collaborationv1beta1.ReceivedShare{}, func(*collaborationv1beta1.ReceivedShare, *collaborationv1beta1.UpdateReceivedShareRequest) {})
			Expect(err).To(MatchError(svc.ErrNoUpdates))
		})

		It("fails if share updates reports an error", func() {
			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				UpdateReceivedShare(context.Background(), mock.Anything, mock.Anything).
				Return(nil, someErr).
				Once()

			_, err := drivesDriveItemService.UpdateShare(context.Background(), &collaborationv1beta1.ReceivedShare{}, func(_ *collaborationv1beta1.ReceivedShare, request *collaborationv1beta1.UpdateReceivedShareRequest) {
				request.Share.State = collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED
				request.UpdateMask.Paths = append(request.UpdateMask.Paths, "state")
			})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, someErr.Error()).WithOrigin(errorcode.ErrorOriginCS3)))
		})

		It("fails if share update does not report an error but the status is off", func() {
			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				UpdateReceivedShare(context.Background(), mock.Anything, mock.Anything).
				Return(&collaborationv1beta1.UpdateReceivedShareResponse{
					Status: status.NewNotFound(context.Background(), someErr.Error()),
				}, nil).
				Once()

			_, err := drivesDriveItemService.UpdateShare(context.Background(), &collaborationv1beta1.ReceivedShare{}, func(_ *collaborationv1beta1.ReceivedShare, request *collaborationv1beta1.UpdateReceivedShareRequest) {
				request.Share.State = collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED
				request.UpdateMask.Paths = append(request.UpdateMask.Paths, "state")
			})
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, someErr.Error()).WithOrigin(errorcode.ErrorOriginCS3)))
		})
	})

	var _ = Describe("UpdateShares", func() {
		It("reports some error if one or multiple shares could not be updated, successfully updates the rest", func() {
			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				UpdateReceivedShare(context.Background(), mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *collaborationv1beta1.UpdateReceivedShareRequest, option ...grpc.CallOption) (*collaborationv1beta1.UpdateReceivedShareResponse, error) {
					if request.GetShare().GetShare().GetId() != nil {
						return &collaborationv1beta1.UpdateReceivedShareResponse{
							Status: status.NewOK(ctx),
						}, nil
					}

					return nil, someErr
				}).
				Times(3)

			shares, err := drivesDriveItemService.UpdateShares(context.Background(), []*collaborationv1beta1.ReceivedShare{
				{},
				{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}},
				{},
			}, func(_ *collaborationv1beta1.ReceivedShare, request *collaborationv1beta1.UpdateReceivedShareRequest) {
				request.Share.State = collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED
				request.UpdateMask.Paths = append(request.UpdateMask.Paths, "state")
			})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, someErr.Error()).WithOrigin(errorcode.ErrorOriginCS3)))
			Expect(err.(interface{ Unwrap() []error }).Unwrap()).To(HaveLen(2))
			Expect(shares).To(HaveLen(1))
		})
	})

	var _ = Describe("UnmountShare", func() {
		It("fails if get share and siblings reports an error", func() {
			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				GetReceivedShare(context.Background(), mock.Anything, mock.Anything).
				Return(nil, someErr).
				Once()

			err := drivesDriveItemService.UnmountShare(context.Background(), &collaborationv1beta1.ShareId{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, someErr.Error()).WithOrigin(errorcode.ErrorOriginCS3)))
		})

		It("requests only accepted shares to be unmounted", func() {
			gatewayClient.
				EXPECT().
				GetReceivedShare(context.Background(), mock.Anything, mock.Anything).
				Return(&collaborationv1beta1.GetReceivedShareResponse{
					Status: status.NewOK(context.Background()),
				}, nil).
				Once()

			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *collaborationv1beta1.ListReceivedSharesRequest, option ...grpc.CallOption) (*collaborationv1beta1.ListReceivedSharesResponse, error) {
					Expect(request.Filters).To(HaveLen(3))
					Expect(request.Filters[0].Type).To(Equal(collaborationv1beta1.Filter_TYPE_RESOURCE_ID))
					Expect(request.Filters[1].Term.(*collaborationv1beta1.Filter_State).State).To(Equal(collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED))
					Expect(request.Filters[2].Term.(*collaborationv1beta1.Filter_State).State).To(Equal(collaborationv1beta1.ShareState_SHARE_STATE_REJECTED))
					return &collaborationv1beta1.ListReceivedSharesResponse{
						Status: status.NewOK(context.Background()),
						Shares: []*collaborationv1beta1.ReceivedShare{
							{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_REJECTED},
							{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED},
							{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED},
						},
					}, nil
				}).
				Once()

			gatewayClient.
				EXPECT().
				UpdateReceivedShare(context.Background(), mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *collaborationv1beta1.UpdateReceivedShareRequest, option ...grpc.CallOption) (*collaborationv1beta1.UpdateReceivedShareResponse, error) {
					return &collaborationv1beta1.UpdateReceivedShareResponse{
						Status: status.NewOK(ctx),
					}, nil
				}).
				Times(2)

			err := drivesDriveItemService.UnmountShare(context.Background(), &collaborationv1beta1.ShareId{})
			Expect(err).To(BeNil())
		})

		It("reports some error if one or multiple shares could not be unmounted", func() {
			gatewayClient.
				EXPECT().
				GetReceivedShare(context.Background(), mock.Anything, mock.Anything).
				Return(&collaborationv1beta1.GetReceivedShareResponse{
					Status: status.NewOK(context.Background()),
				}, nil).
				Once()

			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				Return(&collaborationv1beta1.ListReceivedSharesResponse{
					Status: status.NewOK(context.Background()),
					Shares: []*collaborationv1beta1.ReceivedShare{
						{},
						{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED},
						{},
					},
				}, nil).
				Once()

			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				UpdateReceivedShare(context.Background(), mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *collaborationv1beta1.UpdateReceivedShareRequest, option ...grpc.CallOption) (*collaborationv1beta1.UpdateReceivedShareResponse, error) {
					return nil, someErr
				}).
				Times(1)

			err := drivesDriveItemService.UnmountShare(context.Background(), &collaborationv1beta1.ShareId{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, someErr.Error()).WithOrigin(errorcode.ErrorOriginCS3)))
			Expect(err.(interface{ Unwrap() []error }).Unwrap()).To(HaveLen(1))
		})
	})

	var _ = Describe("MountShare", func() {
		It("fails if name is interpreted as absolute path", func() {
			_, err := drivesDriveItemService.MountShare(context.Background(), nil, "/some")
			Expect(err).To(MatchError(svc.ErrAbsoluteNamePath))
		})

		It("uses the correct filters to list received shares", func() {
			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *collaborationv1beta1.ListReceivedSharesRequest, option ...grpc.CallOption) (*collaborationv1beta1.ListReceivedSharesResponse, error) {
					Expect(request.Filters).To(HaveLen(1))
					Expect(request.Filters[0].Type).To(Equal(collaborationv1beta1.Filter_TYPE_RESOURCE_ID))
					return nil, nil
				}).
				Once()

			_, _ = drivesDriveItemService.MountShare(context.Background(), nil, "some")
		})

		It("reports no errors when shares have mounted", func() {
			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *collaborationv1beta1.ListReceivedSharesRequest, option ...grpc.CallOption) (*collaborationv1beta1.ListReceivedSharesResponse, error) {
					return &collaborationv1beta1.ListReceivedSharesResponse{
						Status: status.NewOK(context.Background()),
						Shares: []*collaborationv1beta1.ReceivedShare{
							{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_REJECTED},
							{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED},
							{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_PENDING},
						},
					}, nil
				}).
				Once()
			gatewayClient.
				EXPECT().
				UpdateReceivedShare(context.Background(), mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *collaborationv1beta1.UpdateReceivedShareRequest, option ...grpc.CallOption) (*collaborationv1beta1.UpdateReceivedShareResponse, error) {
					return &collaborationv1beta1.UpdateReceivedShareResponse{
						Status: status.NewOK(ctx),
					}, nil
				}).
				Times(2)

			shares, err := drivesDriveItemService.MountShare(context.Background(), nil, "some")
			Expect(err).To(BeNil())
			Expect(shares).To(HaveLen(2))
		})

		It("fails if the update instructions produce as many errors as there are shares", func() {
			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				Return(&collaborationv1beta1.ListReceivedSharesResponse{
					Status: status.NewOK(context.Background()),
					Shares: []*collaborationv1beta1.ReceivedShare{
						{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_PENDING},
						{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_PENDING},
						{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_PENDING},
					},
				}, nil).
				Once()

			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				UpdateReceivedShare(context.Background(), mock.Anything, mock.Anything).
				Return(nil, someErr).
				Times(3)

			shares, err := drivesDriveItemService.MountShare(context.Background(), nil, "some")
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, someErr.Error()).WithOrigin(errorcode.ErrorOriginCS3)))
			Expect(err.(interface{ Unwrap() []error }).Unwrap()).To(HaveLen(3))
			Expect(shares).To(HaveLen(0))
		})

		It("reports no errors if not all mount requests fail", func() {
			gatewayClient.
				EXPECT().
				ListReceivedShares(context.Background(), mock.Anything, mock.Anything, mock.Anything).
				Return(&collaborationv1beta1.ListReceivedSharesResponse{
					Status: status.NewOK(context.Background()),
					Shares: []*collaborationv1beta1.ReceivedShare{
						{},
						{Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{}}, State: collaborationv1beta1.ShareState_SHARE_STATE_PENDING},
						{},
					},
				}, nil).
				Once()

			someErr := errors.New("some error")
			gatewayClient.
				EXPECT().
				UpdateReceivedShare(context.Background(), mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *collaborationv1beta1.UpdateReceivedShareRequest, option ...grpc.CallOption) (*collaborationv1beta1.UpdateReceivedShareResponse, error) {
					if request.GetShare().GetShare().GetId() == nil {
						return nil, someErr
					}

					Expect(request.GetShare().GetState()).To(Equal(collaborationv1beta1.ShareState_SHARE_STATE_ACCEPTED))
					Expect(request.GetShare().GetMountPoint().GetPath()).To(Equal("some"))

					return &collaborationv1beta1.UpdateReceivedShareResponse{
						Status: status.NewOK(ctx),
					}, nil
				}).
				Once()

			shares, err := drivesDriveItemService.MountShare(context.Background(), nil, "some")
			Expect(err).To(BeNil())
			Expect(shares).To(HaveLen(1))
		})
	})

	var _ = Describe("CreateChild", func() {
		parentID := &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}

		It("rejects invalid names", func() {
			for _, name := range []string{"", ".", "..", "a/b", "/a"} {
				_, err := drivesDriveItemService.CreateChild(context.Background(), parentID, name, true, false)
				Expect(err).To(MatchError(svc.ErrInvalidItemName))
			}
		})

		It("creates a folder", func() {
			gatewayClient.
				EXPECT().
				CreateContainer(mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *storageprovider.CreateContainerRequest, _ ...grpc.CallOption) (*storageprovider.CreateContainerResponse, error) {
					Expect(request.GetRef().GetResourceId().GetOpaqueId()).To(Equal("2"))
					Expect(request.GetRef().GetPath()).To(Equal("./New Folder"))
					return &storageprovider.CreateContainerResponse{Status: status.NewOK(ctx)}, nil
				}).
				Once()
			gatewayClient.
				EXPECT().
				Stat(mock.Anything, mock.Anything, mock.Anything).
				Return(&storageprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   &storageprovider.ResourceInfo{Id: &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"}},
				}, nil).
				Times(2)

			info, err := drivesDriveItemService.CreateChild(context.Background(), parentID, "New Folder", true, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.GetId().GetOpaqueId()).To(Equal("3"))
		})

		It("creates an empty file", func() {
			gatewayClient.
				EXPECT().
				TouchFile(mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *storageprovider.TouchFileRequest, _ ...grpc.CallOption) (*storageprovider.TouchFileResponse, error) {
					Expect(request.GetRef().GetPath()).To(Equal("./file.txt"))
					return &storageprovider.TouchFileResponse{Status: status.NewOK(ctx)}, nil
				}).
				Once()
			gatewayClient.
				EXPECT().
				Stat(mock.Anything, mock.Anything, mock.Anything).
				Return(&storageprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   &storageprovider.ResourceInfo{Id: &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"}},
				}, nil).
				Times(2)

			_, err := drivesDriveItemService.CreateChild(context.Background(), parentID, "file.txt", false, false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails with nameAlreadyExists without replace", func() {
			gatewayClient.
				EXPECT().
				TouchFile(mock.Anything, mock.Anything, mock.Anything).
				Return(&storageprovider.TouchFileResponse{Status: status.NewAlreadyExists(context.Background(), nil, "exists")}, nil).
				Once()

			_, err := drivesDriveItemService.CreateChild(context.Background(), parentID, "file.txt", false, false)
			var lgErr errorcode.Error
			Expect(errors.As(err, &lgErr)).To(BeTrue())
			Expect(lgErr.GetCode()).To(Equal(errorcode.NameAlreadyExists))
		})

		It("deletes the existing item with replace", func() {
			gatewayClient.
				EXPECT().
				CreateContainer(mock.Anything, mock.Anything, mock.Anything).
				Return(&storageprovider.CreateContainerResponse{Status: status.NewAlreadyExists(context.Background(), nil, "exists")}, nil).
				Once()
			gatewayClient.
				EXPECT().
				Delete(mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *storageprovider.DeleteRequest, _ ...grpc.CallOption) (*storageprovider.DeleteResponse, error) {
					Expect(request.GetRef().GetPath()).To(Equal("./New Folder"))
					return &storageprovider.DeleteResponse{Status: status.NewOK(ctx)}, nil
				}).
				Once()
			gatewayClient.
				EXPECT().
				CreateContainer(mock.Anything, mock.Anything, mock.Anything).
				Return(&storageprovider.CreateContainerResponse{Status: status.NewOK(context.Background())}, nil).
				Once()
			gatewayClient.
				EXPECT().
				Stat(mock.Anything, mock.Anything, mock.Anything).
				Return(&storageprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   &storageprovider.ResourceInfo{Id: &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"}},
				}, nil).
				Times(2)

			_, err := drivesDriveItemService.CreateChild(context.Background(), parentID, "New Folder", true, true)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	var _ = Describe("CreatePath", func() {
		parentID := &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}
		pathID := &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"}

		statResponse := func(code *cs3rpc.Status) *storageprovider.StatResponse {
			res := &storageprovider.StatResponse{Status: code}
			if code.GetCode() == cs3rpc.Code_CODE_OK {
				res.Info = &storageprovider.ResourceInfo{Id: pathID}
			}
			return res
		}

		It("rejects invalid path segments", func() {
			for _, relPath := range []string{"", ".", "..", "a/../b", "a//b"} {
				_, err := drivesDriveItemService.CreatePath(context.Background(), parentID, relPath)
				Expect(err).To(MatchError(svc.ErrInvalidItemName))
			}
		})

		It("returns the id without creating anything when the path exists", func() {
			gatewayClient.
				EXPECT().
				Stat(mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *storageprovider.StatRequest, _ ...grpc.CallOption) (*storageprovider.StatResponse, error) {
					Expect(request.GetRef().GetPath()).To(Equal("./a/b"))
					return statResponse(status.NewOK(ctx)), nil
				}).
				Once()

			id, err := drivesDriveItemService.CreatePath(context.Background(), parentID, "/a/b")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.GetOpaqueId()).To(Equal("3"))
		})

		It("creates the folders along a missing path, reusing existing ones", func() {
			gatewayClient.
				EXPECT().
				Stat(mock.Anything, mock.Anything, mock.Anything).
				Return(statResponse(status.NewNotFound(context.Background(), "missing")), nil).
				Once()
			gatewayClient.
				EXPECT().
				CreateContainer(mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, request *storageprovider.CreateContainerRequest, _ ...grpc.CallOption) (*storageprovider.CreateContainerResponse, error) {
					switch request.GetRef().GetPath() {
					case "./a":
						// exists already, reused
						return &storageprovider.CreateContainerResponse{Status: status.NewAlreadyExists(ctx, nil, "exists")}, nil
					case "./a/b":
						return &storageprovider.CreateContainerResponse{Status: status.NewOK(ctx)}, nil
					default:
						Fail("unexpected CreateContainer path " + request.GetRef().GetPath())
						return nil, nil
					}
				}).
				Times(2)
			gatewayClient.
				EXPECT().
				Stat(mock.Anything, mock.Anything, mock.Anything).
				Return(statResponse(status.NewOK(context.Background())), nil).
				Once()

			id, err := drivesDriveItemService.CreatePath(context.Background(), parentID, "/a/b")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.GetOpaqueId()).To(Equal("3"))
		})

		It("fails when folder creation is denied", func() {
			gatewayClient.
				EXPECT().
				Stat(mock.Anything, mock.Anything, mock.Anything).
				Return(statResponse(status.NewNotFound(context.Background(), "missing")), nil).
				Once()
			gatewayClient.
				EXPECT().
				CreateContainer(mock.Anything, mock.Anything, mock.Anything).
				Return(&storageprovider.CreateContainerResponse{Status: status.NewPermissionDenied(context.Background(), nil, "denied")}, nil).
				Once()

			_, err := drivesDriveItemService.CreatePath(context.Background(), parentID, "/a")
			var lgErr errorcode.Error
			Expect(errors.As(err, &lgErr)).To(BeTrue())
			Expect(lgErr.GetCode()).To(Equal(errorcode.AccessDenied))
		})
	})
})

var _ = Describe("DrivesDriveItemApi", func() {
	var (
		drivesDriveItemProvider *mocks.DrivesDriveItemProvider
		baseGraphProvider       *mocks.BaseGraphProvider
		drivesDriveItemApi      svc.DrivesDriveItemApi
		rCTX                    *chi.Context
	)

	BeforeEach(func() {
		logger := log.NewLogger()

		baseGraphProvider = mocks.NewBaseGraphProvider(GinkgoT())

		drivesDriveItemProvider = mocks.NewDrivesDriveItemProvider(GinkgoT())
		publicBaseURL, err := url.Parse("https://localhost:9200")
		Expect(err).ToNot(HaveOccurred())
		api, err := svc.NewDrivesDriveItemApi(drivesDriveItemProvider, baseGraphProvider, publicBaseURL, logger)
		Expect(err).ToNot(HaveOccurred())

		drivesDriveItemApi = api

		rCTX = chi.NewRouteContext()
	})

	failOnInvalidDriveIDOrItemID := func(handler http.HandlerFunc) {
		It("fails on invalid itemID or driveID", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "invalid")
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			handler(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrInvalidDriveIDOrItemID.Error()))
		})
	}

	failOninvalidDriveItemBody := func(handler http.HandlerFunc) {
		It("fails if the request body is not a valid DriveItem", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			handler(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrInvalidRequestBody.Error()))
		})
	}

	failOnNonShareJailDriveID := func(handler http.HandlerFunc) {
		It("fails on non share jail driveID", func() {
			rCTX.URLParams.Add("driveID", "1$2")
			rCTX.URLParams.Add("itemID", "1$2!3")
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			handler(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrNotAShareJail.Error()))
		})
	}

	Describe("DeleteDriveItem", func() {
		failOnInvalidDriveIDOrItemID(drivesDriveItemApi.DeleteDriveItem)

		failOnNonShareJailDriveID(drivesDriveItemApi.DeleteDriveItem)

		It("fails if unmounting the share fails with the general error", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemProvider.
				EXPECT().
				UnmountShare(mock.Anything, mock.Anything).
				Return(errors.New("some error")).
				Once()

			drivesDriveItemApi.DeleteDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal("generalException: some error"))
		})

		It("successfully unmounts the share", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemProvider.
				EXPECT().
				UnmountShare(mock.Anything, mock.Anything).
				Return(nil).
				Once()

			drivesDriveItemApi.DeleteDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
		})
	})

	Describe("UpdateDriveItem", func() {
		failOnInvalidDriveIDOrItemID(drivesDriveItemApi.UpdateDriveItem)

		failOnNonShareJailDriveID(drivesDriveItemApi.UpdateDriveItem)

		failOninvalidDriveItemBody(drivesDriveItemApi.UpdateDriveItem)

		It("fails if retrieving the share fails", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{})
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemProvider.
				EXPECT().
				GetShare(mock.Anything, mock.Anything).
				Return(nil, errors.New("some error")).
				Once()

			drivesDriveItemApi.UpdateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrNoShares.Error()))
		})

		It("fails if retrieving the shares fail", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{})
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemProvider.
				EXPECT().
				GetShare(mock.Anything, mock.Anything).
				Return(nil, nil).
				Once()

			drivesDriveItemProvider.
				EXPECT().
				GetSharesForResource(mock.Anything, mock.Anything, mock.Anything).
				Return(nil, errors.New("some error")).
				Once()

			drivesDriveItemApi.UpdateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrNoShares.Error()))
		})

		It("fails if updating the share fails", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{})
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemProvider.
				EXPECT().
				GetShare(mock.Anything, mock.Anything).
				Return(nil, nil).
				Once()

			drivesDriveItemProvider.
				EXPECT().
				GetSharesForResource(mock.Anything, mock.Anything, mock.Anything).
				Return(nil, nil).
				Once()

			drivesDriveItemProvider.
				EXPECT().
				UpdateShares(mock.Anything, mock.Anything, mock.Anything).
				Return(nil, errors.New("some error")).
				Once()

			drivesDriveItemApi.UpdateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrUpdateShares.Error()))
		})

		var _ = Describe("UpdateDriveItem", func() {
			var (
				driveItemJson []byte
				w             *httptest.ResponseRecorder
			)

			BeforeEach(func() {
				rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
				rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

				w = httptest.NewRecorder()

				dJson, err := json.Marshal(libregraph.DriveItem{
					UIHidden: conversions.ToPointer(true),
				})
				Expect(err).ToNot(HaveOccurred())

				driveItemJson = dJson

				drivesDriveItemProvider.
					EXPECT().
					GetShare(mock.Anything, mock.Anything).
					Return(nil, nil).
					Once()

				drivesDriveItemProvider.
					EXPECT().
					GetSharesForResource(mock.Anything, mock.Anything, mock.Anything).
					Return([]*collaborationv1beta1.ReceivedShare{}, nil).
					Once()

				drivesDriveItemProvider.
					EXPECT().
					UpdateShares(mock.Anything, mock.Anything, mock.Anything).
					Return([]*collaborationv1beta1.ReceivedShare{}, nil).
					Once()
			})

			It("fails if share to drive conversion reports an error", func() {
				baseGraphProvider.
					EXPECT().
					CS3ReceivedSharesToDriveItems(mock.Anything, mock.Anything).
					Return(nil, errors.New("some error")).
					Once()

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
					WithContext(
						context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
					)

				drivesDriveItemApi.UpdateDriveItem(w, r)
				Expect(w.Code).To(Equal(http.StatusBadRequest))

				jsonData := gjson.Get(w.Body.String(), "error")
				Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrDriveItemConversion.Error()))
			})
			It("fails if share to drive conversion returns more or less than 1 drive item", func() {
				baseGraphProvider.
					EXPECT().
					CS3ReceivedSharesToDriveItems(mock.Anything, mock.Anything).
					Return(nil, nil).
					Once()

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
					WithContext(
						context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
					)

				drivesDriveItemApi.UpdateDriveItem(w, r)
				Expect(w.Code).To(Equal(http.StatusBadRequest))

				jsonData := gjson.Get(w.Body.String(), "error")
				Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrDriveItemConversion.Error()))
			})
		})

		It("successfully updates the share", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				UIHidden: conversions.ToPointer(true),
			})
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			share := &collaborationv1beta1.ReceivedShare{
				Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{
					OpaqueId: "123",
				}},
			}

			drivesDriveItemProvider.
				EXPECT().
				GetShare(mock.Anything, mock.Anything).
				Return(nil, nil).
				Once()

			drivesDriveItemProvider.
				EXPECT().
				GetSharesForResource(mock.Anything, mock.Anything, mock.Anything).
				Return([]*collaborationv1beta1.ReceivedShare{share}, nil).
				Once()

			drivesDriveItemProvider.
				EXPECT().
				UpdateShares(mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, shares []*collaborationv1beta1.ReceivedShare, closure svc.UpdateShareClosure) ([]*collaborationv1beta1.ReceivedShare, error) {
					updateReceivedShareRequest := &collaborationv1beta1.UpdateReceivedShareRequest{
						Share: &collaborationv1beta1.ReceivedShare{
							Share: &collaborationv1beta1.Share{
								Id: share.GetShare().GetId(),
							},
						},
						UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{}},
					}

					closure(share, updateReceivedShareRequest)

					Expect(shares).To(HaveLen(1))
					Expect(updateReceivedShareRequest.GetShare().GetHidden()).To(BeTrue())
					Expect(updateReceivedShareRequest.GetUpdateMask().GetPaths()).To(HaveLen(1))
					Expect(updateReceivedShareRequest.GetUpdateMask().GetPaths()).To(ContainElements("hidden"))

					return shares, nil
				}).
				Once()

			baseGraphProvider.
				EXPECT().
				CS3ReceivedSharesToDriveItems(mock.Anything, mock.Anything).
				Return([]libregraph.DriveItem{{}}, nil).
				Once()

			drivesDriveItemApi.UpdateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("GetDriveItem", func() {
		BeforeEach(func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")
		})

		failOnInvalidDriveIDOrItemID(drivesDriveItemApi.GetDriveItem)

		failOnNonShareJailDriveID(drivesDriveItemApi.GetDriveItem)

		It("fails if retrieving the share fails", func() {
			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{})
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemProvider.
				EXPECT().
				GetShare(mock.Anything, mock.Anything).
				Return(nil, errors.New("some error")).
				Once()

			drivesDriveItemApi.GetDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrNoShares.Error()))
		})

		It("fails if retrieving the shares fail", func() {
			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{})
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemProvider.
				EXPECT().
				GetShare(mock.Anything, mock.Anything).
				Return(nil, nil).
				Once()

			drivesDriveItemProvider.
				EXPECT().
				GetSharesForResource(mock.Anything, mock.Anything, mock.Anything).
				Return(nil, errors.New("some error")).
				Once()

			drivesDriveItemApi.GetDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrNoShares.Error()))
		})

		Describe("returning the share", func() {
			var (
				w *httptest.ResponseRecorder
				r *http.Request
			)

			BeforeEach(func() {
				w = httptest.NewRecorder()

				driveItemJson, _ := json.Marshal(libregraph.DriveItem{
					UIHidden: conversions.ToPointer(true),
				})

				r = httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(driveItemJson)).
					WithContext(
						context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
					)

				share := &collaborationv1beta1.ReceivedShare{
					Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{
						OpaqueId: "123",
					}},
				}

				drivesDriveItemProvider.
					EXPECT().
					GetShare(mock.Anything, mock.Anything).
					Return(nil, nil).
					Once()

				drivesDriveItemProvider.
					EXPECT().
					GetSharesForResource(mock.Anything, mock.Anything, mock.Anything).
					Return([]*collaborationv1beta1.ReceivedShare{share}, nil).
					Once()
			})

			It("fails if more that one share found", func() {
				baseGraphProvider.
					EXPECT().
					CS3ReceivedSharesToDriveItems(mock.Anything, mock.Anything).
					Return([]libregraph.DriveItem{{}, {}}, nil).
					Once()

				drivesDriveItemApi.GetDriveItem(w, r)
				Expect(w.Code).To(Equal(http.StatusBadRequest))

				jsonData := gjson.Get(w.Body.String(), "error")
				Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrDriveItemConversion.Error()))
			})

			It("fails if no shares are found", func() {
				baseGraphProvider.
					EXPECT().
					CS3ReceivedSharesToDriveItems(mock.Anything, mock.Anything).
					Return(nil, errors.New("some error")).
					Once()

				drivesDriveItemApi.GetDriveItem(w, r)
				Expect(w.Code).To(Equal(http.StatusBadRequest))

				jsonData := gjson.Get(w.Body.String(), "error")
				Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrDriveItemConversion.Error()))
			})
		})

		It("successfully returns the share", func() {
			w := httptest.NewRecorder()

			driveItemJson, _ := json.Marshal(libregraph.DriveItem{
				UIHidden: conversions.ToPointer(true),
			})

			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			share := &collaborationv1beta1.ReceivedShare{
				Share: &collaborationv1beta1.Share{Id: &collaborationv1beta1.ShareId{
					OpaqueId: "123",
				}},
			}

			drivesDriveItemProvider.
				EXPECT().
				GetShare(mock.Anything, mock.Anything).
				Return(nil, nil).
				Once()

			drivesDriveItemProvider.
				EXPECT().
				GetSharesForResource(mock.Anything, mock.Anything, mock.Anything).
				Return([]*collaborationv1beta1.ReceivedShare{share}, nil).
				Once()

			baseGraphProvider.
				EXPECT().
				CS3ReceivedSharesToDriveItems(mock.Anything, mock.Anything).
				Return([]libregraph.DriveItem{{}}, nil).
				Once()

			drivesDriveItemApi.GetDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("CreateDriveItem", func() {
		It("fails without a driveID", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrInvalidDriveIDOrItemID.Error()))
		})

		failOninvalidDriveItemBody(drivesDriveItemApi.CreateDriveItem)

		It("fails on non share jail driveID when mounting a share", func() {
			rCTX.URLParams.Add("driveID", "1$2")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				RemoteItem: &libregraph.RemoteItem{
					Id: conversions.ToPointer("123"),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrNotAShareJail.Error()))
		})

		It("fails if not exactly one facet is set", func() {
			rCTX.URLParams.Add("driveID", "1$2")

			for _, driveItem := range []libregraph.DriveItem{
				{},
				{Folder: &libregraph.Folder{}, File: &libregraph.OpenGraphFile{}},
				{Folder: &libregraph.Folder{}, RemoteItem: &libregraph.RemoteItem{}},
			} {
				w := httptest.NewRecorder()

				driveItemJson, err := json.Marshal(driveItem)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
					WithContext(
						context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
					)

				drivesDriveItemApi.CreateDriveItem(w, r)
				Expect(w.Code).To(Equal(http.StatusBadRequest))

				jsonData := gjson.Get(w.Body.String(), "error")
				Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrExactlyOneFacet.Error()))
			}
		})

		It("fails on invalid request body id", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				RemoteItem: &libregraph.RemoteItem{},
			})
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrInvalidID.Error()))
		})

		It("fails if mounting the share fails with the general error", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				RemoteItem: &libregraph.RemoteItem{
					Id: conversions.ToPointer("123"),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			drivesDriveItemProvider.
				EXPECT().
				MountShare(mock.Anything, mock.Anything, mock.Anything).
				Return(nil, errors.New("some error")).
				Once()

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal("generalException: some error"))
		})

		It("fails if drive item conversion fails", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				RemoteItem: &libregraph.RemoteItem{
					Id: conversions.ToPointer("123"),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			drivesDriveItemProvider.
				EXPECT().
				MountShare(mock.Anything, mock.Anything, mock.Anything).
				Return(nil, nil)

			baseGraphProvider.
				EXPECT().
				CS3ReceivedSharesToDriveItems(mock.Anything, mock.Anything).
				Return(nil, errors.New("some error")).
				Once()

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrDriveItemConversion.Error()))

			//
			baseGraphProvider.
				EXPECT().
				CS3ReceivedSharesToDriveItems(mock.Anything, mock.Anything).
				Return(nil, nil).
				Once()

			r = httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData = gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrDriveItemConversion.Error()))
		})

		It("successfully creates the drive item", func() {
			rCTX.URLParams.Add("driveID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668")
			rCTX.URLParams.Add("itemID", "a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668!1")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				RemoteItem: &libregraph.RemoteItem{
					Id: conversions.ToPointer("123"),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			drivesDriveItemProvider.
				EXPECT().
				MountShare(mock.Anything, mock.Anything, mock.Anything).
				Return(nil, nil)

			baseGraphProvider.
				EXPECT().
				CS3ReceivedSharesToDriveItems(mock.Anything, mock.Anything).
				Return([]libregraph.DriveItem{{}}, nil).
				Once()

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusCreated))
		})

		It("fails on an invalid conflictBehavior", func() {
			rCTX.URLParams.Add("driveID", "1$2")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				Name:   conversions.ToPointer("New Folder"),
				Folder: &libregraph.Folder{},
			})
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/?%40libre.graph.conflictBehavior=rename", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			jsonData := gjson.Get(w.Body.String(), "error")
			Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrInvalidConflictBehavior.Error()))
		})

		It("successfully creates a folder at the drive root", func() {
			rCTX.URLParams.Add("driveID", "1$2")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				Name:   conversions.ToPointer("New Folder"),
				Folder: &libregraph.Folder{},
			})
			Expect(err).ToNot(HaveOccurred())

			drivesDriveItemProvider.
				EXPECT().
				CreateChild(mock.Anything, mock.MatchedBy(func(id *storageprovider.ResourceId) bool {
					return id.GetStorageId() == "1" && id.GetSpaceId() == "2" && id.GetOpaqueId() == "2"
				}), "New Folder", true, false).
				Return(&storageprovider.ResourceInfo{
					Id:   &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"},
					Path: "./New Folder",
					Type: storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				}, nil).
				Once()

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusCreated))
			Expect(gjson.Get(w.Body.String(), "name").String()).To(Equal("New Folder"))
			Expect(gjson.Get(w.Body.String(), "folder").Exists()).To(BeTrue())
		})

		It("passes replace on conflictBehavior=replace", func() {
			rCTX.URLParams.Add("driveID", "1$2")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				Name: conversions.ToPointer("file.txt"),
				File: &libregraph.OpenGraphFile{},
			})
			Expect(err).ToNot(HaveOccurred())

			drivesDriveItemProvider.
				EXPECT().
				CreateChild(mock.Anything, mock.Anything, "file.txt", false, true).
				Return(&storageprovider.ResourceInfo{
					Id:   &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"},
					Path: "./file.txt",
					Type: storageprovider.ResourceType_RESOURCE_TYPE_FILE,
				}, nil).
				Once()

			r := httptest.NewRequest(http.MethodPost, "/?%40libre.graph.conflictBehavior=replace", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusCreated))
		})
	})

	Describe("CreateChildDriveItem", func() {
		failOnInvalidDriveIDOrItemID(drivesDriveItemApi.CreateChildDriveItem)

		failOninvalidDriveItemBody(drivesDriveItemApi.CreateChildDriveItem)

		It("fails if not exactly one of folder and file is set", func() {
			rCTX.URLParams.Add("driveID", "1$2")
			rCTX.URLParams.Add("itemID", "1$2!3")

			for _, driveItem := range []libregraph.DriveItem{
				{},
				{Folder: &libregraph.Folder{}, File: &libregraph.OpenGraphFile{}},
				{RemoteItem: &libregraph.RemoteItem{}},
			} {
				w := httptest.NewRecorder()

				driveItemJson, err := json.Marshal(driveItem)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
					WithContext(
						context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
					)

				drivesDriveItemApi.CreateChildDriveItem(w, r)
				Expect(w.Code).To(Equal(http.StatusBadRequest))

				jsonData := gjson.Get(w.Body.String(), "error")
				Expect(jsonData.Get("code").String() + ": " + jsonData.Get("message").String()).To(Equal(svc.ErrExactlyOneChildFacet.Error()))
			}
		})

		It("renders the service error", func() {
			rCTX.URLParams.Add("driveID", "1$2")
			rCTX.URLParams.Add("itemID", "1$2!3")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				Name:   conversions.ToPointer("New Folder"),
				Folder: &libregraph.Folder{},
			})
			Expect(err).ToNot(HaveOccurred())

			drivesDriveItemProvider.
				EXPECT().
				CreateChild(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, errorcode.New(errorcode.NameAlreadyExists, "already exists")).
				Once()

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateChildDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusConflict))
		})

		It("successfully creates a file below the parent", func() {
			rCTX.URLParams.Add("driveID", "1$2")
			rCTX.URLParams.Add("itemID", "1$2!3")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				Name: conversions.ToPointer("file.txt"),
				File: &libregraph.OpenGraphFile{},
			})
			Expect(err).ToNot(HaveOccurred())

			drivesDriveItemProvider.
				EXPECT().
				CreateChild(mock.Anything, mock.MatchedBy(func(id *storageprovider.ResourceId) bool {
					return id.GetStorageId() == "1" && id.GetSpaceId() == "2" && id.GetOpaqueId() == "3"
				}), "file.txt", false, false).
				Return(&storageprovider.ResourceInfo{
					Id:       &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "4"},
					Path:     "./file.txt",
					Type:     storageprovider.ResourceType_RESOURCE_TYPE_FILE,
					MimeType: "text/plain",
				}, nil).
				Once()

			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			drivesDriveItemApi.CreateChildDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusCreated))
			Expect(gjson.Get(w.Body.String(), "name").String()).To(Equal("file.txt"))
			Expect(gjson.Get(w.Body.String(), "file.mimeType").String()).To(Equal("text/plain"))
		})

		It("creates the unresolved parent path from the context first", func() {
			rCTX.URLParams.Add("driveID", "1$2")
			rCTX.URLParams.Add("itemID", "1$2!2")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				Name:   conversions.ToPointer("New Folder"),
				Folder: &libregraph.Folder{},
			})
			Expect(err).ToNot(HaveOccurred())

			parentID := &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"}
			drivesDriveItemProvider.
				EXPECT().
				CreatePath(mock.Anything, mock.MatchedBy(func(id *storageprovider.ResourceId) bool {
					return id.GetOpaqueId() == "2"
				}), "/a/b").
				Return(parentID, nil).
				Once()
			drivesDriveItemProvider.
				EXPECT().
				CreateChild(mock.Anything, parentID, "New Folder", true, false).
				Return(&storageprovider.ResourceInfo{
					Id:   &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "4"},
					Path: "./New Folder",
					Type: storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
				}, nil).
				Once()

			ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rCTX)
			ctx = context.WithValue(ctx, graphm.CreateParentsContextKey, "/a/b")
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(driveItemJson)).WithContext(ctx)

			drivesDriveItemApi.CreateChildDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusCreated))
		})

		It("does not create parent folders when the request is invalid", func() {
			rCTX.URLParams.Add("driveID", "1$2")
			rCTX.URLParams.Add("itemID", "1$2!2")

			w := httptest.NewRecorder()

			driveItemJson, err := json.Marshal(libregraph.DriveItem{
				Name:   conversions.ToPointer("New Folder"),
				Folder: &libregraph.Folder{},
			})
			Expect(err).ToNot(HaveOccurred())

			ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rCTX)
			ctx = context.WithValue(ctx, graphm.CreateParentsContextKey, "/a/b")
			r := httptest.NewRequest(http.MethodPost, "/?%40libre.graph.conflictBehavior=rename", bytes.NewBuffer(driveItemJson)).WithContext(ctx)

			// no CreatePath / CreateChild expectations: the provider must not be called
			drivesDriveItemApi.CreateChildDriveItem(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})
})
