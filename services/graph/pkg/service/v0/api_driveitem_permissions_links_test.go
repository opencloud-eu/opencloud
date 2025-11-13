package svc_test

import (
	"context"
	"errors"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity/cache"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/linktype"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("createLinkTests", func() {
	var (
		svc             service.DriveItemPermissionsService
		driveItemId     *provider.ResourceId
		ctx             context.Context
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector *mocks.Selectable[gateway.GatewayAPIClient]
		currentUser     = &userpb.User{
			Id: &userpb.UserId{
				OpaqueId: "user",
			},
		}
	)
	const (
		ViewerLinkString = "Viewer Link"
	)

	BeforeEach(func() {
		var err error
		logger := log.NewLogger()
		gatewayClient = cs3mocks.NewGatewayAPIClient(GinkgoT())

		gatewaySelector = mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
		gatewaySelector.On("Next").Return(gatewayClient, nil)

		cache := cache.NewIdentityCache(cache.IdentityCacheWithGatewaySelector(gatewaySelector))

		cfg := defaults.FullDefaultConfig()
		svc, err = service.NewDriveItemPermissionsService(logger, gatewaySelector, cache, cfg)
		Expect(err).ToNot(HaveOccurred())
		driveItemId = &provider.ResourceId{
			StorageId: "1",
			SpaceId:   "2",
			OpaqueId:  "3",
		}
		ctx = revactx.ContextSetUser(context.Background(), currentUser)
	})

	Describe("CreateLink", func() {
		var (
			driveItemCreateLink libregraph.DriveItemCreateLink
			statResponse        *provider.StatResponse
			createLinkResponse  *link.CreatePublicShareResponse
		)

		BeforeEach(func() {
			driveItemCreateLink = libregraph.DriveItemCreateLink{
				Type:                nil,
				ExpirationDateTime:  nil,
				Password:            nil,
				DisplayName:         nil,
				LibreGraphQuickLink: nil,
			}

			statResponse = &provider.StatResponse{
				Status: status.NewOK(ctx),
				Info:   &provider.ResourceInfo{Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER},
			}

			createLinkResponse = &link.CreatePublicShareResponse{
				Status: status.NewOK(ctx),
			}

			linkType, err := libregraph.NewSharingLinkTypeFromValue("view")
			Expect(err).ToNot(HaveOccurred())
			driveItemCreateLink.Type = linkType
			driveItemCreateLink.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
			permissions, err := linktype.CS3ResourcePermissionsFromSharingLink(driveItemCreateLink, provider.ResourceType_RESOURCE_TYPE_CONTAINER)
			Expect(err).ToNot(HaveOccurred())
			createLinkResponse.Share = &link.PublicShare{
				Id:                &link.PublicShareId{OpaqueId: "123"},
				Expiration:        utils.TimeToTS(*driveItemCreateLink.ExpirationDateTime),
				PasswordProtected: false,
				DisplayName:       ViewerLinkString,
				Token:             "SomeGOODCoffee",
				Permissions:       &link.PublicSharePermissions{Permissions: permissions},
			}

		})

		// Public Shares / "links" in graph terms
		It("creates a public link as expected (happy path)", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("CreatePublicShare", mock.Anything, mock.Anything).Return(createLinkResponse, nil)
			perm, err := svc.CreateLink(context.Background(), driveItemId, driveItemCreateLink)
			Expect(err).ToNot(HaveOccurred())
			Expect(perm.GetId()).To(Equal("123"))
			Expect(perm.GetExpirationDateTime().Unix()).To(Equal(driveItemCreateLink.ExpirationDateTime.Unix()))
			Expect(perm.GetHasPassword()).To(Equal(false))
			Expect(perm.GetLink().LibreGraphDisplayName).To(Equal(libregraph.PtrString(ViewerLinkString)))
			link := perm.GetLink()
			respLinkType := link.GetType()
			expected, err := libregraph.NewSharingLinkTypeFromValue("view")
			Expect(err).ToNot(HaveOccurred())
			Expect(&respLinkType).To(Equal(expected))
		})

		It("handles a failing CreateLink", func() {
			statResponse.Info = &provider.ResourceInfo{Type: provider.ResourceType_RESOURCE_TYPE_FILE}
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("CreatePublicShare", mock.Anything, mock.Anything).Return(createLinkResponse, nil)

			linkType, err := libregraph.NewSharingLinkTypeFromValue("edit")
			Expect(err).ToNot(HaveOccurred())
			driveItemCreateLink.Type = linkType
			permissions, err := linktype.CS3ResourcePermissionsFromSharingLink(driveItemCreateLink, provider.ResourceType_RESOURCE_TYPE_CONTAINER)
			Expect(err).ToNot(HaveOccurred())
			createLinkResponse.Status = status.NewInternal(ctx, "transport error")
			createLinkResponse.Share = &link.PublicShare{
				Id:          &link.PublicShareId{OpaqueId: "123"},
				Permissions: &link.PublicSharePermissions{Permissions: permissions},
			}

			perm, err := svc.CreateLink(context.Background(), driveItemId, driveItemCreateLink)
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, "transport error")))
			Expect(perm).To(BeZero())
		})

		It("fails when the stat returns access denied", func() {
			err := errors.New("no permission to stat the file")
			statResponse.Status = status.NewPermissionDenied(ctx, err, err.Error())
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			perm, err := svc.CreateLink(context.Background(), driveItemId, driveItemCreateLink)
			Expect(err).To(MatchError(errorcode.New(errorcode.AccessDenied, "no permission to stat the file")))
			Expect(perm).To(BeZero())
		})

		It("fails when the stat returns resource is locked", func() {
			err := errors.New("the resource is locked")
			statResponse.Status = status.NewLocked(ctx, err.Error())
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			perm, err := svc.CreateLink(context.Background(), driveItemId, driveItemCreateLink)
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemIsLocked, "the resource is locked")))
			Expect(perm).To(BeZero())
		})

		It("succeeds when the link type mapping is not successful", func() {
			// we need to send a valid link type
			linkType, err := libregraph.NewSharingLinkTypeFromValue("edit")
			Expect(err).ToNot(HaveOccurred())
			driveItemCreateLink.Type = linkType
			permissions := &provider.ResourcePermissions{
				CreateContainer:    true,
				InitiateFileUpload: true,
				Move:               true,
			}
			// return different permissions which do not match a link type
			createLinkResponse.Share = &link.PublicShare{
				Id:                &link.PublicShareId{OpaqueId: "123"},
				Expiration:        utils.TimeToTS(*driveItemCreateLink.ExpirationDateTime),
				PasswordProtected: false,
				DisplayName:       ViewerLinkString,
				Token:             "SomeGOODCoffee",
				Permissions:       &link.PublicSharePermissions{Permissions: permissions},
			}
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("CreatePublicShare", mock.Anything, mock.Anything).Return(createLinkResponse, nil)

			perm, err := svc.CreateLink(context.Background(), driveItemId, driveItemCreateLink)
			Expect(err).ToNot(HaveOccurred())

			Expect(perm.GetId()).To(Equal("123"))
			Expect(perm.GetExpirationDateTime().Unix()).To(Equal(driveItemCreateLink.ExpirationDateTime.Unix()))
			Expect(perm.GetHasPassword()).To(Equal(false))
			Expect(perm.GetLink().LibreGraphDisplayName).To(Equal(libregraph.PtrString(ViewerLinkString)))
			respLink := perm.GetLink()
			// some conversion gymnastics
			respLinkType := respLink.GetType()
			Expect(err).ToNot(HaveOccurred())
			mockLink := libregraph.SharingLink{}
			lt, _ := linktype.SharingLinkTypeFromCS3Permissions(&link.PublicSharePermissions{Permissions: permissions})
			mockLink.Type = lt
			expectedType := mockLink.GetType()
			Expect(&respLinkType).To(Equal(&expectedType))
			libreGraphActions := perm.LibreGraphPermissionsActions
			Expect(libreGraphActions[0]).To(Equal("libre.graph/driveItem/children/create"))
			Expect(libreGraphActions[1]).To(Equal("libre.graph/driveItem/upload/create"))
			Expect(libreGraphActions[2]).To(Equal("libre.graph/driveItem/path/update"))
		})
	})
	Describe("SetLinPassword", func() {
		var (
			updatePublicShareMockResponse link.UpdatePublicShareResponse
			getPublicShareResponse        link.GetPublicShareResponse
		)

		const TestLinkName = "Test Link"

		BeforeEach(func() {
			updatePublicShareMockResponse = link.UpdatePublicShareResponse{
				Status: status.NewOK(ctx),
				Share:  &link.PublicShare{DisplayName: TestLinkName},
			}
			getPublicShareResponse = link.GetPublicShareResponse{
				Status: status.NewOK(ctx),
				Share: &link.PublicShare{
					Id: &link.PublicShareId{
						OpaqueId: "permissionid",
					},
					ResourceId: driveItemId,
					Permissions: &link.PublicSharePermissions{
						Permissions: linktype.NewViewLinkPermissionSet().GetPermissions(),
					},
					Token: "token",
				},
			}
		})

		It("updates the password on a public share", func() {
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)

			updatePublicShareMockResponse.Share.Permissions = &link.PublicSharePermissions{
				Permissions: linktype.NewViewLinkPermissionSet().Permissions,
			}
			updatePublicShareMockResponse.Share.PasswordProtected = true
			gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(&updatePublicShareMockResponse, nil)

			perm, err := svc.SetPublicLinkPassword(context.Background(), driveItemId, "permissionid", "OC123!")
			Expect(err).ToNot(HaveOccurred())
			linkType := perm.Link.GetType()
			Expect(string(linkType)).To(Equal("view"))
			Expect(perm.GetHasPassword()).To(BeTrue())
		})
	})
})
