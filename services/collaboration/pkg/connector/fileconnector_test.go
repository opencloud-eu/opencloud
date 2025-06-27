package connector_test

import (
	"context"
	"encoding/hex"
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"

	appproviderv1beta1 "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	auth "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	collabmocks "github.com/opencloud-eu/opencloud/services/collaboration/mocks"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/connector"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/connector/fileinfo"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

var _ = Describe("FileConnector", func() {
	var (
		fc              *connector.FileConnector
		ccs             *collabmocks.ContentConnectorService
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector *mocks.Selectable[gateway.GatewayAPIClient]
		cfg             *config.Config
		wopiCtx         middleware.WopiContext
	)

	BeforeEach(func() {
		cfg = &config.Config{
			Commons: &shared.Commons{
				OpenCloudURL: "https://cloud.opencloud.test",
			},
			App: config.App{
				Name:    "test",
				Product: "Microsoft",
			},
			Wopi: config.Wopi{
				WopiSrc: "https://wopi.opencloud.test",
				Secret:  "topsecret",
			},
			TokenManager: &config.TokenManager{JWTSecret: "secret"},
		}
		ccs = &collabmocks.ContentConnectorService{}

		gatewayClient = cs3mocks.NewGatewayAPIClient(GinkgoT())

		gatewaySelector = mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
		gatewaySelector.On("Next").Return(gatewayClient, nil)
		fc = connector.NewFileConnector(gatewaySelector, cfg, nil)

		wopiCtx = middleware.WopiContext{
			// a real token is needed for the PutRelativeFileSuggested tests
			// although we aren't checking anything inside the token
			AccessToken: "eyJhbGciOiJQUzI1NiIsImtpZCI6InByaXZhdGUta2V5IiwidHlwIjoiSldUIn0.eyJhdWQiOiJ3ZWIiLCJleHAiOjE3MjAwOTIyODAsImlhdCI6MTcyMDA5MTk4MCwiaXNzIjoiaHR0cHM6Ly9vY2lzLmpwLnNvbGlkZ2Vhci5wcnYiLCJqdGkiOiJmQldpN0FYaFFQdUhhaDJDV0VQVFFLcENmZ3BGbEFpTCIsImxnLmkiOnsiZG4iOiJicm8iLCJpZCI6Im93bkNsb3VkVVVJRD1mYWYxMTY0Ny03NDUxLTRiOWEtYmZmZS0zYjVkZGNjNTk3MmIiLCJ1biI6ImJyb3RhdG8ifSwibGcucCI6ImlkZW50aWZpZXItbGRhcCIsImxnLnQiOiIxIiwic2NwIjoib3BlbmlkIHByb2ZpbGUgZW1haWwiLCJzdWIiOiJjQXZ1elg4Z1hMWmRpWHgtQDFOV1RKdENQRHFVSjQ0bnQ0NkZ0RDlwNUw3dGplUEZkWk1WSjlFMzBOeDItZHVpN0hLQ0x4QWlXYUNUdGJYNTExSmNkSHcifQ.StpQpE4ipxk8Nhk6xgob1Tovbk6bcUVs5-fkej2hIoKoJKfR2OY-CiFQ3wwgEcFro8notxeVfOmxs36z_ezFeJBZRbxpSggcr77LFtQwlsWvD5AuAgLZN1otdvULehunXE_DtxRJZ1rqnsOBT03zKOZLx8Q7QTy6DeRuf1KQtCIowa9D4ymPM4TTmtQdiW2XjByO3OCLFEMVBfDFGPibR6gMnftGQ5kfiZGDTUVCauEXwE-msZVZ42QY-wFRppX_RIL1Z0p6T4dr_6_y-VM1lNYJ5-dB5c5rg_c03Xu1y_TIxs31-8--dtUyZmBVOZFk8bB9msNk-iaOEjzKeUZLymo_-2qVYvXxzNrkq1QA8luaLR6jec_CRT2P8wsB2nyebFU6_myKe34m6f8uqGhOzcOwPB4TpoxPx4ucQgo1CQJwQZHZsZ7Q6TVYZUXJdWwzzMuvJXmnn36iybw0Ub6On4sGKj3gHetjoJg8VnL-TQkBvf1iHX2ktRG3Nq2rnPrB2OTpi2rLpleWg_s8Y8FXxIgYqM0JG8kO1n5RPGMeYQG7qd6f9wdcaPIvgxCa_HsZtMr7eGcDzZtxp-NivgJOS6ode0ZAJ3wGU-AVhmyshpds3DFECcvkBcP_4dD52AXiAq9X3UVkVdNsxs_yB9P7zBcdsKsD6QDJv5gf-6DEu34",
			FileReference: &providerv1beta1.Reference{
				ResourceId: &providerv1beta1.ResourceId{
					StorageId: "abc",
					OpaqueId:  "12345",
					SpaceId:   "zzz",
				},
				Path: ".",
			},
			ViewMode: appproviderv1beta1.ViewMode_VIEW_MODE_READ_WRITE,
		}
	})

	Describe("GetLock", func() {
		It("No valid context", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := context.Background()
			response, err := fc.GetLock(ctx)
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("Get lock failed", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewInternal(ctx, "Something failed"),
			}, targetErr)

			response, err := fc.GetLock(ctx)
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Get lock failed status not ok", func() {
			// assume failure happens because the target file doesn't exists
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewNotFound(ctx, "File is missing"),
			}, nil)

			response, err := fc.GetLock(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(404))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal(""))
		})

		It("Get lock success", func() {
			// assume failure happens because the target file doesn't exists
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewOK(ctx),
				Lock: &providerv1beta1.Lock{
					LockId: "zzz999",
					Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
				},
			}, nil)

			response, err := fc.GetLock(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal("zzz999"))
		})
	})

	Describe("Lock", func() {
		Describe("Lock", func() {
			It("No valid context", func() {
				gatewaySelector.EXPECT().Next().Unset()
				ctx := context.Background()
				response, err := fc.Lock(ctx, "newLock", "")
				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
			})

			It("Empty lockId", func() {
				gatewaySelector.EXPECT().Next().Unset()
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				response, err := fc.Lock(ctx, "", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(400))
				Expect(response.Headers).To(BeNil())
			})

			It("Set lock failed", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				targetErr := errors.New("Something went wrong")
				gatewayClient.On("SetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.SetLockResponse{
					Status: status.NewInternal(ctx, "Something failed"),
				}, targetErr)

				response, err := fc.Lock(ctx, "abcdef123", "")
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(targetErr))
				Expect(response).To(BeNil())
			})

			It("Set lock success", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("SetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.SetLockResponse{
					Status: status.NewOK(ctx),
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(
						&providerv1beta1.StatResponse{
							Status: status.NewOK(ctx),
							Info: &providerv1beta1.ResourceInfo{
								Mtime: &typesv1beta1.Timestamp{
									Seconds: 12345,
									Nanos:   6789,
								},
							},
						},
						nil,
					)

				response, err := fc.Lock(ctx, "abcdef123", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(200))
				Expect(response.Headers).To(HaveLen(1))
				Expect(response.Headers[connector.HeaderWopiVersion]).To(Equal("v123456789"))
			})

			It("Set lock mismatches error getting lock", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("SetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.SetLockResponse{
					Status: status.NewFailedPrecondition(ctx, nil, "lock mismatch"),
				}, nil)

				targetErr := errors.New("Something went wrong getting the lock")
				gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
					Status: status.NewInternal(ctx, "lock mismatch"),
				}, targetErr)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "")
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(targetErr))
				Expect(response).To(BeNil())
			})

			It("Set lock mismatches", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("SetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.SetLockResponse{
					Status: status.NewFailedPrecondition(ctx, nil, "lock mismatch"),
				}, nil)

				gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
					Status: status.NewOK(ctx),
					Lock: &providerv1beta1.Lock{
						LockId: "zzz999",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(409))
				Expect(response.Headers).To(HaveLen(2))
				Expect(response.Headers[connector.HeaderWopiLock]).To(Equal("zzz999"))
				Expect(response.Headers[connector.HeaderWopiLockFailureReason]).To(Equal("Conflicting LockID"))
			})

			It("Set lock mismatches but get lock matches", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("SetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.SetLockResponse{
					Status: status.NewFailedPrecondition(ctx, nil, "lock mismatch"),
				}, nil)

				gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
					Status: status.NewOK(ctx),
					Lock: &providerv1beta1.Lock{
						LockId: "abcdef123",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{
						Status: status.NewOK(ctx),
						Info: &providerv1beta1.ResourceInfo{
							Mtime: &typesv1beta1.Timestamp{Seconds: uint64(12345), Nanos: uint32(6789)},
						},
					}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(200))
				Expect(response.Headers).To(HaveLen(1))
				Expect(response.Headers[connector.HeaderWopiVersion]).To(Equal("v123456789"))
			})

			It("Set lock mismatches but get lock doesn't return lockId", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("SetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.SetLockResponse{
					Status: status.NewFailedPrecondition(ctx, nil, "lock mismatch"),
				}, nil)

				gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
					Status: status.NewOK(ctx),
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(500))
				Expect(response.Headers).To(BeNil())
			})

			It("File not found", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("SetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.SetLockResponse{
					Status: status.NewNotFound(ctx, "file not found"),
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(404))
				Expect(response.Headers).To(BeNil())
			})

			It("Default error handling (insufficient storage)", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("SetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.SetLockResponse{
					Status: status.NewInsufficientStorage(ctx, nil, "file too big"),
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(500))
				Expect(response.Headers).To(BeNil())
			})
		})

		Describe("Unlock and relock", func() {
			It("No valid context", func() {
				gatewaySelector.EXPECT().Next().Unset()
				ctx := context.Background()
				response, err := fc.Lock(ctx, "newLock", "oldLock")
				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
			})

			It("Empty lockId", func() {
				gatewaySelector.EXPECT().Next().Unset()
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				response, err := fc.Lock(ctx, "", "oldLock")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(400))
				Expect(response.Headers).To(BeNil())
			})

			It("Refresh lock failed", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				targetErr := errors.New("Something went wrong")
				gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
					Status: status.NewInternal(ctx, "Something failed"),
				}, targetErr)

				response, err := fc.Lock(ctx, "abcdef123", "oldLock")
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(targetErr))
				Expect(response).To(BeNil())
			})

			It("Refresh lock success", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
					Status: status.NewOK(ctx),
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{
						Status: status.NewOK(ctx),
						Info: &providerv1beta1.ResourceInfo{
							Mtime: &typesv1beta1.Timestamp{Seconds: uint64(12345), Nanos: uint32(6789)},
						},
					}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "oldLock")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(200))
				Expect(response.Headers).To(HaveLen(1))
				Expect(response.Headers[connector.HeaderWopiVersion]).To(Equal("v123456789"))
			})

			It("Refresh lock mismatches error getting lock", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
					Status: status.NewConflict(ctx, nil, "lock mismatch"),
				}, nil)

				targetErr := errors.New("Something went wrong getting the lock")
				gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
					Status: status.NewInternal(ctx, "lock mismatch"),
				}, targetErr)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "112233")
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(targetErr))
				Expect(response).To(BeNil())
			})

			It("Refresh lock mismatches", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
					Status: status.NewConflict(ctx, nil, "lock mismatch"),
				}, nil)

				gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
					Status: status.NewOK(ctx),
					Lock: &providerv1beta1.Lock{
						LockId: "zzz999",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "112233")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(409))
				Expect(response.Headers[connector.HeaderWopiLock]).To(Equal("zzz999"))
			})

			It("Refresh lock mismatches but get lock matches", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
					Status: status.NewConflict(ctx, nil, "lock mismatch"),
				}, nil)

				gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
					Status: status.NewOK(ctx),
					Lock: &providerv1beta1.Lock{
						LockId: "abcdef123",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{
						Status: status.NewOK(ctx),
						Info: &providerv1beta1.ResourceInfo{
							Mtime: &typesv1beta1.Timestamp{Seconds: uint64(12345), Nanos: uint32(6789)},
						},
					}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "112233")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(200))
				Expect(response.Headers).To(HaveLen(1))
				Expect(response.Headers[connector.HeaderWopiVersion]).To(Equal("v123456789"))
			})

			It("Refresh lock mismatches but get lock doesn't return lockId", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
					Status: status.NewConflict(ctx, nil, "lock mismatch"),
				}, nil)

				gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
					Status: status.NewOK(ctx),
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "112233")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(500))
				Expect(response.Headers).To(BeNil())
			})

			It("File not found", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
					Status: status.NewNotFound(ctx, "file not found"),
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "112233")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(404))
				Expect(response.Headers).To(BeNil())
			})

			It("Default error handling (insufficient storage)", func() {
				ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

				gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
					Status: status.NewInsufficientStorage(ctx, nil, "file too big"),
				}, nil)

				gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
					Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

				response, err := fc.Lock(ctx, "abcdef123", "112233")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Status).To(Equal(500))
				Expect(response.Headers).To(BeNil())
			})
		})
	})

	Describe("RefreshLock", func() {
		It("No valid context", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := context.Background()

			response, err := fc.RefreshLock(ctx, "")
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("Empty lockId", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			response, err := fc.RefreshLock(ctx, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(400))
			Expect(response.Headers).To(BeNil())
		})

		It("Refresh lock fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
				Status: status.NewConflict(ctx, nil, "lock mismatch"),
			}, targetErr)

			response, err := fc.RefreshLock(ctx, "abcdef123")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Refresh lock success", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
				Status: status.NewOK(ctx),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{
					Status: status.NewOK(ctx),
					Info: &providerv1beta1.ResourceInfo{
						Mtime: &typesv1beta1.Timestamp{Seconds: uint64(12345), Nanos: uint32(6789)},
					},
				}, nil)

			response, err := fc.RefreshLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers).To(HaveLen(1))
			Expect(response.Headers[connector.HeaderWopiVersion]).To(Equal("v123456789"))
		})

		It("Refresh lock file not found", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
				Status: status.NewNotFound(ctx, "file not found"),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.RefreshLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(404))
			Expect(response.Headers).To(BeNil())
		})

		It("Refresh lock mismatch and get lock fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
				Status: status.NewConflict(ctx, nil, "lock mismatch"),
			}, nil)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewConflict(ctx, nil, "lock mismatch"),
			}, targetErr)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.RefreshLock(ctx, "abcdef123")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Refresh lock mismatch and get lock status not ok", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
				Status: status.NewConflict(ctx, nil, "lock mismatch"),
			}, nil)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewInternal(ctx, "lock mismatch"),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.RefreshLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
		})

		It("Refresh lock mismatch and no lock", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
				Status: status.NewConflict(ctx, nil, "lock mismatch"),
			}, nil)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewOK(ctx),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.RefreshLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(409))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal(""))
		})

		It("Refresh lock mismatch", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
				Status: status.NewConflict(ctx, nil, "lock mismatch"),
			}, nil)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewOK(ctx),
				Lock: &providerv1beta1.Lock{
					LockId: "zzz999",
					Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
				},
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.RefreshLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(409))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal("zzz999"))
		})

		It("Default error handling (insufficient storage)", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("RefreshLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.RefreshLockResponse{
				Status: status.NewInsufficientStorage(ctx, nil, "file too big"),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.RefreshLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
		})
	})

	Describe("Unlock", func() {
		It("No valid context", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := context.Background()

			response, err := fc.UnLock(ctx, "")
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("Empty lockId", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			response, err := fc.UnLock(ctx, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(400))
			Expect(response.Headers).To(BeNil())
		})

		It("Unlock fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("Unlock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.UnlockResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, targetErr)

			response, err := fc.UnLock(ctx, "abcdef123")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Unlock success", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Unlock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.UnlockResponse{
				Status: status.NewOK(ctx),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{
					Status: status.NewOK(ctx),
					Info: &providerv1beta1.ResourceInfo{
						Mtime: &typesv1beta1.Timestamp{Seconds: uint64(12345), Nanos: uint32(6789)},
					},
				}, nil)

			response, err := fc.UnLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers).To(HaveLen(1))
			Expect(response.Headers[connector.HeaderWopiVersion]).To(Equal("v123456789"))
		})

		It("Unlock file isn't locked", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Unlock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.UnlockResponse{
				Status: status.NewConflict(ctx, nil, "lock mismatch"),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{
					Status: status.NewOK(ctx),
					Info: &providerv1beta1.ResourceInfo{
						Mtime: &typesv1beta1.Timestamp{Seconds: uint64(12345), Nanos: uint32(6789)},
					},
				}, nil)

			response, err := fc.UnLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(409))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal(""))
		})

		It("Unlock mismatch get lock fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Unlock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.UnlockResponse{
				Status: status.NewLocked(ctx, "lock mismatch"),
			}, nil)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, targetErr)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.UnLock(ctx, "abcdef123")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Unlock mismatch get lock status not ok", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Unlock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.UnlockResponse{
				Status: status.NewLocked(ctx, "lock mismatch"),
			}, nil)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.UnLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
		})

		It("Unlock mismatch get lock doesn't return lock", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Unlock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.UnlockResponse{
				Status: status.NewLocked(ctx, "lock mismatch"),
			}, nil)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewOK(ctx),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.UnLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(409))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal(""))
		})

		It("Unlock mismatch", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Unlock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.UnlockResponse{
				Status: status.NewLocked(ctx, "lock mismatch"),
			}, nil)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewOK(ctx),
				Lock: &providerv1beta1.Lock{
					LockId: "zzz999",
					Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
				},
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.UnLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(409))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal("zzz999"))
		})

		It("Default error handling (insufficient storage)", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Unlock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.UnlockResponse{
				Status: status.NewInsufficientStorage(ctx, nil, "file too big"),
			}, nil)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).
				Return(&providerv1beta1.StatResponse{Status: status.NewOK(ctx)}, nil)

			response, err := fc.UnLock(ctx, "abcdef123")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
		})
	})

	Describe("PutRelativeFileSuggested", func() {
		It("No valid context", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := context.Background()
			stream := strings.NewReader("This is the content of a file")
			response, err := fc.PutRelativeFileSuggested(ctx, ccs, stream, int64(stream.Len()), "newFile.txt")
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("Stat fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, targetErr)

			stream := strings.NewReader("This is the content of a file")
			response, err := fc.PutRelativeFileSuggested(ctx, ccs, stream, int64(stream.Len()), "newFile.txt")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Stat fails status not ok", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			stream := strings.NewReader("This is the content of a file")
			response, err := fc.PutRelativeFileSuggested(ctx, ccs, stream, int64(stream.Len()), "newFile.txt")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
			Expect(response.Body).To(BeNil())
		})

		It("PutRelativeFileSuggested success", func() {
			// requested filename is "newDocument.docx" so we'll write that
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			stream := strings.NewReader("This is the content of a file")

			stat1ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				return statReq.Ref.ResourceId == wopiCtx.FileReference.ResourceId
			})
			gatewayClient.On("Stat", mock.Anything, stat1ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/file.docx",
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			ccs.On("PutFile", mock.Anything, stream, int64(stream.Len()), "").Times(1).Return(connector.NewResponse(200), nil)

			stat2ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				if statReq.Ref.ResourceId.StorageId == "storageid" &&
					statReq.Ref.ResourceId.OpaqueId == "opaqueid" &&
					statReq.Ref.ResourceId.SpaceId == "spaceid" &&
					statReq.Ref.Path == "./newDocument.docx" {
					return true
				}
				return false
			})
			gatewayClient.On("Stat", mock.Anything, stat2ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/newDocument.docx",
					Id: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid_newDoc",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			response, err := fc.PutRelativeFileSuggested(ctx, ccs, stream, int64(stream.Len()), "newDocument.docx")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers).To(BeNil())
			rBody := response.Body.(map[string]interface{})
			Expect(rBody["Name"]).To(Equal("newDocument.docx"))
			Expect(rBody["Url"]).To(HavePrefix("https://wopi.opencloud.test/wopi/files/")) // skip checking the actual reference
			Expect(rBody["HostEditUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/newDocument.docx?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=write"))
			Expect(rBody["HostViewUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/newDocument.docx?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=view"))
		})

		It("PutRelativeFileSuggested success only extension", func() {
			// requested file is ".pdf" so we'll change the "file.docx" to "file.pdf"
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			stream := strings.NewReader("This is the content of a file")

			stat1ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				return statReq.Ref.ResourceId == wopiCtx.FileReference.ResourceId
			})
			gatewayClient.On("Stat", mock.Anything, stat1ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/file.docx",
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			ccs.On("PutFile", mock.Anything, stream, int64(stream.Len()), "").Times(1).Return(connector.NewResponse(200), nil)

			stat2ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				if statReq.Ref.ResourceId.StorageId == "storageid" &&
					statReq.Ref.ResourceId.OpaqueId == "opaqueid" &&
					statReq.Ref.ResourceId.SpaceId == "spaceid" &&
					statReq.Ref.Path == "./file.pdf" {
					return true
				}
				return false
			})
			gatewayClient.On("Stat", mock.Anything, stat2ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/file.pdf",
					Id: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid_newDoc",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			response, err := fc.PutRelativeFileSuggested(ctx, ccs, stream, int64(stream.Len()), ".pdf")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers).To(BeNil())
			rBody := response.Body.(map[string]interface{})
			Expect(rBody["Name"]).To(Equal("file.pdf"))
			Expect(rBody["Url"]).To(HavePrefix("https://wopi.opencloud.test/wopi/files/")) // skip checking the actual reference
			Expect(rBody["HostEditUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/file.pdf?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=write"))
			Expect(rBody["HostViewUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/file.pdf?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=view"))
		})

		It("PutRelativeFileSuggested success conflict", func() {
			// requested file is ".pdf", but "file.pdf" exists as target file (we get a conflict)
			// so we change the "file.docx" to "<base64> file.pdf" file, where the <base64> is a
			// sequence of base64 chars containing alphanumeric chars plus "-" and "_" (the char
			// sequence is based on time, so we can't be too specific)
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			stream := strings.NewReader("This is the content of a file")

			stat1ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				return statReq.Ref.ResourceId == wopiCtx.FileReference.ResourceId
			})
			gatewayClient.On("Stat", mock.Anything, stat1ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/file.docx",
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			// first call will fail with conflict, second call succeeds.
			// we're only interested on whether the file is locked or not, the actual lockID is irrelevant
			ccs.On("PutFile", mock.Anything, stream, int64(stream.Len()), "").Times(1).Return(connector.NewResponse(409), nil).Once()
			ccs.On("PutFile", mock.Anything, stream, int64(stream.Len()), "").Times(1).Return(connector.NewResponse(200), nil).Once()

			newFilePath := new(string)
			stat2ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				r := regexp.MustCompile(`^\./[a-zA-Z0-9_-]+ file\.pdf$`)
				if statReq.Ref.ResourceId.StorageId == "storageid" &&
					statReq.Ref.ResourceId.OpaqueId == "opaqueid" &&
					statReq.Ref.ResourceId.SpaceId == "spaceid" &&
					r.MatchString(statReq.Ref.Path) {
					*newFilePath = statReq.Ref.Path
					return true
				}
				return false
			})

			gatewayClient.EXPECT().Stat(mock.Anything, stat2ParamMatcher).
				RunAndReturn(func(ctx context.Context, req *providerv1beta1.StatRequest, opts ...grpc.CallOption) (*providerv1beta1.StatResponse, error) {
					return &providerv1beta1.StatResponse{
						Status: status.NewOK(ctx),
						Info: &providerv1beta1.ResourceInfo{
							Path: path.Join("/personal/path/to", path.Base(req.GetRef().GetPath())),
							Id: &providerv1beta1.ResourceId{
								StorageId: "storageid",
								OpaqueId:  "opaqueid_newDoc",
								SpaceId:   "spaceid",
							},
							Lock: &providerv1beta1.Lock{
								LockId: "zzz999",
								Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
							},
						},
					}, nil
				})

			response, err := fc.PutRelativeFileSuggested(ctx, ccs, stream, int64(stream.Len()), ".pdf")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers).To(BeNil())
			rBody := response.Body.(map[string]interface{})
			Expect(rBody["Name"]).To(MatchRegexp(`[a-zA-Z0-9_-] file\.pdf`))
			Expect(rBody["Url"]).To(HavePrefix("https://wopi.opencloud.test/wopi/files/")) // skip checking the actual reference
			Expect(rBody["HostEditUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/" + url.PathEscape(path.Base(*newFilePath)) + "?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=write"))
			Expect(rBody["HostViewUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/" + url.PathEscape(path.Base(*newFilePath)) + "?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=view"))
		})

		It("PutRelativeFileSuggested put file fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			stream := strings.NewReader("This is the content of a file")

			stat1ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				return statReq.Ref.ResourceId == wopiCtx.FileReference.ResourceId
			})
			gatewayClient.On("Stat", mock.Anything, stat1ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/file.docx",
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			ccs.On("PutFile", mock.Anything, stream, int64(stream.Len()), "").Times(1).Return(connector.NewResponse(500), nil)

			response, err := fc.PutRelativeFileSuggested(ctx, ccs, stream, int64(stream.Len()), ".pdf")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
			Expect(response.Body).To(BeNil())
		})
	})

	Describe("PutRelativeFileRelative", func() {
		It("No valid context", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := context.Background()
			stream := strings.NewReader("This is the content of a file")
			response, err := fc.PutRelativeFileRelative(ctx, ccs, stream, int64(stream.Len()), "newFile.txt")
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("Stat fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, targetErr)

			stream := strings.NewReader("This is the content of a file")
			response, err := fc.PutRelativeFileRelative(ctx, ccs, stream, int64(stream.Len()), "newFile.txt")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Stat fails status not ok", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			stream := strings.NewReader("This is the content of a file")
			response, err := fc.PutRelativeFileRelative(ctx, ccs, stream, int64(stream.Len()), "newFile.txt")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
			Expect(response.Body).To(BeNil())
		})

		It("PutRelativeFileRelative success", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			stream := strings.NewReader("This is the content of a file")

			stat1ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				return statReq.Ref.ResourceId == wopiCtx.FileReference.ResourceId
			})
			gatewayClient.On("Stat", mock.Anything, stat1ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/file.docx",
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			ccs.On("PutFile", mock.Anything, stream, int64(stream.Len()), "").Times(1).Return(connector.NewResponse(200), nil)

			stat2ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				if statReq.Ref.ResourceId.StorageId == "storageid" &&
					statReq.Ref.ResourceId.OpaqueId == "opaqueid" &&
					statReq.Ref.ResourceId.SpaceId == "spaceid" &&
					statReq.Ref.Path == "./newDocument.docx" {
					return true
				}
				return false
			})
			gatewayClient.On("Stat", mock.Anything, stat2ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/newDocument.docx",
					Id: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid_newDoc",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			response, err := fc.PutRelativeFileRelative(ctx, ccs, stream, int64(stream.Len()), "newDocument.docx")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers).To(BeNil())
			rBody := response.Body.(map[string]interface{})
			Expect(rBody["Name"]).To(Equal("newDocument.docx"))
			Expect(rBody["Url"]).To(HavePrefix("https://wopi.opencloud.test/wopi/files/")) // skip checking the actual reference
			Expect(rBody["HostEditUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/newDocument.docx?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=write"))
			Expect(rBody["HostViewUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/newDocument.docx?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=view"))
		})

		It("PutRelativeFileRelative conflict", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			stream := strings.NewReader("This is the content of a file")

			stat1ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				return statReq.Ref.ResourceId == wopiCtx.FileReference.ResourceId
			})
			gatewayClient.On("Stat", mock.Anything, stat1ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/file.docx",
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			ccs.On("PutFile", mock.Anything, stream, int64(stream.Len()), "").Times(1).Return(connector.NewResponseWithLock(409, "zzz999"), nil)

			stat2ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				if statReq.Ref.ResourceId.StorageId == "storageid" &&
					statReq.Ref.ResourceId.OpaqueId == "opaqueid" &&
					statReq.Ref.ResourceId.SpaceId == "spaceid" &&
					statReq.Ref.Path == "./convFile.pdf" {
					return true
				}
				return false
			})
			gatewayClient.On("Stat", mock.Anything, stat2ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/convFile.pdf",
					Id: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid_newDoc",
						SpaceId:   "spaceid",
					},
					Lock: &providerv1beta1.Lock{
						LockId: "zzz999",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				},
			}, nil)

			response, err := fc.PutRelativeFileRelative(ctx, ccs, stream, int64(stream.Len()), "convFile.pdf")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(409))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal("zzz999"))
			Expect(response.Headers[connector.HeaderWopiValidRT]).To(MatchRegexp(`[a-zA-Z0-9_-] convFile\.pdf`))
			rBody := response.Body.(map[string]interface{})
			Expect(rBody["Name"]).To(Equal("convFile.pdf"))
			Expect(rBody["Url"]).To(HavePrefix("https://wopi.opencloud.test/wopi/files/")) // skip checking the actual reference
			Expect(rBody["HostEditUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/convFile.pdf?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=write"))
			Expect(rBody["HostViewUrl"]).To(Equal("https://cloud.opencloud.test/external-test/personal/path/to/convFile.pdf?fileId=storageid%24spaceid%21opaqueid_newDoc&view_mode=view"))
		})

		It("PutRelativeFileRelative put file fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			stream := strings.NewReader("This is the content of a file")

			stat1ParamMatcher := mock.MatchedBy(func(statReq *providerv1beta1.StatRequest) bool {
				return statReq.Ref.ResourceId == wopiCtx.FileReference.ResourceId
			})
			gatewayClient.On("Stat", mock.Anything, stat1ParamMatcher).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Path: "/personal/path/to/file.docx",
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			ccs.On("PutFile", mock.Anything, stream, int64(stream.Len()), "").Times(1).Return(nil, connector.NewConnectorError(500, "Something happened"))

			response, err := fc.PutRelativeFileRelative(ctx, ccs, stream, int64(stream.Len()), "convFile.pdf")
			targetErr := connector.NewConnectorError(500, "Something happened")
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})
	})

	Describe("DeleteFile", func() {
		It("No valid context", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := context.Background()
			response, err := fc.DeleteFile(ctx, "lock")
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("Delete fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("Delete", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.DeleteResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, targetErr)

			gatewayClient.EXPECT().Stat(mock.Anything, mock.Anything).Unset()

			response, err := fc.DeleteFile(ctx, "newlock")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Delete fails status not ok, get lock fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Delete", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.DeleteResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, targetErr)

			response, err := fc.DeleteFile(ctx, "newlock")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Delete fails file missing", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Delete", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.DeleteResponse{
				Status: status.NewNotFound(ctx, "something failed"),
			}, nil)

			response, err := fc.DeleteFile(ctx, "newlock")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(404))
			Expect(response.Headers).To(BeNil())
		})

		It("Delete fails status not ok, get lock not ok", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Delete", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.DeleteResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			response, err := fc.DeleteFile(ctx, "newlock")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
		})

		It("Delete fails, file locked", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Delete", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.DeleteResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewOK(ctx),
				Lock: &providerv1beta1.Lock{
					LockId: "zzz999",
					Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
				},
			}, nil)

			response, err := fc.DeleteFile(ctx, "newlock")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(409))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal("zzz999"))
		})

		It("Delete fails, file not locked", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Delete", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.DeleteResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			gatewayClient.On("GetLock", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.GetLockResponse{
				Status: status.NewOK(ctx),
			}, nil)

			response, err := fc.DeleteFile(ctx, "newlock")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
		})

		It("Delete success", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Delete", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.DeleteResponse{
				Status: status.NewOK(ctx),
			}, nil)

			response, err := fc.DeleteFile(ctx, "newlock")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers).To(BeNil())
		})
	})

	Describe("RenameFile", func() {
		It("No valid context", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := context.Background()
			response, err := fc.RenameFile(ctx, "lockid", "newFile.doc")
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("Stat fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, targetErr)

			response, err := fc.RenameFile(ctx, "lockid", "newFile.doc")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Stat fails status not ok", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			response, err := fc.RenameFile(ctx, "lockid", "newFile.doc")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
			Expect(response.Body).To(BeNil())
		})

		It("Rename failed", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Lock: &providerv1beta1.Lock{
						LockId: "zzz999",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				},
			}, nil)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("Move", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.MoveResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, targetErr)

			response, err := fc.RenameFile(ctx, "lockid", "newFile.doc")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Rename failed status not ok", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Lock: &providerv1beta1.Lock{
						LockId: "zzz999",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				},
			}, nil)

			gatewayClient.On("Move", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.MoveResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			response, err := fc.RenameFile(ctx, "lockid", "newFile.doc")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Headers).To(BeNil())
			Expect(response.Body).To(BeNil())
		})

		It("Rename conflict", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Lock: &providerv1beta1.Lock{
						LockId: "zzz999",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				},
			}, nil)

			gatewayClient.On("Move", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.MoveResponse{
				Status: status.NewLocked(ctx, "lock mismatch"),
			}, nil)

			response, err := fc.RenameFile(ctx, "lockid", "newFile.doc")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(409))
			Expect(response.Headers[connector.HeaderWopiLock]).To(Equal("zzz999"))
			Expect(response.Body).To(BeNil())
		})

		It("Rename already exists", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Lock: &providerv1beta1.Lock{
						LockId: "zzz999",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				},
			}, nil)

			moveParamMatcher := mock.MatchedBy(func(moveReq *providerv1beta1.MoveRequest) bool {
				if moveReq.Destination.Path == "./newFile.doc" &&
					moveReq.LockId == "zzz999" {
					return true
				}
				return false
			})
			gatewayClient.On("Move", mock.Anything, moveParamMatcher).Times(1).Return(&providerv1beta1.MoveResponse{
				Status: status.NewAlreadyExists(ctx, nil, "target already exists"),
			}, nil).Once()

			move2ParamMatcher := mock.MatchedBy(func(moveReq *providerv1beta1.MoveRequest) bool {
				r := regexp.MustCompile(`^\./[a-zA-Z0-9_-]+ newFile\.doc$`)
				if r.MatchString(moveReq.Destination.Path) &&
					moveReq.LockId == "zzz999" {
					return true
				}
				return false
			})
			gatewayClient.On("Move", mock.Anything, move2ParamMatcher).Times(1).Return(&providerv1beta1.MoveResponse{
				Status: status.NewOK(ctx),
			}, nil).Once()

			response, err := fc.RenameFile(ctx, "zzz999", "newFile.doc")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers).To(BeNil())
			rBody := response.Body.(map[string]interface{})
			Expect(rBody["Name"]).To(MatchRegexp(`^[a-zA-Z0-9_-]+ newFile$`))
		})

		It("Success", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Lock: &providerv1beta1.Lock{
						LockId: "zzz999",
						Type:   providerv1beta1.LockType_LOCK_TYPE_WRITE,
					},
				},
			}, nil)

			moveParamMatcher := mock.MatchedBy(func(moveReq *providerv1beta1.MoveRequest) bool {
				if moveReq.Destination.Path == "./newFile.doc" &&
					moveReq.LockId == "zzz999" {
					return true
				}
				return false
			})
			gatewayClient.On("Move", mock.Anything, moveParamMatcher).Times(1).Return(&providerv1beta1.MoveResponse{
				Status: status.NewOK(ctx),
			}, nil).Once()

			response, err := fc.RenameFile(ctx, "zzz999", "newFile.doc")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Headers).To(BeNil())
			rBody := response.Body.(map[string]interface{})
			Expect(rBody["Name"]).To(Equal("newFile"))
		})
	})

	Describe("CheckFileInfo", func() {
		It("No valid context", func() {
			gatewaySelector.EXPECT().Next().Unset()
			ctx := context.Background()
			response, err := fc.CheckFileInfo(ctx)
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("Stat fails", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			targetErr := errors.New("Something went wrong")
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, targetErr)

			response, err := fc.CheckFileInfo(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(targetErr))
			Expect(response).To(BeNil())
		})

		It("Stat fails status not ok", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewInternal(ctx, "something failed"),
			}, nil)

			response, err := fc.CheckFileInfo(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(500))
			Expect(response.Body).To(BeNil())
		})

		It("Stat fails status not found", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewNotFound(ctx, "something not found"),
			}, nil)

			response, err := fc.CheckFileInfo(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(404))
			Expect(response.Body).To(BeNil())
		})

		It("Stat success", func() {
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)
			u := &userv1beta1.User{
				Id: &userv1beta1.UserId{
					Idp:      "customIdp",
					OpaqueId: "admin",
				},
				DisplayName: "Pet Shaft",
			}
			ctx = ctxpkg.ContextSetUser(ctx, u)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Owner: &userv1beta1.UserId{
						Idp:      "customIdp",
						OpaqueId: "aabbcc",
						Type:     userv1beta1.UserType_USER_TYPE_PRIMARY,
					},
					Size: uint64(998877),
					Mtime: &typesv1beta1.Timestamp{
						Seconds: uint64(16273849),
					},
					Path: "/path/to/test.txt",
					Id: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "parentopaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			expectedFileInfo := &fileinfo.Microsoft{
				OwnerID:                    "61616262636340637573746f6d496470", // hex of aabbcc@customIdp
				Size:                       int64(998877),
				Version:                    "v162738490",
				BaseFileName:               "test.txt",
				BreadcrumbDocName:          "test.txt",
				BreadcrumbFolderName:       "/path/to",
				BreadcrumbFolderURL:        "https://cloud.opencloud.test/f/storageid$spaceid%21parentopaqueid",
				UserCanNotWriteRelative:    false,
				SupportsExtendedLockLength: true,
				SupportsGetLock:            true,
				SupportsLocks:              true,
				SupportsUpdate:             true,
				SupportsDeleteFile:         true,
				SupportsRename:             true,
				UserCanWrite:               true,
				UserCanRename:              true,
				UserID:                     "61646d696e40637573746f6d496470", // hex of admin@customIdp
				UserFriendlyName:           "Pet Shaft",
				FileSharingURL:             "https://cloud.opencloud.test/f/storageid$spaceid%21opaqueid?details=sharing",
				FileVersionURL:             "https://cloud.opencloud.test/f/storageid$spaceid%21opaqueid?details=versions",
				HostEditURL:                "https://cloud.opencloud.test/external-test/path/to/test.txt?fileId=storageid%24spaceid%21opaqueid&view_mode=write",
				HostViewURL:                "https://cloud.opencloud.test/external-test/path/to/test.txt?fileId=storageid%24spaceid%21opaqueid&view_mode=view",
			}

			response, err := fc.CheckFileInfo(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Body.(*fileinfo.Microsoft)).To(Equal(expectedFileInfo))
		})

		It("Stat success guests", func() {
			// add user's opaque to include public-share-role
			u := &userv1beta1.User{}
			u.Opaque = &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"public-share-role": {
						Decoder: "plain",
						Value:   []byte("viewer"),
					},
				},
			}
			// change view mode to view only
			wopiCtx.ViewMode = appproviderv1beta1.ViewMode_VIEW_MODE_VIEW_ONLY

			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)
			ctx = ctxpkg.ContextSetUser(ctx, u)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Owner: &userv1beta1.UserId{
						Idp:      "customIdp",
						OpaqueId: "aabbcc",
						Type:     userv1beta1.UserType_USER_TYPE_PRIMARY,
					},
					Size: uint64(998877),
					Mtime: &typesv1beta1.Timestamp{
						Seconds: uint64(16273849),
					},
					Path: "/path/to/test.txt",
					Id: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "parentopaqueid",
						SpaceId:   "spaceid",
					},
					// Other properties aren't used for now.
				},
			}, nil)

			// change wopi app provider
			cfg.App.Name = "Collabora"
			cfg.App.Product = "Collabora"

			expectedFileInfo := &fileinfo.Collabora{
				OwnerID:                 "61616262636340637573746f6d496470", // hex of aabbcc@customIdp
				Size:                    int64(998877),
				BaseFileName:            "test.txt",
				UserCanNotWriteRelative: false,
				DisableExport:           true,
				DisableCopy:             true,
				DisablePrint:            true,
				UserID:                  "guest-zzz000",
				UserFriendlyName:        "guest zzz000",
				EnableOwnerTermination:  true,
				SupportsLocks:           true,
				SupportsRename:          true,
				UserCanRename:           false,
				BreadcrumbDocName:       "test.txt",
				PostMessageOrigin:       "https://cloud.opencloud.test",
			}

			response, err := fc.CheckFileInfo(ctx)

			// UserID and UserFriendlyName have random Ids generated which are impossible to guess
			// Check both separately
			Expect(response.Body.(*fileinfo.Collabora).UserID).To(HavePrefix(hex.EncodeToString([]byte("guest-"))))
			Expect(response.Body.(*fileinfo.Collabora).UserFriendlyName).To(HavePrefix("Guest "))
			// overwrite UserID and UserFriendlyName here for easier matching
			response.Body.(*fileinfo.Collabora).UserID = "guest-zzz000"
			response.Body.(*fileinfo.Collabora).UserFriendlyName = "guest zzz000"

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Body.(*fileinfo.Collabora)).To(Equal(expectedFileInfo))
		})

		It("Stat success guests", func() {
			ResourceId := &providerv1beta1.ResourceId{
				StorageId: "storageid",
				OpaqueId:  "opaqueid",
				SpaceId:   "spaceid",
			}

			// add user's opaque to include public-share-role
			u := &userv1beta1.User{}
			u.Opaque = &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"public-share-role": {
						Decoder: "plain",
						Value:   []byte("viewer"),
					},
				},
			}
			// Create a new "scope share" to only expose the required fields `ResourceId` and `Token` to the scope.
			scopeShare := &link.PublicShare{ResourceId: ResourceId, Token: "ABC123"}
			val, err := utils.MarshalProtoV1ToJSON(scopeShare)
			Expect(err).ToNot(HaveOccurred())

			scopes := map[string]*auth.Scope{
				"publicshare:ABC123": {
					Resource: &typesv1beta1.OpaqueEntry{
						Decoder: "json",
						Value:   val,
					},
					Role: auth.Role(appproviderv1beta1.ViewMode_VIEW_MODE_VIEW_ONLY), //
				},
			}
			// change view mode to view only
			wopiCtx.ViewMode = appproviderv1beta1.ViewMode_VIEW_MODE_VIEW_ONLY

			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)
			ctx = ctxpkg.ContextSetUser(ctx, u)
			ctx = ctxpkg.ContextSetScopes(ctx, scopes)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Owner: &userv1beta1.UserId{
						Idp:      "customIdp",
						OpaqueId: "aabbcc",
						Type:     userv1beta1.UserType_USER_TYPE_PRIMARY,
					},
					Size: uint64(998877),
					Mtime: &typesv1beta1.Timestamp{
						Seconds: uint64(16273849),
					},
					Path: "/path/to/test.txt",
					Id:   ResourceId,
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "parentopaqueid",
						SpaceId:   "spaceid",
					},
					// Other properties aren't used for now.
				},
			}, nil)

			// change wopi app provider
			cfg.App.Name = "OnlyOffice"
			cfg.App.Product = "OnlyOffice"

			expectedFileInfo := &fileinfo.OnlyOffice{
				Version:                 "v162738490",
				BaseFileName:            "test.txt",
				BreadcrumbDocName:       "test.txt",
				BreadcrumbFolderName:    "/path/to",
				BreadcrumbFolderURL:     "https://cloud.opencloud.test/s/ABC123",
				DisablePrint:            true,
				UserCanNotWriteRelative: false,
				SupportsLocks:           true,
				SupportsUpdate:          true,
				SupportsRename:          true,
				IsAnonymousUser:         true,
				UserCanRename:           false,
				UserCanReview:           false,
				UserCanWrite:            false,
				EnableInsertRemoteImage: false,
				UserID:                  "guest-zzz000",
				UserFriendlyName:        "guest zzz000",
				FileSharingURL:          "https://cloud.opencloud.test/f/storageid$spaceid%21opaqueid?details=sharing",
				FileVersionURL:          "https://cloud.opencloud.test/f/storageid$spaceid%21opaqueid?details=versions",
				HostEditURL:             "https://cloud.opencloud.test/external-onlyoffice/path/to/test.txt?fileId=storageid%24spaceid%21opaqueid&view_mode=write",
				PostMessageOrigin:       "https://cloud.opencloud.test",
			}

			response, err := fc.CheckFileInfo(ctx)

			// UserID and UserFriendlyName have random Ids generated which are impossible to guess
			// Check both separately
			Expect(response.Body.(*fileinfo.OnlyOffice).UserID).To(HavePrefix(hex.EncodeToString([]byte("guest-"))))
			Expect(response.Body.(*fileinfo.OnlyOffice).UserFriendlyName).To(HavePrefix("Guest "))
			// overwrite UserID and UserFriendlyName here for easier matching
			response.Body.(*fileinfo.OnlyOffice).UserID = "guest-zzz000"
			response.Body.(*fileinfo.OnlyOffice).UserFriendlyName = "guest zzz000"
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Body.(*fileinfo.OnlyOffice)).To(Equal(expectedFileInfo))
		})

		It("Stat success authenticated user", func() {
			// change view mode to view only
			wopiCtx.ViewMode = appproviderv1beta1.ViewMode_VIEW_MODE_VIEW_ONLY

			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)
			u := &userv1beta1.User{
				Id: &userv1beta1.UserId{
					Idp:      "example.com",
					OpaqueId: "aabbcc",
					Type:     userv1beta1.UserType_USER_TYPE_PRIMARY,
				},
				DisplayName: "Pet Shaft",
				Mail:        "shaft@example.com",
			}
			ctx = ctxpkg.ContextSetUser(ctx, u)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Owner: &userv1beta1.UserId{
						Idp:      "customIdp",
						OpaqueId: "aabbcc",
						Type:     userv1beta1.UserType_USER_TYPE_PRIMARY,
					},
					Size: uint64(998877),
					Mtime: &typesv1beta1.Timestamp{
						Seconds: uint64(16273849),
					},
					Path: "/path/to/test.txt",
					Id: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "parentopaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			// change wopi app provider
			cfg.App.Name = "Collabora"
			cfg.App.Product = "Collabora"

			expectedFileInfo := &fileinfo.Collabora{
				OwnerID:                 "61616262636340637573746f6d496470", // hex of aabbcc@customIdp
				Size:                    int64(998877),
				BaseFileName:            "test.txt",
				UserCanNotWriteRelative: false,
				DisableExport:           true,
				DisableCopy:             true,
				DisablePrint:            true,
				UserID:                  hex.EncodeToString([]byte("aabbcc@example.com")),
				UserFriendlyName:        "Pet Shaft",
				EnableOwnerTermination:  true,
				WatermarkText:           "Pet Shaft shaft@example.com",
				SupportsLocks:           true,
				SupportsRename:          true,
				UserCanRename:           false,
				BreadcrumbDocName:       "test.txt",
				PostMessageOrigin:       "https://cloud.opencloud.test",
			}

			response, err := fc.CheckFileInfo(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))
			Expect(response.Body.(*fileinfo.Collabora)).To(Equal(expectedFileInfo))
		})
		It("Stat success with template", func() {
			wopiCtx.TemplateReference = &providerv1beta1.Reference{
				ResourceId: &providerv1beta1.ResourceId{
					StorageId: "storageid",
					OpaqueId:  "opaqueid2",
					SpaceId:   "spaceid",
				},
			}
			ctx := middleware.WopiContextToCtx(context.Background(), wopiCtx)
			u := &userv1beta1.User{
				Id: &userv1beta1.UserId{
					Idp:      "customIdp",
					OpaqueId: "admin",
				},
				DisplayName: "Pet Shaft",
			}
			ctx = ctxpkg.ContextSetUser(ctx, u)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Times(1).Return(&providerv1beta1.StatResponse{
				Status: status.NewOK(ctx),
				Info: &providerv1beta1.ResourceInfo{
					Owner: &userv1beta1.UserId{
						Idp:      "customIdp",
						OpaqueId: "aabbcc",
						Type:     userv1beta1.UserType_USER_TYPE_PRIMARY,
					},
					Size: uint64(0),
					Mtime: &typesv1beta1.Timestamp{
						Seconds: uint64(16273849),
					},
					Path: "/path/to/test.txt",
					Id: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "opaqueid",
						SpaceId:   "spaceid",
					},
					ParentId: &providerv1beta1.ResourceId{
						StorageId: "storageid",
						OpaqueId:  "parentopaqueid",
						SpaceId:   "spaceid",
					},
				},
			}, nil)

			expectedFileInfo := &fileinfo.OnlyOffice{
				Version:                 "v162738490",
				BaseFileName:            "test.txt",
				BreadcrumbDocName:       "test.txt",
				BreadcrumbFolderName:    "/path/to",
				BreadcrumbFolderURL:     "https://cloud.opencloud.test/f/storageid$spaceid%21parentopaqueid",
				UserCanNotWriteRelative: false,
				SupportsLocks:           true,
				SupportsUpdate:          true,
				SupportsRename:          true,
				UserCanWrite:            true,
				UserCanRename:           true,
				UserID:                  "61646d696e40637573746f6d496470", // hex of admin@customIdp
				UserFriendlyName:        "Pet Shaft",
				FileSharingURL:          "https://cloud.opencloud.test/f/storageid$spaceid%21opaqueid?details=sharing",
				FileVersionURL:          "https://cloud.opencloud.test/f/storageid$spaceid%21opaqueid?details=versions",
				HostEditURL:             "https://cloud.opencloud.test/external-onlyoffice/path/to/test.txt?fileId=storageid%24spaceid%21opaqueid&view_mode=write",
				PostMessageOrigin:       "https://cloud.opencloud.test",
			}

			// change wopi app provider
			cfg.App.Name = "OnlyOffice"
			cfg.App.Product = "OnlyOffice"

			response, err := fc.CheckFileInfo(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Status).To(Equal(200))

			returnedFileInfo := response.Body.(*fileinfo.OnlyOffice)
			templateSource := returnedFileInfo.TemplateSource
			expectedTemplateSource := "https://wopi.opencloud.test/wopi/templates/a340d017568d0d579ee061a9ac02109e32fb07082d35c40aa175864303bd9107?access_token="

			// take TemplateSource out of the response for easier comparison
			returnedFileInfo.TemplateSource = ""
			Expect(returnedFileInfo).To(Equal(expectedFileInfo))
			// check if the template source is correct
			// the url is using a generated access token which always has a new ttl
			// so we can't compare the whole url
			Expect(templateSource).To(HavePrefix(expectedTemplateSource))
		})
	})
})
