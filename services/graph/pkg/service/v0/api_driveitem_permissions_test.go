package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/tidwall/gjson"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	roleconversions "github.com/opencloud-eu/reva/v2/pkg/conversions"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	identitycache "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/cache"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/linktype"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	svc "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

// logEntries decodes the JSON lines written by a zerolog logger.
func logEntries(buf *bytes.Buffer) []map[string]any {
	var entries []map[string]any
	dec := json.NewDecoder(buf)
	for dec.More() {
		entry := map[string]any{}
		Expect(dec.Decode(&entry)).To(Succeed())
		entries = append(entries, entry)
	}
	return entries
}

// rawLog re-encodes the log entries, so a sentinel can be searched for in every field.
func rawLog(entries []map[string]any) string {
	raw, err := json.Marshal(entries)
	Expect(err).ToNot(HaveOccurred())
	return string(raw)
}

// newSpanRecorder returns a span recorder and a TracerProvider recording into it.
func newSpanRecorder() (*tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	DeferCleanup(func() {
		Expect(tp.Shutdown(context.Background())).To(Succeed())
	})
	return sr, tp
}

// endedSpan returns the single ended span with the given name. It checks that the span carries no
// events and none of the sentinels in its status or attributes.
func endedSpan(sr *tracetest.SpanRecorder, name string, sentinels ...string) sdktrace.ReadOnlySpan {
	var spans []sdktrace.ReadOnlySpan
	for _, s := range sr.Ended() {
		if s.Name() == name {
			spans = append(spans, s)
		}
	}
	Expect(spans).To(HaveLen(1))
	span := spans[0]
	Expect(span.Events()).To(BeEmpty())
	raw, err := json.Marshal([]any{span.Status(), span.Attributes()})
	Expect(err).ToNot(HaveOccurred())
	for _, sentinel := range sentinels {
		Expect(string(raw)).ToNot(ContainSubstring(sentinel))
	}
	return span
}

// gatewayHelperEntry matches the error entry a share helper logs when it cannot select a gateway.
func gatewayHelperEntry(message, errorText string) gomegatypes.GomegaMatcher {
	return SatisfyAll(
		HaveKeyWithValue("level", "error"),
		HaveKeyWithValue("message", message),
		HaveKeyWithValue("error", errorText),
	)
}

// entriesAtLevel returns the log entries logged at the given level.
func entriesAtLevel(entries []map[string]any, level string) []map[string]any {
	var filtered []map[string]any
	for _, entry := range entries {
		if entry["level"] == level {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

var _ = Describe("DriveItemPermissionsService", func() {
	var (
		driveItemPermissionsService svc.DriveItemPermissionsService
		gatewayClient               *cs3mocks.GatewayAPIClient
		gatewaySelector             *mocks.Selectable[gateway.GatewayAPIClient]
		getUserResponse             *userpb.GetUserResponse
		listPublicSharesResponse    *link.ListPublicSharesResponse
		listSpacesResponse          *provider.ListStorageSpacesResponse
		currentUser                 = &userpb.User{
			Id: &userpb.UserId{
				OpaqueId: "user",
			},
		}
		cache        identitycache.IdentityCache
		statResponse *provider.StatResponse
		driveItemId  *provider.ResourceId
		ctx          context.Context
		cfg          *config.Config
	)

	BeforeEach(func() {
		logger := log.NewLogger()
		gatewayClient = cs3mocks.NewGatewayAPIClient(GinkgoT())

		gatewaySelector = mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
		gatewaySelector.On("Next").Return(gatewayClient, nil)

		cache = identitycache.NewIdentityCache(identitycache.IdentityCacheWithGatewaySelector(gatewaySelector))

		cfg = defaults.FullDefaultConfig()
		service, err := svc.NewDriveItemPermissionsService(logger, gatewaySelector, cache, cfg, nil)
		Expect(err).ToNot(HaveOccurred())
		driveItemPermissionsService = service
		ctx = revactx.ContextSetUser(context.Background(), currentUser)
		statResponse = &provider.StatResponse{
			Status: status.NewOK(ctx),
		}
		getUserResponse = &userpb.GetUserResponse{
			Status: status.NewOK(ctx),
			User: &userpb.User{
				Id:          &userpb.UserId{OpaqueId: "1"},
				DisplayName: "Cem Kaner",
			},
		}
		listPublicSharesResponse = &link.ListPublicSharesResponse{
			Status: status.NewOK(ctx),
		}

		driveItemId = &provider.ResourceId{
			StorageId: "1",
			SpaceId:   "2",
			OpaqueId:  "3",
		}

	})

	// newLoggedService returns a service logging into a new buffer. It uses the mocks and the
	// config of the current spec, so call it after the spec has adjusted them.
	newLoggedService := func() (svc.DriveItemPermissionsService, *bytes.Buffer) {
		logBuf := &bytes.Buffer{}
		service, err := svc.NewDriveItemPermissionsService(log.Logger{Logger: zerolog.New(logBuf)}, gatewaySelector, cache, cfg, nil)
		Expect(err).ToNot(HaveOccurred())
		return service, logBuf
	}

	// A failing gateway selection is an unexpected failure of every operation, logged with its error text.
	// Where a probing helper selects the gateway, the helper also logs the selector's error.
	DescribeTable("reports a gateway selection failure once, as error log correlated with the request and as error span",
		func(name string, call func(context.Context, svc.DriveItemPermissionsService) error, ids map[string]string, errorText gomegatypes.GomegaMatcher, helperEntries int) {
			gatewayErr := errors.New("no gateway available at 10.0.0.42")
			gatewaySelector.EXPECT().Next().Unset()
			gatewaySelector.EXPECT().Next().Return(nil, gatewayErr)
			var logBuf bytes.Buffer
			sr, tp := newSpanRecorder()
			service, err := svc.NewDriveItemPermissionsService(log.Logger{Logger: zerolog.New(&logBuf)}, gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			Expect(call(context.WithValue(ctx, middleware.RequestIDKey, "request-42"), service)).To(MatchError(gatewayErr))

			// the error text is added once per entry: decoding into a map would hide a duplicate key
			for line := range strings.SplitSeq(strings.TrimSpace(logBuf.String()), "\n") {
				Expect(strings.Count(line, `"error":`)).To(BeNumerically("<=", 1), line)
			}
			entries := logEntries(&logBuf)
			// the operation and the helpers log one entry each, all correlated with the request
			const helperMessage = "selecting gatewaySelector failed"
			var correlated, helpers []map[string]any
			for _, entry := range entriesAtLevel(entries, "error") {
				if entry["message"] == helperMessage {
					helpers = append(helpers, entry)
				} else {
					correlated = append(correlated, entry)
				}
			}
			Expect(correlated).To(HaveLen(1))
			Expect(helpers).To(HaveLen(helperEntries))
			for _, entry := range helpers {
				Expect(entry).To(SatisfyAll(
					gatewayHelperEntry(helperMessage, gatewayErr.Error()),
					HaveKeyWithValue("request-id", "request-42"),
					HaveKeyWithValue("permissionID", "permissionid"),
				))
			}
			expected := []gomegatypes.GomegaMatcher{
				HaveKeyWithValue("request-id", "request-42"),
				HaveKeyWithValue("errorCode", "generalException"),
				errorText,
			}
			for k, v := range ids {
				expected = append(expected, HaveKeyWithValue(k, v))
			}
			Expect(correlated[0]).To(SatisfyAll(expected...))
			// no serialized requests, responses or gateway clients
			for _, payloadKey := range []string{"stat.res", "gateway.res", "Body", "body", "request", "response"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}

			// the span carries the error code only, the error message is not recorded
			Expect(sr.Ended()).To(HaveLen(1))
			span := endedSpan(sr, name, "10.0.0.42")
			Expect(span.Status().Code).To(Equal(codes.Error))
			Expect(span.Status().Description).To(Equal("generalException"))
			Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "generalException")))
		},
		Entry("Invite", "Invite", func(ctx context.Context, s svc.DriveItemPermissionsService) error {
			_, err := s.Invite(ctx, driveItemId, libregraph.DriveItemInvite{})
			return err
		}, map[string]string{"itemID": "1$2!3"}, HaveKeyWithValue("error", "no gateway available at 10.0.0.42"), 0),
		Entry("SpaceRootInvite", "SpaceRootInvite", func(ctx context.Context, s svc.DriveItemPermissionsService) error {
			_, err := s.SpaceRootInvite(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, libregraph.DriveItemInvite{})
			return err
		}, map[string]string{"driveID": "1$2"}, HaveKeyWithValue("error", "no gateway available at 10.0.0.42"), 0),
		Entry("ListPermissions", "ListPermissions", func(ctx context.Context, s svc.DriveItemPermissionsService) error {
			_, err := s.ListPermissions(ctx, driveItemId, svc.ListPermissionsQueryOptions{})
			return err
		}, map[string]string{"itemID": "1$2!3"}, HaveKeyWithValue("error", "no gateway available at 10.0.0.42"), 0),
		Entry("ListSpaceRootPermissions", "ListSpaceRootPermissions", func(ctx context.Context, s svc.DriveItemPermissionsService) error {
			_, err := s.ListSpaceRootPermissions(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, svc.ListPermissionsQueryOptions{})
			return err
		}, map[string]string{"driveID": "1$2"}, HaveKeyWithValue("error", "no gateway available at 10.0.0.42"), 0),
		Entry("DeletePermission (selected by the probing helpers)", "DeletePermission", func(ctx context.Context, s svc.DriveItemPermissionsService) error {
			return s.DeletePermission(ctx, driveItemId, "permissionid")
		}, map[string]string{"itemID": "1$2!3", "permissionID": "permissionid"}, HaveKeyWithValue("error", "no gateway available at 10.0.0.42"), 2),
		Entry("DeleteSpaceRootPermission", "DeleteSpaceRootPermission", func(ctx context.Context, s svc.DriveItemPermissionsService) error {
			return s.DeleteSpaceRootPermission(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, "permissionid")
		}, map[string]string{"driveID": "1$2", "permissionID": "permissionid"}, HaveKeyWithValue("error", "no gateway available at 10.0.0.42"), 0),
		Entry("UpdatePermission (selected by the probing helpers)", "UpdatePermission", func(ctx context.Context, s svc.DriveItemPermissionsService) error {
			_, err := s.UpdatePermission(ctx, driveItemId, "permissionid", libregraph.Permission{})
			return err
		}, map[string]string{"itemID": "1$2!3", "permissionID": "permissionid"}, HaveKeyWithValue("error", "no gateway available at 10.0.0.42"), 1),
		Entry("UpdateSpaceRootPermission", "UpdateSpaceRootPermission", func(ctx context.Context, s svc.DriveItemPermissionsService) error {
			_, err := s.UpdateSpaceRootPermission(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, "permissionid", libregraph.Permission{})
			return err
		}, map[string]string{"driveID": "1$2", "permissionID": "permissionid"}, HaveKeyWithValue("error", "no gateway available at 10.0.0.42"), 0),
	)

	It("does not mark the span of an expected client error as error, but records its error code", func() {
		sr, tp := newSpanRecorder()
		service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
		Expect(err).ToNot(HaveOccurred())
		gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(
			&provider.ListStorageSpacesResponse{Status: status.NewNotFound(ctx, "space Project Alpha not found")}, nil)

		_, err = service.ListSpaceRootPermissions(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, svc.ListPermissionsQueryOptions{})
		Expect(errorcode.IsErrorCode(err, errorcode.ItemNotFound)).To(BeTrue())

		Expect(sr.Ended()).To(HaveLen(1))
		span := sr.Ended()[0]
		Expect(span.Status().Code).To(Equal(codes.Unset))
		Expect(span.Status().Description).To(BeEmpty())
		Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "itemNotFound")))
		Expect(span.Events()).To(BeEmpty())
	})

	// The stat failure tables use their own invite, the Invite specs share theirs between specs.
	inviteAfterStat := func(s svc.DriveItemPermissionsService) error {
		_, err := s.Invite(ctx, driveItemId, libregraph.DriveItemInvite{
			Roles:      []string{unifiedrole.UnifiedRoleViewerID},
			Recipients: []libregraph.DriveRecipient{{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")}},
		})
		return err
	}
	listAfterStat := func(s svc.DriveItemPermissionsService) error {
		_, err := s.ListPermissions(ctx, driveItemId, svc.ListPermissionsQueryOptions{})
		return err
	}
	DescribeTable("logs stat failures by cause with the item id, without the stat payload",
		func(call func(svc.DriveItemPermissionsService) error, newStatus func(context.Context, string) *cs3rpc.Status, code errorcode.ErrorCode, level string) {
			service, logBuf := newLoggedService()
			// decomposedfs reports a missing node as "<parent id>/<name>"
			statMessage := "stat: 7c3e9a1b-parent/private-financial-report-2026.xlsx"
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
				Status: newStatus(ctx, statMessage),
				Info: &provider.ResourceInfo{
					Id:            driveItemId,
					Path:          "/private/project-plan.odt",
					PermissionSet: roleconversions.NewViewerRole().CS3ResourcePermissions(),
				},
			}, nil)

			Expect(call(service)).To(MatchError(errorcode.New(code, statMessage).WithOrigin(errorcode.ErrorOriginCS3)))
			// nothing is shared or listed after the failed stat
			gatewayClient.AssertCalled(GinkgoT(), "Stat", mock.Anything, mock.Anything)
			for _, c := range gatewayClient.Calls {
				Expect(c.Method).To(Equal("Stat"))
			}

			entries := logEntries(logBuf)
			var itemEntries []map[string]any
			for _, entry := range entries {
				if entry["itemID"] == storagespace.FormatResourceID(driveItemId) {
					itemEntries = append(itemEntries, entry)
				}
			}
			Expect(itemEntries).To(ConsistOf(HaveKeyWithValue("level", level)))
			// the stat failure is the only entry at warn or error level
			for _, entry := range entries {
				if entry["level"] == "warn" || entry["level"] == "error" {
					Expect(entry).To(Equal(itemEntries[0]))
				}
			}
			// the failure is logged with its error text, whatever its level
			Expect(itemEntries[0]).To(HaveKeyWithValue("error", errorcode.New(code, statMessage).WithOrigin(errorcode.ErrorOriginCS3).Error()))
			for _, payloadKey := range []string{"stat.res", "res", "response", "request", "info", "Body", "body"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
			// the stat payload never reaches the log
			raw := rawLog(entries)
			for _, sentinel := range []string{"/private/project-plan.odt", "permission_set"} {
				Expect(raw).ToNot(ContainSubstring(sentinel))
			}
		},
		Entry("Invite: missing item as debug", inviteAfterStat, status.NewNotFound, errorcode.ItemNotFound, "debug"),
		Entry("Invite: storage failure as error", inviteAfterStat, status.NewInternal, errorcode.GeneralException, "error"),
		Entry("ListPermissions: missing item as debug", listAfterStat, status.NewNotFound, errorcode.ItemNotFound, "debug"),
		Entry("ListPermissions: storage failure as error", listAfterStat, status.NewInternal, errorcode.GeneralException, "error"),
	)

	DescribeTable("logs unsupported space types as debug with the resource ids",
		func(spaceType string, call func(svc.DriveItemPermissionsService) error, ids map[string]string) {
			service, logBuf := newLoggedService()
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status: status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{{
					Id:        &provider.StorageSpaceId{OpaqueId: "2"},
					SpaceType: spaceType,
					Root:      &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"},
				}},
			}, nil)

			Expect(call(service)).To(MatchError(errorcode.New(errorcode.InvalidRequest, "unsupported space type")))

			// the request was rejected after resolving the space, before delegating to the item method
			for _, c := range gatewayClient.Calls {
				Expect(c.Method).To(Equal("ListStorageSpaces"))
			}

			expected := []gomegatypes.GomegaMatcher{
				HaveKeyWithValue("level", "debug"),
				HaveKeyWithValue("message", "unsupported space type"),
				HaveKeyWithValue("spaceType", spaceType),
			}
			for _, k := range []string{"driveID", "permissionID", "itemID"} {
				if v, ok := ids[k]; ok {
					expected = append(expected, HaveKeyWithValue(k, v))
				} else {
					expected = append(expected, Not(HaveKey(k)))
				}
			}
			entries := logEntries(logBuf)
			Expect(entries).To(ContainElement(SatisfyAll(expected...)))
			Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
		},
		Entry("SpaceRootInvite", "mountpoint", func(s svc.DriveItemPermissionsService) error {
			_, err := s.SpaceRootInvite(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, libregraph.DriveItemInvite{})
			return err
		}, map[string]string{"driveID": "1$2"}),
		Entry("ListSpaceRootPermissions", "mountpoint", func(s svc.DriveItemPermissionsService) error {
			_, err := s.ListSpaceRootPermissions(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, svc.ListPermissionsQueryOptions{})
			return err
		}, map[string]string{"driveID": "1$2"}),
		Entry("DeleteSpaceRootPermission", "personal", func(s svc.DriveItemPermissionsService) error {
			return s.DeleteSpaceRootPermission(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, "permissionid")
		}, map[string]string{"driveID": "1$2", "permissionID": "permissionid"}),
		Entry("UpdateSpaceRootPermission", "personal", func(s svc.DriveItemPermissionsService) error {
			_, err := s.UpdateSpaceRootPermission(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, "permissionid", libregraph.Permission{})
			return err
		}, map[string]string{"driveID": "1$2", "permissionID": "permissionid"}),
	)

	var (
		spaceRootInvite = func(s svc.DriveItemPermissionsService) error {
			_, err := s.SpaceRootInvite(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, libregraph.DriveItemInvite{})
			return err
		}
		listSpaceRootPermissions = func(s svc.DriveItemPermissionsService) error {
			_, err := s.ListSpaceRootPermissions(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, svc.ListPermissionsQueryOptions{})
			return err
		}
		deleteSpaceRootPermission = func(s svc.DriveItemPermissionsService) error {
			return s.DeleteSpaceRootPermission(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, "permissionid")
		}
		updateSpaceRootPermission = func(s svc.DriveItemPermissionsService) error {
			_, err := s.UpdateSpaceRootPermission(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, "permissionid", libregraph.Permission{})
			return err
		}
		missingSpaceResponse = &provider.ListStorageSpacesResponse{Status: status.NewOK(context.Background())}
		missingSpaceErr      = errorcode.New(errorcode.ItemNotFound, "getting space").WithOrigin(errorcode.ErrorOriginCS3)
		// storage providers may report space names or paths in their status messages
		internalSpaceResponse = &provider.ListStorageSpacesResponse{Status: status.NewInternal(context.Background(), "storage provider unavailable for space Project Alpha")}
		internalSpaceErr      = errorcode.New(errorcode.GeneralException, "storage provider unavailable for space Project Alpha").WithOrigin(errorcode.ErrorOriginCS3)
		transportSpaceErr     = errorcode.New(errorcode.GeneralException, "connection refused").WithOrigin(errorcode.ErrorOriginCS3)
	)
	DescribeTable("logs space lookup failures by cause with the resource ids",
		func(call func(svc.DriveItemPermissionsService) error, ids map[string]string, res *provider.ListStorageSpacesResponse, expectedErr error, level string) {
			service, logBuf := newLoggedService()
			// a nil response stands for a failed request
			var transportErr error
			if res == nil {
				transportErr = errors.New("connection refused")
			}
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(res, transportErr)

			Expect(call(service)).To(MatchError(expectedErr))

			// the request failed while resolving the space, before delegating to the item method
			for _, c := range gatewayClient.Calls {
				Expect(c.Method).To(Equal("ListStorageSpaces"))
			}

			graphErr, _ := errorcode.ToError(expectedErr)
			// the failure is logged with its error code and error text, whatever its level
			expected := []gomegatypes.GomegaMatcher{
				HaveKeyWithValue("level", level),
				HaveKeyWithValue("errorCode", graphErr.GetCode().String()),
				HaveKeyWithValue("error", expectedErr.Error()),
			}
			for _, k := range []string{"driveID", "permissionID", "itemID"} {
				if v, ok := ids[k]; ok {
					expected = append(expected, HaveKeyWithValue(k, v))
				} else {
					expected = append(expected, Not(HaveKey(k)))
				}
			}
			entries := logEntries(logBuf)
			Expect(entries).To(ContainElement(SatisfyAll(expected...)))
			for _, payloadKey := range []string{"space", "spaces", "res", "response", "stat.res"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
			if level == "debug" {
				Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
			} else {
				errorEntries := entriesAtLevel(entries, "error")
				Expect(errorEntries).To(HaveLen(1))
			}
		},
		Entry("SpaceRootInvite: missing space", spaceRootInvite,
			map[string]string{"driveID": "1$2"}, missingSpaceResponse, missingSpaceErr, "debug"),
		Entry("SpaceRootInvite: internal gateway error", spaceRootInvite,
			map[string]string{"driveID": "1$2"}, internalSpaceResponse, internalSpaceErr, "error"),
		Entry("SpaceRootInvite: transport error", spaceRootInvite,
			map[string]string{"driveID": "1$2"}, nil, transportSpaceErr, "error"),
		Entry("ListSpaceRootPermissions: missing space", listSpaceRootPermissions,
			map[string]string{"driveID": "1$2"}, missingSpaceResponse, missingSpaceErr, "debug"),
		Entry("ListSpaceRootPermissions: internal gateway error", listSpaceRootPermissions,
			map[string]string{"driveID": "1$2"}, internalSpaceResponse, internalSpaceErr, "error"),
		Entry("DeleteSpaceRootPermission: missing space", deleteSpaceRootPermission,
			map[string]string{"driveID": "1$2", "permissionID": "permissionid"}, missingSpaceResponse, missingSpaceErr, "debug"),
		Entry("DeleteSpaceRootPermission: internal gateway error", deleteSpaceRootPermission,
			map[string]string{"driveID": "1$2", "permissionID": "permissionid"}, internalSpaceResponse, internalSpaceErr, "error"),
		Entry("UpdateSpaceRootPermission: missing space", updateSpaceRootPermission,
			map[string]string{"driveID": "1$2", "permissionID": "permissionid"}, missingSpaceResponse, missingSpaceErr, "debug"),
		Entry("UpdateSpaceRootPermission: internal gateway error", updateSpaceRootPermission,
			map[string]string{"driveID": "1$2", "permissionID": "permissionid"}, internalSpaceResponse, internalSpaceErr, "error"),
	)

	Describe("Invite", func() {
		var (
			createShareResponse *collaboration.CreateShareResponse
			driveItemInvite     libregraph.DriveItemInvite
			getGroupResponse    *grouppb.GetGroupResponse
		)

		BeforeEach(func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			statResponse.Info = &provider.ResourceInfo{
				Id:   driveItemId,
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
			}

			getGroupResponse = &grouppb.GetGroupResponse{
				Status: status.NewOK(ctx),
				Group: &grouppb.Group{
					Id:          &grouppb.GroupId{OpaqueId: "2"},
					GroupName:   "Florida Institute of Technology",
					DisplayName: "Florida Institute of Technology",
				},
			}

			createShareResponse = &collaboration.CreateShareResponse{
				Status: status.NewOK(ctx),
			}
		})

		It("creates user shares as expected (happy path)", func() {
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
			createShareResponse.Share = &collaboration.Share{
				Id:         &collaboration.ShareId{OpaqueId: "123"},
				Expiration: utils.TimeToTS(*driveItemInvite.ExpirationDateTime),
			}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())
			Expect(permission.GetId()).To(Equal("123"))
			Expect(permission.GetExpirationDateTime().Equal(*driveItemInvite.ExpirationDateTime)).To(BeTrue())
			Expect(permission.GrantedToV2.User.GetDisplayName()).To(Equal(getUserResponse.User.DisplayName))
			Expect(permission.GrantedToV2.User.GetId()).To(Equal("1"))
		})

		It("creates group shares as expected (happy path)", func() {
			gatewayClient.On("GetGroup", mock.Anything, mock.Anything).Return(getGroupResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("2"), LibreGraphRecipientType: libregraph.PtrString("group")},
			}
			driveItemInvite.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
			createShareResponse.Share = &collaboration.Share{
				Id:         &collaboration.ShareId{OpaqueId: "123"},
				Expiration: utils.TimeToTS(*driveItemInvite.ExpirationDateTime),
			}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())
			Expect(permission.GetId()).To(Equal("123"))
			Expect(permission.GetExpirationDateTime().Equal(*driveItemInvite.ExpirationDateTime)).To(BeTrue())
			Expect(permission.GrantedToV2.Group.GetDisplayName()).To(Equal(getGroupResponse.Group.DisplayName))
			Expect(permission.GrantedToV2.Group.GetId()).To(Equal("2"))
		})

		It("creates guest share using an email address", func() {
			cfg.Commons = &shared.Commons{EnableGuestLinks: true}
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{Email: libregraph.PtrString("guest@example.com")},
			}
			createShareResponse.Share = &collaboration.Share{
				Id:      &collaboration.ShareId{OpaqueId: "guest123"},
				Grantee: guestGrantee("guest@example.com"),
			}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())
			Expect(permission.GetId()).To(Equal("guest123"))
			Expect(permission.GrantedToV2.User.GetDisplayName()).To(Equal("guest@example.com"))
			Expect(permission.GrantedToV2.User.GetId()).To(Equal("guest@example.com"))
			Expect(permission.GrantedToV2.User.GetLibreGraphUserType()).To(Equal("Guest"))
		})

		It("returns the guest id as stored by the share provider", func() {
			cfg.Commons = &shared.Commons{EnableGuestLinks: true}
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{Email: libregraph.PtrString("Guest@\u0130nfocorp.com")},
			}
			createShareResponse.Share = &collaboration.Share{
				Id:      &collaboration.ShareId{OpaqueId: "guest123"},
				Grantee: guestGrantee("guest@xn--infocorp-o0e.com"),
			}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())
			Expect(permission.GrantedToV2.User.GetId()).To(Equal("guest@xn--infocorp-o0e.com"))
		})

		It("logs a failed grantee lookup of a mail share by cause with the item id", func() {
			cfg.Commons = &shared.Commons{EnableGuestLinks: true}
			service, logBuf := newLoggedService()

			// the share provider returns a regular user as grantee, whose lookup fails in the identity backend
			createShareResponse.Share = &collaboration.Share{
				Id: &collaboration.ShareId{OpaqueId: "guest123"},
				Grantee: &provider.Grantee{
					Type: provider.GranteeType_GRANTEE_TYPE_USER,
					Id: &provider.Grantee_UserId{UserId: &userpb.UserId{
						OpaqueId: "grantee-id", Type: userpb.UserType_USER_TYPE_PRIMARY,
					}},
				},
			}
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(nil, errors.New("connection to guest@example.com refused"))
			invite := libregraph.DriveItemInvite{
				Roles:      []string{unifiedrole.UnifiedRoleViewerID},
				Recipients: []libregraph.DriveRecipient{{Email: libregraph.PtrString("guest@example.com")}},
			}

			_, err := service.Invite(ctx, driveItemId, invite)
			Expect(errorcode.IsErrorCode(err, errorcode.GeneralException)).To(BeTrue())

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "failed to convert guest user id to identity"),
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(driveItemId)),
				HaveKeyWithValue("recipientType", "mail"),
				HaveKeyWithValue("errorCode", "generalException"),
				HaveKeyWithValue("error", err.Error()),
			))
		})

		It("rejects guest shares when guest invites are disabled by default", func() {
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{Email: libregraph.PtrString("guest@example.com")},
			}

			_, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not enabled"))
		})

		It("succeeds with file roles (happy path)", func() {
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())

			Expect(permission.GetRoles()).To(HaveLen(1))
			Expect(permission.GetRoles()[0]).To(Equal(unifiedrole.UnifiedRoleViewerID))
		})

		It("succeeds with folder roles (happy path)", func() {
			statResponse.Info.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleEditorID}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())

			Expect(permission.GetRoles()).To(HaveLen(1))
			Expect(permission.GetRoles()[0]).To(Equal(unifiedrole.UnifiedRoleEditorID))
		})

		DescribeTable("logs invite validation failures as debug with the resource ids",
			func(setup func() *provider.ResourceId, expectedErr error, ids map[string]string) {
				itemID := setup()
				service, logBuf := newLoggedService()

				_, err := service.Invite(ctx, itemID, driveItemInvite)
				Expect(err).To(MatchError(expectedErr))
				// validation failed before anything was shared
				gatewayClient.AssertNotCalled(GinkgoT(), "CreateShare", mock.Anything, mock.Anything)
				gatewayClient.AssertNotCalled(GinkgoT(), "CreateOCMShare", mock.Anything, mock.Anything)

				expected := []gomegatypes.GomegaMatcher{
					HaveKeyWithValue("level", "debug"),
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(itemID)),
				}
				for k, v := range ids {
					expected = append(expected, HaveKeyWithValue(k, v))
				}
				entries := logEntries(logBuf)
				Expect(entries).To(ContainElement(SatisfyAll(expected...)))
				Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
				for _, payloadKey := range []string{"invite", "recipient", "recipients", "email", "mail", "Body", "body"} {
					Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
				}
			},
			Entry("unsupported resource type",
				func() *provider.ResourceId {
					statResponse.Info.Type = provider.ResourceType_RESOURCE_TYPE_SYMLINK
					driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID}
					driveItemInvite.Recipients = []libregraph.DriveRecipient{
						{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
					}
					return driveItemId
				},
				errorcode.New(errorcode.InvalidRequest, "unsupported resource type"),
				map[string]string{}),
			Entry("unknown or disabled role",
				func() *provider.ResourceId {
					driveItemInvite.Roles = []string{"unknown-role-id"}
					driveItemInvite.Recipients = []libregraph.DriveRecipient{
						{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
					}
					return driveItemId
				},
				unifiedrole.ErrUnknownRole,
				map[string]string{"roleID": "unknown-role-id"}),
			Entry("enabled role without a definition",
				func() *provider.ResourceId {
					// the failing role is not the first requested one
					cfg.UnifiedRoles.AvailableRoles = append(cfg.UnifiedRoles.AvailableRoles, "undefined-role-id")
					driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID, "undefined-role-id"}
					driveItemInvite.Recipients = []libregraph.DriveRecipient{
						{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
					}
					return driveItemId
				},
				unifiedrole.ErrUnknownRole,
				map[string]string{"roleID": "undefined-role-id"}),
			Entry("role not applicable to the resource",
				func() *provider.ResourceId {
					driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleManagerID}
					driveItemInvite.Recipients = []libregraph.DriveRecipient{
						{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
					}
					return driveItemId
				},
				errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource"),
				map[string]string{"roleID": unifiedrole.UnifiedRoleManagerID}),
			Entry("mail sharing disabled",
				func() *provider.ResourceId {
					driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID}
					driveItemInvite.Recipients = []libregraph.DriveRecipient{
						{Email: libregraph.PtrString("guest@example.com")},
					}
					return driveItemId
				},
				errorcode.New(errorcode.NotSupported, "sharing with mail recipients is not enabled"),
				map[string]string{}),
			Entry("federated user invited as a space member",
				func() *provider.ResourceId {
					cfg.IncludeOCMSharees = true
					spaceRootID := &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}
					statResponse.Info = &provider.ResourceInfo{
						Id:    spaceRootID,
						Type:  provider.ResourceType_RESOURCE_TYPE_CONTAINER,
						Space: &provider.StorageSpace{Root: spaceRootID},
					}
					gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(&userpb.GetUserResponse{
						Status: status.NewNotFound(ctx, "user not found"),
					}, nil)
					gatewayClient.On("GetAcceptedUser", mock.Anything, mock.Anything).Return(&invitepb.GetAcceptedUserResponse{
						Status: status.NewOK(ctx),
						RemoteUser: &userpb.User{
							Id: &userpb.UserId{OpaqueId: "remote", Idp: "ocm.example.org", Type: userpb.UserType_USER_TYPE_FEDERATED},
						},
					}, nil)
					driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleSpaceViewerID}
					driveItemInvite.Recipients = []libregraph.DriveRecipient{
						{ObjectId: libregraph.PtrString("remote@ocm.example.org"), LibreGraphRecipientType: libregraph.PtrString("user")},
					}
					return spaceRootID
				},
				errorcode.New(errorcode.InvalidRequest, "federated user can not become a space member"),
				map[string]string{}),
		)

		DescribeTable("logs invite recipient lookup failures by cause with the item id",
			func(recipientType string, setup func(), expectedErr error, level string) {
				setup()
				service, logBuf := newLoggedService()
				driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID}
				driveItemInvite.Recipients = []libregraph.DriveRecipient{
					{ObjectId: libregraph.PtrString("recipient-id-42"), LibreGraphRecipientType: libregraph.PtrString(recipientType)},
				}

				_, err := service.Invite(ctx, driveItemId, driveItemInvite)
				Expect(err).To(MatchError(expectedErr))
				// nothing is shared after the failed lookup
				gatewayClient.AssertNotCalled(GinkgoT(), "CreateShare", mock.Anything, mock.Anything)
				gatewayClient.AssertNotCalled(GinkgoT(), "CreateOCMShare", mock.Anything, mock.Anything)

				entries := logEntries(logBuf)
				// the failure is logged with the error text of the lookup, whatever its level
				Expect(entries).To(ContainElement(SatisfyAll(
					HaveKeyWithValue("level", level),
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(driveItemId)),
					HaveKeyWithValue("recipientType", recipientType),
					HaveKey("error"),
				)))
				if level == "debug" {
					Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
				} else {
					Expect(entriesAtLevel(entries, "error")).To(HaveLen(1))
				}
				for _, payloadKey := range []string{"objectID", "groupId", "userId", "recipient", "recipients", "invite", "Body", "body"} {
					Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
				}
			},
			Entry("A: group not found", "group",
				func() {
					gatewayClient.On("GetGroup", mock.Anything, mock.Anything).Return(&grouppb.GetGroupResponse{
						Status: status.NewNotFound(ctx, "group recipient-id-42 not found"),
					}, nil)
				},
				errorcode.New(errorcode.InvalidRequest, "itemNotFound: not found"), "debug"),
			Entry("B: group lookup backend failure", "group",
				func() {
					gatewayClient.On("GetGroup", mock.Anything, mock.Anything).Return(&grouppb.GetGroupResponse{
						Status: status.NewInternal(ctx, "lookup of recipient-id-42 failed"),
					}, nil)
				},
				errorcode.New(errorcode.InvalidRequest, "generalException: lookup of recipient-id-42 failed"), "error"),
			Entry("C: user not found", "user",
				func() {
					gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(&userpb.GetUserResponse{
						Status: status.NewNotFound(ctx, "user recipient-id-42 not found"),
					}, nil)
				},
				errorcode.New(errorcode.InvalidRequest, "itemNotFound: not found"), "debug"),
			Entry("D: user lookup backend failure", "user",
				func() {
					gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(&userpb.GetUserResponse{
						Status: status.NewInternal(ctx, "lookup of recipient-id-42 failed"),
					}, nil)
				},
				errorcode.New(errorcode.InvalidRequest, "generalException: unexpected status code while getting user: CODE_INTERNAL"), "error"),
		)

		DescribeTable("logs invite share creation failures by cause with the item id",
			func(recipientType string, setup func(), expectedErr error, level string) {
				setup()
				service, logBuf := newLoggedService()
				driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID}

				_, err := service.Invite(ctx, driveItemId, driveItemInvite)
				Expect(err).To(MatchError(expectedErr))
				// the share creation was attempted
				gatewayClient.AssertCalled(GinkgoT(), "CreateShare", mock.Anything, mock.Anything)

				entries := logEntries(logBuf)
				var creationEntries []map[string]any
				for _, entry := range entries {
					if entry["level"] == level && entry["itemID"] == storagespace.FormatResourceID(driveItemId) && entry["recipientType"] == recipientType {
						creationEntries = append(creationEntries, entry)
					}
				}
				Expect(creationEntries).To(HaveLen(1))
				// the failure is logged with its error text, whatever its level
				Expect(creationEntries[0]).To(HaveKeyWithValue("error", expectedErr.Error()))
				if level == "error" {
					Expect(entriesAtLevel(entries, "error")).To(HaveLen(1))
				} else {
					Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
				}
				for _, payloadKey := range []string{"objectID", "groupId", "userId", "email", "mail", "recipient", "recipients", "share", "invite", "Body", "body"} {
					Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
				}
			},
			Entry("A: mail recipient, internal error", "mail",
				func() {
					cfg.Commons = &shared.Commons{EnableGuestLinks: true}
					driveItemInvite.Recipients = []libregraph.DriveRecipient{{Email: libregraph.PtrString("guest@example.com")}}
					gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(&collaboration.CreateShareResponse{
						Status: status.NewInternal(ctx, "could not share with guest@example.com"),
					}, nil)
				},
				errorcode.New(errorcode.GeneralException, "could not share with guest@example.com").WithOrigin(errorcode.ErrorOriginCS3), "error"),
			Entry("B: group recipient, internal error", "group",
				func() {
					driveItemInvite.Recipients = []libregraph.DriveRecipient{
						{ObjectId: libregraph.PtrString("recipient-id-42"), LibreGraphRecipientType: libregraph.PtrString("group")},
					}
					gatewayClient.On("GetGroup", mock.Anything, mock.Anything).Return(getGroupResponse, nil)
					gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(&collaboration.CreateShareResponse{
						Status: status.NewInternal(ctx, "could not share with recipient-id-42"),
					}, nil)
				},
				errorcode.New(errorcode.GeneralException, "could not share with recipient-id-42").WithOrigin(errorcode.ErrorOriginCS3), "error"),
			Entry("C: user recipient, internal error", "user",
				func() {
					driveItemInvite.Recipients = []libregraph.DriveRecipient{
						{ObjectId: libregraph.PtrString("recipient-id-42"), LibreGraphRecipientType: libregraph.PtrString("user")},
					}
					gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
					gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(&collaboration.CreateShareResponse{
						Status: status.NewInternal(ctx, "could not share with recipient-id-42"),
					}, nil)
				},
				errorcode.New(errorcode.GeneralException, "could not share with recipient-id-42").WithOrigin(errorcode.ErrorOriginCS3), "error"),
			Entry("D: user recipient, share already exists", "user",
				func() {
					driveItemInvite.Recipients = []libregraph.DriveRecipient{
						{ObjectId: libregraph.PtrString("recipient-id-42"), LibreGraphRecipientType: libregraph.PtrString("user")},
					}
					gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
					gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(&collaboration.CreateShareResponse{
						Status: status.NewAlreadyExists(ctx, nil, "share with recipient-id-42 already exists"),
					}, nil)
				},
				errorcode.New(errorcode.NameAlreadyExists, "share with recipient-id-42 already exists").WithOrigin(errorcode.ErrorOriginCS3), "debug"),
		)

		DescribeTable("logs invite ocm failures by cause with the item id",
			func(setup func(), attempted string, expectedErr error, level string) {
				// a federated recipient, found via the accepted user fallback, on a regular item
				cfg.IncludeOCMSharees = true
				gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(&userpb.GetUserResponse{
					Status: status.NewNotFound(ctx, "user not found"),
				}, nil)
				gatewayClient.On("GetAcceptedUser", mock.Anything, mock.Anything).Return(&invitepb.GetAcceptedUserResponse{
					Status: status.NewOK(ctx),
					RemoteUser: &userpb.User{
						Id: &userpb.UserId{OpaqueId: "remote", Idp: "ocm.example.org", Type: userpb.UserType_USER_TYPE_FEDERATED},
					},
				}, nil)
				setup()
				service, logBuf := newLoggedService()
				driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID}
				driveItemInvite.Recipients = []libregraph.DriveRecipient{
					{ObjectId: libregraph.PtrString("remote@ocm.example.org"), LibreGraphRecipientType: libregraph.PtrString("user")},
				}

				_, err := service.Invite(ctx, driveItemId, driveItemInvite)
				Expect(err).To(MatchError(expectedErr))
				// the failing ocm operation was attempted, nothing else was shared
				gatewayClient.AssertCalled(GinkgoT(), attempted, mock.Anything, mock.Anything)
				gatewayClient.AssertNotCalled(GinkgoT(), "CreateShare", mock.Anything, mock.Anything)

				entries := logEntries(logBuf)
				var ocmEntries []map[string]any
				for _, entry := range entries {
					if entry["level"] == level && entry["itemID"] == storagespace.FormatResourceID(driveItemId) && entry["recipientType"] == "ocm" {
						ocmEntries = append(ocmEntries, entry)
					}
				}
				Expect(ocmEntries).To(HaveLen(1))
				// the failure is logged with its error text, whatever its level
				Expect(ocmEntries[0]).To(HaveKeyWithValue("error", expectedErr.Error()))
				if level == "error" {
					Expect(entriesAtLevel(entries, "error")).To(HaveLen(1))
				} else {
					Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
				}
				for _, payloadKey := range []string{"objectID", "userId", "domain", "provider", "share", "recipient", "recipients", "invite", "Body", "body"} {
					Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
				}
			},
			Entry("A: provider info lookup, internal error",
				func() {
					gatewayClient.On("GetInfoByDomain", mock.Anything, mock.Anything).Return(&ocmprovider.GetInfoByDomainResponse{
						Status: status.NewInternal(ctx, "no provider info for ocm.example.org"),
					}, nil)
				},
				"GetInfoByDomain",
				errorcode.New(errorcode.GeneralException, "no provider info for ocm.example.org").WithOrigin(errorcode.ErrorOriginCS3), "error"),
			Entry("B: ocm share creation, internal error",
				func() {
					gatewayClient.On("GetInfoByDomain", mock.Anything, mock.Anything).Return(&ocmprovider.GetInfoByDomainResponse{
						Status:       status.NewOK(ctx),
						ProviderInfo: &ocmprovider.ProviderInfo{Domain: "ocm.example.org"},
					}, nil)
					gatewayClient.On("CreateOCMShare", mock.Anything, mock.Anything).Return(&ocm.CreateOCMShareResponse{
						Status: status.NewInternal(ctx, "could not share with remote@ocm.example.org"),
					}, nil)
				},
				"CreateOCMShare",
				errorcode.New(errorcode.GeneralException, "could not share with remote@ocm.example.org").WithOrigin(errorcode.ErrorOriginCS3), "error"),
			Entry("C: ocm share creation, share already exists",
				func() {
					gatewayClient.On("GetInfoByDomain", mock.Anything, mock.Anything).Return(&ocmprovider.GetInfoByDomainResponse{
						Status:       status.NewOK(ctx),
						ProviderInfo: &ocmprovider.ProviderInfo{Domain: "ocm.example.org"},
					}, nil)
					gatewayClient.On("CreateOCMShare", mock.Anything, mock.Anything).Return(&ocm.CreateOCMShareResponse{
						Status: status.NewAlreadyExists(ctx, nil, "share with remote@ocm.example.org already exists"),
					}, nil)
				},
				"CreateOCMShare",
				errorcode.New(errorcode.NameAlreadyExists, "share with remote@ocm.example.org already exists").WithOrigin(errorcode.ErrorOriginCS3), "debug"),
		)

		DescribeTable("marks the Invite span by the cause of a failed identity lookup",
			func(recipientType string, setup func(), wantStatus codes.Code, wantDescription string) {
				sr, tp := newSpanRecorder()
				service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
				Expect(err).ToNot(HaveOccurred())
				setup()
				invite := libregraph.DriveItemInvite{
					Roles: []string{unifiedrole.UnifiedRoleViewerID},
					Recipients: []libregraph.DriveRecipient{
						{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString(recipientType)},
					},
				}

				_, err = service.Invite(ctx, driveItemId, invite)
				// the identity lookup failure is still returned as invalid request
				Expect(errorcode.IsErrorCode(err, errorcode.InvalidRequest)).To(BeTrue())

				var span sdktrace.ReadOnlySpan
				for _, s := range sr.Ended() {
					if s.Name() == "Invite" {
						span = s
					}
				}
				Expect(span).ToNot(BeNil())
				Expect(span.Status().Code).To(Equal(wantStatus))
				Expect(span.Status().Description).To(Equal(wantDescription))
				Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "invalidRequest")))
				Expect(span.Events()).To(BeEmpty())
				// backend identity text never reaches the span
				raw, err := json.Marshal([]any{span.Status(), span.Attributes()})
				Expect(err).ToNot(HaveOccurred())
				Expect(string(raw)).ToNot(ContainSubstring("alice@example.com"))
			},
			Entry("recipient not found", "user", func() {
				gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(&userpb.GetUserResponse{
					Status: status.NewNotFound(ctx, "alice@example.com not found"),
				}, nil)
			}, codes.Unset, ""),
			Entry("recipient lookup backend failure", "user", func() {
				gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(nil, errors.New("connection to alice@example.com refused"))
			}, codes.Error, "generalException"),
			Entry("group recipient lookup backend failure", "group", func() {
				gatewayClient.On("GetGroup", mock.Anything, mock.Anything).Return(&grouppb.GetGroupResponse{
					Status: status.NewInternal(ctx, "lookup of alice@example.com failed"),
				}, nil)
			}, codes.Error, "generalException"),
			Entry("inviter lookup backend failure", "user", func() {
				gatewayClient.On("GetUser", mock.Anything, mock.MatchedBy(func(req *userpb.GetUserRequest) bool {
					return req.GetUserId().GetOpaqueId() == "1"
				})).Return(getUserResponse, nil)
				createShareResponse.Share = &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: "123"}}
				gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
				gatewayClient.On("GetUser", mock.Anything, mock.MatchedBy(func(req *userpb.GetUserRequest) bool {
					return req.GetUserId().GetOpaqueId() == currentUser.GetId().GetOpaqueId()
				})).Return(nil, errors.New("connection to alice@example.com refused"))
			}, codes.Error, "generalException"),
		)

		It("logs a failed inviter identity lookup as error with the item id and the error text", func() {
			service, logBuf := newLoggedService()

			// the recipient is found and the share is created ...
			gatewayClient.On("GetUser", mock.Anything, mock.MatchedBy(func(req *userpb.GetUserRequest) bool {
				return req.GetUserId().GetOpaqueId() == "1"
			})).Return(getUserResponse, nil)
			createShareResponse.Share = &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: "123"}}
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			// ... but the lookup of the inviting user fails afterwards
			gatewayClient.On("GetUser", mock.Anything, mock.MatchedBy(func(req *userpb.GetUserRequest) bool {
				return req.GetUserId().GetOpaqueId() == currentUser.GetId().GetOpaqueId()
			})).Return(nil, errors.New("connection to inviter@example.com refused"))
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID}
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}

			_, err := service.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "generalException: connection to inviter@example.com refused")))
			// the share was created before the inviter lookup, as in the existing control flow
			gatewayClient.AssertCalled(GinkgoT(), "CreateShare", mock.Anything, mock.Anything)
			gatewayClient.AssertCalled(GinkgoT(), "GetUser", mock.Anything, mock.MatchedBy(func(req *userpb.GetUserRequest) bool {
				return req.GetUserId().GetOpaqueId() == currentUser.GetId().GetOpaqueId()
			}))

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(driveItemId)),
				// the cause of the failed lookup, with its error text
				HaveKeyWithValue("errorCode", "generalException"),
				HaveKeyWithValue("error", "generalException: connection to inviter@example.com refused"),
			))
			for _, payloadKey := range []string{"userId", "userID", "objectID", "user", "invite", "share", "Body", "body"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
		})

		It("fails with when trying to set a space role on a file", func() {
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleManagerID}
			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)

			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(permission).To(BeZero())
		})

		It("fails with when trying to set a folder role on a file", func() {
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleEditorID}
			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)

			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(permission).To(BeZero())
		})

		It("fails with when trying to set a file role on a folder", func() {
			statResponse.Info.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleFileEditorID}
			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)

			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(permission).To(BeZero())
		})

		It("with actions (happy path)", func() {
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.Roles = nil
			driveItemInvite.LibreGraphPermissionsActions = []string{unifiedrole.DriveItemContentRead}

			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())

			Expect(permission).NotTo(BeZero())
			Expect(permission.GetRoles()).To(HaveLen(0))
			Expect(permission.GetLibreGraphPermissionsActions()).To(HaveLen(1))
			Expect(permission.GetLibreGraphPermissionsActions()[0]).To(Equal(unifiedrole.DriveItemContentRead))
		})

		It("fails with a missing driveritem", func() {
			statResponse.Status = status.NewNotFound(context.Background(), "not found")
			permission, err := driveItemPermissionsService.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "not found").WithOrigin(errorcode.ErrorOriginCS3)))
			Expect(permission).To(BeZero())
		})

		It("fails with unknown or disable role", func() {
			cfg := defaults.FullDefaultConfig()
			slices.DeleteFunc(cfg.UnifiedRoles.AvailableRoles, func(s string) bool {
				// SecureViewer is enabled in ci, we need to remove it in the unit test
				return s != unifiedrole.UnifiedRoleSecureViewerID
			})
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, nil)
			Expect(err).ToNot(HaveOccurred())

			driveItemInvite.Roles = []string{unifiedrole.UnifiedRoleViewerID, unifiedrole.UnifiedRoleSecureViewerID}
			_, err = service.Invite(ctx, driveItemId, driveItemInvite)
			Expect(err).To(MatchError(unifiedrole.ErrUnknownRole))
		})
	})
	Describe("SpaceRootInvite", func() {
		var (
			createShareResponse *collaboration.CreateShareResponse
			driveItemInvite     libregraph.DriveItemInvite
			driveId             *provider.ResourceId
			getUserResponse     *userpb.GetUserResponse
		)

		BeforeEach(func() {
			driveId = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
			}

			listSpacesResponse = &provider.ListStorageSpacesResponse{
				Status: status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{
					{
						Id: &provider.StorageSpaceId{
							OpaqueId: "2",
						},
					},
				},
			}

			getUserResponse = &userpb.GetUserResponse{
				Status: status.NewOK(ctx),
				User: &userpb.User{
					Id:          &userpb.UserId{OpaqueId: "1"},
					DisplayName: "Cem Kaner",
				},
			}

			createShareResponse = &collaboration.CreateShareResponse{
				Status: status.NewOK(ctx),
			}
			statResponse.Info = &provider.ResourceInfo{}
		})

		It("adds a user to a space as expected (happy path)", func() {
			listSpacesResponse.StorageSpaces[0].SpaceType = "project"
			listSpacesResponse.StorageSpaces[0].Root = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "3",
			}
			statResponse.Info.Id = listSpacesResponse.StorageSpaces[0].Root
			statResponse.Info.Space = &provider.StorageSpace{
				Root: listSpacesResponse.StorageSpaces[0].Root,
			}

			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
			createShareResponse.Share = &collaboration.Share{
				Id:         &collaboration.ShareId{OpaqueId: "123"},
				Expiration: utils.TimeToTS(*driveItemInvite.ExpirationDateTime),
			}

			permission, err := driveItemPermissionsService.SpaceRootInvite(ctx, driveId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())
			Expect(permission.GetId()).To(Equal("123"))
			Expect(permission.GetExpirationDateTime().Equal(*driveItemInvite.ExpirationDateTime)).To(BeTrue())
			Expect(permission.GrantedToV2.User.GetDisplayName()).To(Equal(getUserResponse.User.DisplayName))
			Expect(permission.GrantedToV2.User.GetId()).To(Equal("1"))
		})
		It("records a SpaceRootInvite span as the parent of the Invite span", func() {
			sr, tp := newSpanRecorder()
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			listSpacesResponse.StorageSpaces[0].SpaceType = "project"
			listSpacesResponse.StorageSpaces[0].Root = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "3",
			}
			statResponse.Info.Id = listSpacesResponse.StorageSpaces[0].Root
			statResponse.Info.Space = &provider.StorageSpace{
				Root: listSpacesResponse.StorageSpaces[0].Root,
			}
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("CreateShare", mock.Anything, mock.Anything).Return(createShareResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			createShareResponse.Share = &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: "123"}}

			permission, err := service.SpaceRootInvite(ctx, driveId, driveItemInvite)
			Expect(err).ToNot(HaveOccurred())
			// the space was resolved and the call was delegated to Invite, which created the share
			Expect(permission.GetId()).To(Equal("123"))
			gatewayClient.AssertCalled(GinkgoT(), "CreateShare", mock.Anything, mock.Anything)

			spans := map[string]sdktrace.ReadOnlySpan{}
			for _, s := range sr.Ended() {
				spans[s.Name()] = s
			}
			Expect(spans).To(HaveKey("SpaceRootInvite"))
			Expect(spans).To(HaveKey("Invite"))
			parent := spans["SpaceRootInvite"].SpanContext()
			child := spans["Invite"]
			Expect(parent.IsValid()).To(BeTrue())
			Expect(child.SpanContext().IsValid()).To(BeTrue())
			Expect(child.Parent().SpanID()).To(Equal(parent.SpanID()))
			Expect(child.SpanContext().TraceID()).To(Equal(parent.TraceID()))
		})
		It("rejects to add a user to a personal space", func() {
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)
			driveItemInvite.Recipients = []libregraph.DriveRecipient{
				{ObjectId: libregraph.PtrString("1"), LibreGraphRecipientType: libregraph.PtrString("user")},
			}
			driveItemInvite.ExpirationDateTime = libregraph.PtrTime(time.Now().Add(time.Hour))
			createShareResponse.Share = &collaboration.Share{
				Id:         &collaboration.ShareId{OpaqueId: "123"},
				Expiration: utils.TimeToTS(*driveItemInvite.ExpirationDateTime),
			}

			permission, err := driveItemPermissionsService.SpaceRootInvite(ctx, driveId, driveItemInvite)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "unsupported space type")))
			Expect(permission).To(BeZero())
		})
	})
	Describe("ListPermissions", func() {
		var (
			itemID             *provider.ResourceId
			listSharesResponse *collaboration.ListSharesResponse
		)
		BeforeEach(func() {
			itemID = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "3",
			}
			listSharesResponse = &collaboration.ListSharesResponse{
				Status: status.NewOK(ctx),
				Shares: []*collaboration.Share{},
			}
			statResponse.Info = &provider.ResourceInfo{
				Id:            itemID,
				Type:          provider.ResourceType_RESOURCE_TYPE_FILE,
				PermissionSet: roleconversions.NewViewerRole().CS3ResourcePermissions(),
			}
		})
		It("logs a share listing failure once as error with the item id", func() {
			service, logBuf := newLoggedService()

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(&collaboration.ListSharesResponse{
				Status: status.NewInternal(context.Background(), "share manager unavailable"),
			}, nil)

			_, err := service.ListPermissions(ctx, itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, "share manager unavailable")))
			// listing stops at the failed user share lookup
			gatewayClient.AssertNotCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not list user shares"),
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(itemID)),
				HaveKeyWithValue("error", err.Error()),
				HaveKeyWithValue("errorCode", "generalException"),
			))
			for _, payloadKey := range []string{"shares", "share", "res", "response", "stat.res"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
		})
		// spaceRootSharesRequest matches a share listing request for space root (or regular) shares only
		spaceRootSharesRequest := func(spaceRoot bool) any {
			return mock.MatchedBy(func(req *collaboration.ListSharesRequest) bool {
				return slices.ContainsFunc(req.GetFilters(), func(f *collaboration.Filter) bool {
					return f.GetType() == collaboration.Filter_TYPE_SPACE_ROOT && f.GetSpaceRoot() == spaceRoot
				})
			})
		}
		DescribeTable("logs the identity backend error when a share grantee lookup fails",
			func(setup func(groupGrantee *provider.Grantee) *provider.ResourceId, listingMethod string) {
				groupGrantee := &provider.Grantee{
					Type: provider.GranteeType_GRANTEE_TYPE_GROUP,
					Id:   &provider.Grantee_GroupId{GroupId: &grouppb.GroupId{OpaqueId: "groupid"}},
				}
				listedItemID := setup(groupGrantee)
				service, logBuf := newLoggedService()

				const identityBackendMessage = "ldap search failed for cn=private-group-dn-42,ou=groups"
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
				gatewayClient.On("GetGroup", mock.Anything, mock.Anything).Return(&grouppb.GetGroupResponse{
					Status: status.NewInternal(context.Background(), identityBackendMessage),
				}, nil)

				_, err := service.ListPermissions(ctx, listedItemID, svc.ListPermissionsQueryOptions{})
				// the returned error is unchanged: the share conversion error wrapped by the listing
				Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException,
					errorcode.New(errorcode.GeneralException,
						errorcode.New(errorcode.GeneralException, identityBackendMessage).Error()).Error())))
				// the shares were listed and the group grantee was looked up, listing stops before the public shares
				gatewayClient.AssertCalled(GinkgoT(), listingMethod, mock.Anything, mock.Anything)
				gatewayClient.AssertCalled(GinkgoT(), "GetGroup", mock.Anything, mock.MatchedBy(func(req *grouppb.GetGroupRequest) bool {
					return req.GetGroupId().GetOpaqueId() == "groupid"
				}))
				gatewayClient.AssertNotCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)

				entries := logEntries(logBuf)
				errorEntries := entriesAtLevel(entries, "error")
				Expect(errorEntries).To(HaveLen(1))
				Expect(errorEntries[0]).To(SatisfyAll(
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(listedItemID)),
					HaveKeyWithValue("error", err.Error()),
				))
				for _, payloadKey := range []string{"shares", "share", "group", "res", "response", "stat.res"} {
					Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
				}
			},
			Entry("user share listing",
				func(groupGrantee *provider.Grantee) *provider.ResourceId {
					gatewayClient.On("ListShares", mock.Anything, spaceRootSharesRequest(false)).Return(&collaboration.ListSharesResponse{
						Status: status.NewOK(ctx),
						Shares: []*collaboration.Share{{
							Id:          &collaboration.ShareId{OpaqueId: "groupshareid"},
							ResourceId:  itemID,
							Grantee:     groupGrantee,
							Permissions: &collaboration.SharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
						}},
					}, nil)
					return itemID
				},
				"ListShares",
			),
			Entry("space root share listing",
				func(groupGrantee *provider.Grantee) *provider.ResourceId {
					spaceRootID := &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}
					statResponse.Info.Id = spaceRootID
					statResponse.Info.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
					statResponse.Info.Space = &provider.StorageSpace{Root: spaceRootID}
					gatewayClient.On("ListShares", mock.Anything, spaceRootSharesRequest(true)).Return(&collaboration.ListSharesResponse{
						Status: status.NewOK(ctx),
						Shares: []*collaboration.Share{{
							Id:          &collaboration.ShareId{OpaqueId: "groupshareid"},
							ResourceId:  spaceRootID,
							Grantee:     groupGrantee,
							Permissions: &collaboration.SharePermissions{Permissions: roleconversions.NewSpaceViewerRole().CS3ResourcePermissions()},
						}},
					}, nil)
					return spaceRootID
				},
				"ListShares",
			),
			Entry("ocm share listing",
				func(groupGrantee *provider.Grantee) *provider.ResourceId {
					cfg.IncludeOCMSharees = true
					gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
					gatewayClient.On("ListOCMShares", mock.Anything, mock.Anything).Return(&ocm.ListOCMSharesResponse{
						Status: status.NewOK(ctx),
						Shares: []*ocm.Share{{
							Id:         &ocm.ShareId{OpaqueId: "ocmgroupshareid"},
							ResourceId: itemID,
							Grantee:    groupGrantee,
						}},
					}, nil)
					return itemID
				},
				"ListOCMShares",
			),
		)
		It("logs an unsupported resource type as debug with the item id", func() {
			service, logBuf := newLoggedService()

			// neither a space root, nor a folder, nor a file
			statResponse.Info.Type = provider.ResourceType_RESOURCE_TYPE_SYMLINK
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)

			_, err := service.ListPermissions(ctx, itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "unsupported resource type")))
			// no shares are listed after the validation failed
			for _, method := range []string{"ListShares", "ListOCMShares", "ListPublicShares"} {
				gatewayClient.AssertNotCalled(GinkgoT(), method, mock.Anything, mock.Anything)
			}

			entries := logEntries(logBuf)
			var debugEntries []map[string]any
			for _, entry := range entries {
				if entry["level"] == "debug" && entry["itemID"] == storagespace.FormatResourceID(itemID) {
					debugEntries = append(debugEntries, entry)
				}
			}
			Expect(debugEntries).To(HaveLen(1))
			Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
		})
		It("logs a public share listing failure once as error with the item id", func() {
			service, logBuf := newLoggedService()

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(&link.ListPublicSharesResponse{
				Status: status.NewInternal(context.Background(), "listing failed for /Users/alice/Private/salary-2026.ods"),
			}, nil)

			_, err := service.ListPermissions(ctx, itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, "listing failed for /Users/alice/Private/salary-2026.ods")))
			// user shares were listed before the public share lookup failed
			gatewayClient.AssertCalled(GinkgoT(), "ListShares", mock.Anything, mock.Anything)
			gatewayClient.AssertCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not list public shares"),
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(itemID)),
				HaveKeyWithValue("errorCode", "generalException"),
				HaveKeyWithValue("error", err.Error()),
			))
			for _, payloadKey := range []string{"shares", "share", "Link", "res", "response"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
		})
		It("logs a gateway selection failure while listing public shares as error in the helper and the operation with the item id", func() {
			gatewayErr := errors.New("no gateway available")
			// ListPermissions and the user share listing get a client, the public share listing does not
			gatewaySelector.EXPECT().Next().Unset()
			gatewaySelector.EXPECT().Next().Return(gatewayClient, nil).Times(2)
			gatewaySelector.EXPECT().Next().Return(nil, gatewayErr).Times(1)
			service, logBuf := newLoggedService()
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)

			_, err := service.ListPermissions(ctx, itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, gatewayErr.Error())))
			// user shares were listed, listing stops at the public share lookup
			gatewayClient.AssertCalled(GinkgoT(), "ListShares", mock.Anything, mock.Anything)
			gatewayClient.AssertNotCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)

			// the helper and the operation both log the reason, the operation with the item id and the error code
			Expect(entriesAtLevel(logEntries(logBuf), "error")).To(ConsistOf(
				gatewayHelperEntry("could not select next gateway client", gatewayErr.Error()),
				SatisfyAll(
					HaveKeyWithValue("message", "could not list public shares"),
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(itemID)),
					HaveKeyWithValue("errorCode", "generalException"),
					HaveKeyWithValue("error", err.Error()),
				),
			))
		})
		It("logs an ocm share listing failure once as error with the item id", func() {
			cfg.IncludeOCMSharees = true
			service, logBuf := newLoggedService()

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("ListOCMShares", mock.Anything, mock.Anything).Return(&ocm.ListOCMSharesResponse{
				Status: status.NewInternal(context.Background(), "ocm share manager unavailable"),
			}, nil)

			_, err := service.ListPermissions(ctx, itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, "ocm share manager unavailable")))
			// listing stops at the failed ocm share lookup
			gatewayClient.AssertCalled(GinkgoT(), "ListOCMShares", mock.Anything, mock.Anything)
			gatewayClient.AssertNotCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not list ocm shares"),
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(itemID)),
				HaveKeyWithValue("error", err.Error()),
			))
			for _, payloadKey := range []string{"shares", "share", "res", "response"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
		})
		It("logs a gateway selection failure while listing ocm shares as error in the helper and the operation with the item id", func() {
			gatewayErr := errors.New("no gateway available")
			cfg.IncludeOCMSharees = true
			// ListPermissions and the user share listing get a client, the ocm share listing does not
			gatewaySelector.EXPECT().Next().Unset()
			gatewaySelector.EXPECT().Next().Return(gatewayClient, nil).Times(2)
			gatewaySelector.EXPECT().Next().Return(nil, gatewayErr).Times(1)
			service, logBuf := newLoggedService()
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)

			_, err := service.ListPermissions(ctx, itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, gatewayErr.Error())))
			// user shares were listed, listing stops at the ocm share lookup
			gatewayClient.AssertCalled(GinkgoT(), "ListShares", mock.Anything, mock.Anything)
			gatewayClient.AssertNotCalled(GinkgoT(), "ListOCMShares", mock.Anything, mock.Anything)
			gatewayClient.AssertNotCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)

			// the helper and the operation both log the reason, the operation with the item id and the error code
			Expect(entriesAtLevel(logEntries(logBuf), "error")).To(ConsistOf(
				gatewayHelperEntry("could not select next gateway client", gatewayErr.Error()),
				SatisfyAll(
					HaveKeyWithValue("message", "could not list ocm shares"),
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(itemID)),
					HaveKeyWithValue("error", err.Error()),
				),
			))
		})
		It("logs a gateway selection failure while listing shares as error in the helper and the operation with the item id", func() {
			gatewayErr := errors.New("no gateway available")
			// ListPermissions gets a client, the share listing helper does not
			gatewaySelector.EXPECT().Next().Unset()
			gatewaySelector.EXPECT().Next().Return(gatewayClient, nil).Times(1)
			gatewaySelector.EXPECT().Next().Return(nil, gatewayErr).Times(1)
			service, logBuf := newLoggedService()
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)

			_, err := service.ListPermissions(ctx, itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, gatewayErr.Error())))
			gatewayClient.AssertNotCalled(GinkgoT(), "ListShares", mock.Anything, mock.Anything)

			// the helper and the operation both log the reason, the operation with the item id and the error code
			Expect(entriesAtLevel(logEntries(logBuf), "error")).To(ConsistOf(
				gatewayHelperEntry("could not select next gateway client", gatewayErr.Error()),
				SatisfyAll(
					HaveKeyWithValue("message", "could not list user shares"),
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(itemID)),
					HaveKeyWithValue("error", err.Error()),
				),
			))
		})
		It("logs a gateway selection failure while listing space root shares as error in the helper and the operation with the item id", func() {
			gatewayErr := errors.New("no gateway available")
			spaceRootID := &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}
			// ListPermissions gets a client, the space root share listing helper does not
			gatewaySelector.EXPECT().Next().Unset()
			gatewaySelector.EXPECT().Next().Return(gatewayClient, nil).Times(1)
			gatewaySelector.EXPECT().Next().Return(nil, gatewayErr).Times(1)
			service, logBuf := newLoggedService()
			statResponse.Info.Id = spaceRootID
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)

			_, err := service.ListPermissions(ctx, spaceRootID, svc.ListPermissionsQueryOptions{})
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, gatewayErr.Error())))
			// listing stops at the failed space root share lookup
			gatewayClient.AssertNotCalled(GinkgoT(), "ListShares", mock.Anything, mock.Anything)
			gatewayClient.AssertNotCalled(GinkgoT(), "ListPublicShares", mock.Anything, mock.Anything)

			// the helper and the operation both log the reason, the operation with the item id and the error code
			Expect(entriesAtLevel(logEntries(logBuf), "error")).To(ConsistOf(
				gatewayHelperEntry("could not select next gateway client", gatewayErr.Error()),
				SatisfyAll(
					HaveKeyWithValue("message", "could not list space root shares"),
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(spaceRootID)),
					HaveKeyWithValue("error", err.Error()),
				),
			))
		})
		It("records a ListPermissions span", func() {
			sr, tp := newSpanRecorder()
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			_, err = service.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(sr.Ended()).To(HaveLen(1))
			span := sr.Ended()[0]
			Expect(span.Name()).To(Equal("ListPermissions"))
			// a successful operation is neither marked as failed nor carries an error code
			Expect(span.Status().Code).To(Equal(codes.Unset))
			for _, attr := range span.Attributes() {
				Expect(attr.Key).ToNot(Equal(attribute.Key("error.type")))
			}
		})
		It("does not panic and does not record spans without a TraceProvider", func() {
			// a recording span from the caller, e.g. the http server span
			callerTP := sdktrace.NewTracerProvider()
			DeferCleanup(func() {
				Expect(callerTP.Shutdown(context.Background())).To(Succeed())
			})
			callerCtx, callerSpan := callerTP.Tracer("test").Start(context.Background(), "caller")
			defer callerSpan.End()

			var statCtx context.Context
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				statCtx = args.Get(0).(context.Context)
			}).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)

			Expect(func() {
				service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, nil)
				Expect(err).ToNot(HaveOccurred())
				_, err = service.ListPermissions(callerCtx, itemID, svc.ListPermissionsQueryOptions{})
				Expect(err).ToNot(HaveOccurred())
			}).NotTo(Panic())

			// the span active during the gateway call is a non-recording no-op span
			// that still carries the caller's span context
			Expect(statCtx).ToNot(BeNil())
			active := trace.SpanFromContext(statCtx)
			Expect(active.IsRecording()).To(BeFalse())
			Expect(active.SpanContext().TraceID()).To(Equal(callerSpan.SpanContext().TraceID()))
			Expect(active.SpanContext().SpanID()).To(Equal(callerSpan.SpanContext().SpanID()))
		})
		It("populates allowedValues for files that are not shared", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
		})
		It("sends SpaceRootFilter(false) when listing shares for a non-space-root item", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			gatewayClient.On("ListShares",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.ListSharesRequest) bool {
					for _, f := range req.Filters {
						if f.Type == collaboration.Filter_TYPE_SPACE_ROOT && !f.GetSpaceRoot() {
							return true
						}
					}
					return false
				}),
			).Return(listSharesResponse, nil)

			_, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			gatewayClient.AssertExpectations(GinkgoT())
		})
		It("returns one permission per share", func() {
			statResponse.Info.PermissionSet = roleconversions.NewEditorRole().CS3ResourcePermissions()
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{
				{
					Id: &link.PublicShareId{
						OpaqueId: "public-share-id",
					},
					Token: "public-share-token",
					// the link shares the same resource id
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Permissions: &link.PublicSharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
				},
			}

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.Value)).To(Equal(2))
		})
		It("still returns the user shares when listing public shares reports an already exists error", func() {
			statResponse.Info.PermissionSet = roleconversions.NewEditorRole().CS3ResourcePermissions()
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Status = status.NewInternal(ctx, errtypes.AlreadyExists("user service-user does not have access to metadata space jsoncs3-public-share-manager-metadata, but it exists").Error())

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.Value)).To(Equal(1))
		})
		It("fails when listing public shares fails for other reasons", func() {
			listPublicSharesResponse.Status = status.NewInternal(ctx, "error listing public shares")

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			_, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).To(HaveOccurred())
		})
		It("returns role denied", func() {
			statResponse.Info.PermissionSet = roleconversions.NewManagerRole().CS3ResourcePermissions()
			statResponse.Info.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER

			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewDeniedRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{}

			cfg = defaults.FullDefaultConfig()
			cfg.UnifiedRoles.AvailableRoles = []string{unifiedrole.UnifiedRoleViewerID, unifiedrole.UnifiedRoleDeniedID, unifiedrole.UnifiedRoleManagerID}
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, nil)

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := service.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.Value)).To(Equal(1))
			Expect(permissions.Value[0].GetRoles()[0]).To(Equal(unifiedrole.UnifiedRoleDeniedID))
		})
		It("returns access denied when no role and no actions are set (full denial)", func() {
			statResponse.Info.PermissionSet = roleconversions.NewManagerRole().CS3ResourcePermissions()
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewDeniedRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{}

			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.Value)).To(Equal(1))
			Expect(permissions.Value[0].GetLibreGraphPermissionsActions()[0]).To(Equal("none"))
		})
		It("Does not list public shares when requested so", func() {
			opt := svc.ListPermissionsQueryOptions{
				NoLinkPermissions: true,
			}
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, opt)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
		})
		It("Does not return permissions when the NoValues option is set", func() {
			opt := svc.ListPermissionsQueryOptions{
				NoValues: true,
			}
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{
				{
					Id: &link.PublicShareId{
						OpaqueId: "public-share-id",
					},
					Token: "public-share-token",
					// the link shares the same resource id
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Permissions: &link.PublicSharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
				},
			}
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, opt)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.Value)).To(BeZero())
		})
		It("Returns a count when the Count option is set", func() {
			opt := svc.ListPermissionsQueryOptions{
				Count: true,
			}
			listSharesResponse.Shares = []*collaboration.Share{
				{
					Id: &collaboration.ShareId{OpaqueId: "1"},
					Permissions: &collaboration.SharePermissions{
						Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions(),
					},
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Grantee: &provider.Grantee{
						Type: provider.GranteeType_GRANTEE_TYPE_USER,
						Id: &provider.Grantee_UserId{
							UserId: &userpb.UserId{
								OpaqueId: "user-id",
							},
						},
					},
				},
			}
			listPublicSharesResponse.Share = []*link.PublicShare{
				{
					Id: &link.PublicShareId{
						OpaqueId: "public-share-id",
					},
					Token: "public-share-token",
					// the link shares the same resource id
					ResourceId: &provider.ResourceId{
						StorageId: "1",
						SpaceId:   "2",
						OpaqueId:  "3",
					},
					Permissions: &link.PublicSharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
				},
			}
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(listSharesResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			permissions, err := driveItemPermissionsService.ListPermissions(context.Background(), itemID, opt)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
			Expect(len(permissions.LibreGraphPermissionsRolesAllowedValues)).ToNot(BeZero())
			count := int(permissions.GetOdataCount())
			Expect(count).To(Equal(2)) // 1 share + 1 public share
			Expect(len(permissions.Value)).To(Equal(count))
		})
	})
	Describe("ListSpaceRootPermissions", func() {
		var (
			driveId *provider.ResourceId
		)

		BeforeEach(func() {
			driveId = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
			}

			listSpacesResponse = &provider.ListStorageSpacesResponse{
				Status: status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{
					{
						Id: &provider.StorageSpaceId{
							OpaqueId: "2",
						},
					},
				},
			}
			statResponse.Info = &provider.ResourceInfo{
				Type:          provider.ResourceType_RESOURCE_TYPE_FILE,
				PermissionSet: roleconversions.NewViewerRole().CS3ResourcePermissions(),
			}
		})

		It("adds a user to a space as expected (happy path)", func() {
			listSpacesResponse.StorageSpaces[0].SpaceType = "project"
			listSpacesResponse.StorageSpaces[0].Root = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "2",
			}

			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(&collaboration.ListSharesResponse{Status: status.NewOK(ctx)}, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			statResponse.Info.Id = listSpacesResponse.StorageSpaces[0].Root
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			permissions, err := driveItemPermissionsService.ListSpaceRootPermissions(context.Background(), driveId, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(permissions.LibreGraphPermissionsActionsAllowedValues)).ToNot(BeZero())
		})

		It("records the ListPermissions span as a child of the ListSpaceRootPermissions span", func() {
			sr, tp := newSpanRecorder()
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			listSpacesResponse.StorageSpaces[0].SpaceType = "project"
			listSpacesResponse.StorageSpaces[0].Root = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "2",
			}
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(&collaboration.ListSharesResponse{Status: status.NewOK(ctx)}, nil)
			gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(listPublicSharesResponse, nil)
			statResponse.Info.Id = listSpacesResponse.StorageSpaces[0].Root
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			_, err = service.ListSpaceRootPermissions(context.Background(), driveId, svc.ListPermissionsQueryOptions{})
			Expect(err).ToNot(HaveOccurred())

			spans := map[string]sdktrace.ReadOnlySpan{}
			for _, s := range sr.Ended() {
				spans[s.Name()] = s
			}
			Expect(spans).To(HaveKey("ListSpaceRootPermissions"))
			Expect(spans).To(HaveKey("ListPermissions"))
			parent := spans["ListSpaceRootPermissions"].SpanContext()
			child := spans["ListPermissions"]
			Expect(child.Parent().SpanID()).To(Equal(parent.SpanID()))
			Expect(child.SpanContext().TraceID()).To(Equal(parent.TraceID()))
		})
	})
	Describe("GetPermission", func() {
		viewLink := func(id string) *link.PublicShare {
			return &link.PublicShare{
				Id:          &link.PublicShareId{OpaqueId: id},
				Token:       id,
				ResourceId:  driveItemId,
				Permissions: &link.PublicSharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
			}
		}

		BeforeEach(func() {
			statResponse.Info = &provider.ResourceInfo{
				Id:            driveItemId,
				Type:          provider.ResourceType_RESOURCE_TYPE_FILE,
				PermissionSet: roleconversions.NewEditorRole().CS3ResourcePermissions(),
			}
		})

		It("returns not found for a permission that is not listed for the item", func() {
			// shares are only listed for the requested item; permissions of other resources
			// must not be resolved by ID (the strict gateway mock fails on GetShare/GetPublicShare)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("ListShares", mock.Anything, mock.MatchedBy(func(req *collaboration.ListSharesRequest) bool {
				return slices.ContainsFunc(req.GetFilters(), func(f *collaboration.Filter) bool {
					return utils.ResourceIDEqual(f.GetResourceId(), driveItemId)
				})
			})).Return(&collaboration.ListSharesResponse{Status: status.NewOK(ctx)}, nil)
			listPublicSharesResponse.Share = []*link.PublicShare{viewLink("aaaaaaaaaaaaaaa")}
			gatewayClient.On("ListPublicShares", mock.Anything, mock.MatchedBy(func(req *link.ListPublicSharesRequest) bool {
				return slices.ContainsFunc(req.GetFilters(), func(f *link.ListPublicSharesRequest_Filter) bool {
					return utils.ResourceIDEqual(f.GetResourceId(), driveItemId)
				})
			})).Return(listPublicSharesResponse, nil)

			_, err := driveItemPermissionsService.GetPermission(ctx, driveItemId, "ccccccccccccccc")
			Expect(errorcode.IsErrorCode(err, errorcode.ItemNotFound)).To(BeTrue())
		})

		It("propagates stat errors without listing permissions", func() {
			statResponse.Status = status.NewPermissionDenied(ctx, nil, "denied")
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)

			_, err := driveItemPermissionsService.GetPermission(ctx, driveItemId, "aaaaaaaaaaaaaaa")
			Expect(errorcode.IsErrorCode(err, errorcode.AccessDenied)).To(BeTrue())
		})
	})

	Describe("GetSpaceRootPermission", func() {
		DescribeTable("rejects space types other than project",
			func(spaceType string) {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{{
						Id:        &provider.StorageSpaceId{OpaqueId: "1$2"},
						SpaceType: spaceType,
						Root:      &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"},
					}},
				}, nil)

				_, err := driveItemPermissionsService.GetSpaceRootPermission(ctx, &provider.ResourceId{StorageId: "1", SpaceId: "2"}, "aaaaaaaaaaaaaaa")
				Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "unsupported space type")))
			},
			Entry("personal", "personal"),
			Entry("virtual", "virtual"),
		)
	})

	Describe("DeletePermission", func() {
		var (
			getShareResponse       collaboration.GetShareResponse
			getPublicShareResponse link.GetPublicShareResponse
		)
		BeforeEach(func() {
			getPublicShareResponse.Status = status.NewOK(context.Background())
			getPublicShareResponse.Share = nil
			getShareResponse.Status = status.NewOK(context.Background())
			getShareResponse.Share = &collaboration.Share{
				Id: &collaboration.ShareId{
					OpaqueId: "permissionid",
				},
				ResourceId: &provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "3",
				},
			}
		})
		It("fails to deletes a public link permission when it can be resolved", func() {
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				getShareResponse.Share.ResourceId,
				"permissionid",
			)
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "failed to resolve resource id for shared resource")))
		})
		It("deletes a user permission as expected", func() {
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			gatewayClient.On("GetShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(&getShareResponse, nil)

			rmShareMockResponse := &collaboration.RemoveShareResponse{
				Status: status.NewOK(ctx),
			}
			gatewayClient.On("RemoveShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.RemoveShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(rmShareMockResponse, nil)

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				getShareResponse.Share.ResourceId,
				"permissionid",
			)
			Expect(err).ToNot(HaveOccurred())
		})
		It("deletes a link permission as expected", func() {
			getPublicShareResponse.Share = &link.PublicShare{
				Id: &link.PublicShareId{
					OpaqueId: "linkpermissionid",
				},
				ResourceId: &provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "3",
				},
			}
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)

			gatewayClient.On("RemovePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.RemovePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "linkpermissionid"
				}),
			).Return(
				&link.RemovePublicShareResponse{
					Status: status.NewOK(ctx),
				}, nil,
			)

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				getShareResponse.Share.ResourceId,
				"linkpermissionid",
			)
			Expect(err).ToNot(HaveOccurred())
		})
		It("logs a share removal failure once as error with the item and permission ids", func() {
			service, logBuf := newLoggedService()

			// not a public link, but a user share whose removal fails
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&getShareResponse, nil)
			gatewayClient.On("RemoveShare", mock.Anything, mock.Anything).Return(&collaboration.RemoveShareResponse{
				Status: status.NewInternal(context.Background(), "removal failed for /Users/alice/Private/salary-2026.ods"),
			}, nil)

			err := service.DeletePermission(ctx, getShareResponse.Share.ResourceId, "permissionid")
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, "removal failed for /Users/alice/Private/salary-2026.ods").WithOrigin(errorcode.ErrorOriginCS3)))
			gatewayClient.AssertCalled(GinkgoT(), "RemoveShare", mock.Anything, mock.MatchedBy(func(req *collaboration.RemoveShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			}))

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(getShareResponse.Share.ResourceId)),
				HaveKeyWithValue("permissionID", "permissionid"),
				HaveKeyWithValue("errorCode", "generalException"),
				// the error text tells operators the reason
				HaveKeyWithValue("error", err.Error()),
			))
			for _, payloadKey := range []string{"share", "res", "response", "request", "Body", "body"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
		})
		It("logs the error text when resolving a permission fails", func() {
			service, logBuf := newLoggedService()

			// not a public link, the user share lookup fails in the share manager
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&collaboration.GetShareResponse{
				Status: status.NewInternal(context.Background(), "lookup failed for /Users/alice/Private/salary-2026.ods"),
			}, nil)

			err := service.DeletePermission(ctx, getShareResponse.Share.ResourceId, "permissionid")
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, "lookup failed for /Users/alice/Private/salary-2026.ods").WithOrigin(errorcode.ErrorOriginCS3)))
			gatewayClient.AssertNotCalled(GinkgoT(), "RemoveShare", mock.Anything, mock.Anything)

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not resolve permission"),
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(getShareResponse.Share.ResourceId)),
				HaveKeyWithValue("permissionID", "permissionid"),
				HaveKeyWithValue("errorCode", "generalException"),
				HaveKeyWithValue("error", err.Error()),
			))
		})
		DescribeTable("logs permission removal failures by cause with the item and permission ids",
			func(setup func() string, expectedErr error, level string) {
				removeMethod := setup()
				service, logBuf := newLoggedService()

				err := service.DeletePermission(ctx, getShareResponse.Share.ResourceId, "permissionid")
				Expect(err).To(MatchError(expectedErr))
				// the permission was resolved and its removal was attempted
				gatewayClient.AssertCalled(GinkgoT(), removeMethod, mock.Anything, mock.Anything)

				graphErr, ok := errorcode.ToError(expectedErr)
				Expect(ok).To(BeTrue())
				entries := logEntries(logBuf)
				// the failure is logged with its error code and error text, whatever its level
				Expect(entries).To(ContainElement(SatisfyAll(
					HaveKeyWithValue("level", level),
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(getShareResponse.Share.ResourceId)),
					HaveKeyWithValue("permissionID", "permissionid"),
					HaveKeyWithValue("errorCode", graphErr.GetCode().String()),
					HaveKeyWithValue("error", expectedErr.Error()),
				)))
				errorEntries := entriesAtLevel(entries, "error")
				if level == "error" {
					Expect(errorEntries).To(HaveLen(1))
				} else {
					Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
				}
				for _, payloadKey := range []string{"share", "res", "response", "request", "Body", "body"} {
					Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
				}
			},
			Entry("public link: internal error",
				func() string {
					// resolved as a public link on the item
					getPublicShareResponse.Share = &link.PublicShare{
						Id:         &link.PublicShareId{OpaqueId: "permissionid"},
						ResourceId: getShareResponse.Share.ResourceId,
					}
					gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
					gatewayClient.On("RemovePublicShare", mock.Anything, mock.Anything).Return(&link.RemovePublicShareResponse{
						Status: status.NewInternal(context.Background(), "link manager unavailable"),
					}, nil)
					return "RemovePublicShare"
				},
				errorcode.New(errorcode.GeneralException, "link manager unavailable").WithOrigin(errorcode.ErrorOriginCS3), "error"),
			Entry("ocm share: internal error",
				func() string {
					// neither a public link nor a user share, but an ocm share on the item
					cfg.IncludeOCMSharees = true
					getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
					gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
					gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&collaboration.GetShareResponse{
						Status: status.NewNotFound(context.Background(), ""),
					}, nil)
					gatewayClient.On("GetOCMShare", mock.Anything, mock.Anything).Return(&ocm.GetOCMShareResponse{
						Status: status.NewOK(context.Background()),
						Share: &ocm.Share{
							Id:         &ocm.ShareId{OpaqueId: "permissionid"},
							ResourceId: getShareResponse.Share.ResourceId,
						},
					}, nil)
					gatewayClient.On("RemoveOCMShare", mock.Anything, mock.Anything).Return(&ocm.RemoveOCMShareResponse{
						Status: status.NewInternal(context.Background(), "ocm share manager unavailable"),
					}, nil)
					return "RemoveOCMShare"
				},
				errorcode.New(errorcode.GeneralException, "ocm share manager unavailable").WithOrigin(errorcode.ErrorOriginCS3), "error"),
			Entry("user share: already removed",
				func() string {
					// resolved as a user share, which is gone by the time it is removed
					getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
					gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
					gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&getShareResponse, nil)
					gatewayClient.On("RemoveShare", mock.Anything, mock.Anything).Return(&collaboration.RemoveShareResponse{
						Status: status.NewNotFound(context.Background(), "share already removed"),
					}, nil)
					return "RemoveShare"
				},
				errorcode.New(errorcode.ItemNotFound, "share already removed").WithOrigin(errorcode.ErrorOriginCS3), "debug"),
		)
		It("logs and traces a share lookup failure hidden by the ocm fallback once as error", func() {
			var logBuf bytes.Buffer
			sr, tp := newSpanRecorder()
			cfg.IncludeOCMSharees = true
			service, err := svc.NewDriveItemPermissionsService(log.Logger{Logger: zerolog.New(&logBuf)}, gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			// the user share lookup fails in the share manager, the ocm fallback does not know the permission
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&collaboration.GetShareResponse{
				Status: status.NewInternal(context.Background(), "lookup failed for /Users/alice/Private/salary-2026.ods"),
			}, nil)
			gatewayClient.On("GetOCMShare", mock.Anything, mock.Anything).Return(&ocm.GetOCMShareResponse{
				Status: status.NewNotFound(context.Background(), "not found"),
			}, nil)

			err = service.DeletePermission(ctx, getShareResponse.Share.ResourceId, "permissionid")
			// the ocm fallback result is still returned
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "not found").WithOrigin(errorcode.ErrorOriginCS3)))
			gatewayClient.AssertCalled(GinkgoT(), "GetOCMShare", mock.Anything, mock.Anything)

			entries := logEntries(&logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not resolve permission"),
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(getShareResponse.Share.ResourceId)),
				HaveKeyWithValue("permissionID", "permissionid"),
				HaveKeyWithValue("errorCode", "itemNotFound"),
				HaveKeyWithValue("lookupErrorCode", "generalException"),
				// the texts of both the hidden infrastructure failure and the returned not found are logged
				HaveKeyWithValue("lookupError", errorcode.New(errorcode.GeneralException, "lookup failed for /Users/alice/Private/salary-2026.ods").WithOrigin(errorcode.ErrorOriginCS3).Error()),
				HaveKeyWithValue("error", err.Error()),
			))

			// the span reports the hidden infrastructure failure by its code only
			span := endedSpan(sr, "DeletePermission", "salary-2026.ods", "lookup failed")
			Expect(span.Status().Code).To(Equal(codes.Error))
			Expect(span.Status().Description).To(Equal("generalException"))
			Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "itemNotFound")))
		})
		It("does not mark the span when no share lookup finds the permission", func() {
			sr, tp := newSpanRecorder()
			cfg.IncludeOCMSharees = true
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&collaboration.GetShareResponse{
				Status: status.NewNotFound(context.Background(), "not found"),
			}, nil)
			gatewayClient.On("GetOCMShare", mock.Anything, mock.Anything).Return(&ocm.GetOCMShareResponse{
				Status: status.NewNotFound(context.Background(), "not found"),
			}, nil)

			err = service.DeletePermission(ctx, getShareResponse.Share.ResourceId, "permissionid")
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "not found").WithOrigin(errorcode.ErrorOriginCS3)))

			span := endedSpan(sr, "DeletePermission")
			Expect(span.Status().Code).To(Equal(codes.Unset))
			Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "itemNotFound")))
		})
		It("does not mark the span of a permission that is not found without the ocm fallback", func() {
			sr, tp := newSpanRecorder()
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&collaboration.GetShareResponse{
				Status: status.NewNotFound(context.Background(), "not found"),
			}, nil)

			err = service.DeletePermission(ctx, getShareResponse.Share.ResourceId, "permissionid")
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "not found").WithOrigin(errorcode.ErrorOriginCS3)))
			gatewayClient.AssertNotCalled(GinkgoT(), "GetOCMShare", mock.Anything, mock.Anything)

			span := endedSpan(sr, "DeletePermission")
			Expect(span.Status().Code).To(Equal(codes.Unset))
			Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "itemNotFound")))
		})
		It("does not mark the span when the ocm fallback succeeds after a share lookup failure", func() {
			sr, tp := newSpanRecorder()
			cfg.IncludeOCMSharees = true
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&collaboration.GetShareResponse{
				Status: status.NewInternal(context.Background(), "share manager unavailable"),
			}, nil)
			gatewayClient.On("GetOCMShare", mock.Anything, mock.Anything).Return(&ocm.GetOCMShareResponse{
				Status: status.NewOK(context.Background()),
				Share: &ocm.Share{
					Id:         &ocm.ShareId{OpaqueId: "permissionid"},
					ResourceId: getShareResponse.Share.ResourceId,
				},
			}, nil)
			gatewayClient.On("RemoveOCMShare", mock.Anything, mock.Anything).Return(&ocm.RemoveOCMShareResponse{
				Status: status.NewOK(context.Background()),
			}, nil)

			Expect(service.DeletePermission(ctx, getShareResponse.Share.ResourceId, "permissionid")).To(Succeed())
			gatewayClient.AssertCalled(GinkgoT(), "RemoveOCMShare", mock.Anything, mock.Anything)

			span := endedSpan(sr, "DeletePermission")
			Expect(span.Status().Code).To(Equal(codes.Unset))
			Expect(span.Attributes()).ToNot(ContainElement(HaveField("Key", attribute.Key("error.type"))))
		})
		It("logs a permission without a resolvable resource id as debug with the item and permission ids", func() {
			service, logBuf := newLoggedService()

			// a public link without a resource id
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)

			err := service.DeletePermission(ctx, getShareResponse.Share.ResourceId, "permissionid")
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "failed to resolve resource id for shared resource")))

			entries := logEntries(logBuf)
			Expect(entries).To(ContainElement(SatisfyAll(
				HaveKeyWithValue("level", "debug"),
				HaveKeyWithValue("message", "could not resolve the shared resource of the permission"),
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(getShareResponse.Share.ResourceId)),
				HaveKeyWithValue("permissionID", "permissionid"),
			)))
			Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
		})
		It("does not log the not found results of permission type probing as errors", func() {
			service, logBuf := newLoggedService()

			// not a public link, but a user share
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&getShareResponse, nil)
			gatewayClient.On("RemoveShare", mock.Anything, mock.Anything).Return(&collaboration.RemoveShareResponse{Status: status.NewOK(ctx)}, nil)

			err := service.DeletePermission(ctx, getShareResponse.Share.ResourceId, "permissionid")
			Expect(err).ToNot(HaveOccurred())
			gatewayClient.AssertCalled(GinkgoT(), "RemoveShare", mock.Anything, mock.Anything)

			Expect(logEntries(logBuf)).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
		})
		It("deletes a space permission as expected", func() {
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)

			gatewayClient.On("RemoveShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.RemoveShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "shareid"
				}),
			).Return(&collaboration.RemoveShareResponse{Status: status.NewOK(ctx)}, nil)

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				&provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "2",
				},
				"shareid",
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails to delete permission when the item id does not match the shared resource's id", func() {
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			getShareResponse.Share.ResourceId = &provider.ResourceId{
				StorageId: "3",
				SpaceId:   "4",
				OpaqueId:  "5",
			}
			gatewayClient.On("GetShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(&getShareResponse, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("driveID", "1$2")
			rctx.URLParams.Add("itemID", "1$2!3")
			rctx.URLParams.Add("permissionID", "permissionid")

			ctx = context.WithValue(context.Background(), chi.RouteCtxKey, rctx)

			err := driveItemPermissionsService.DeletePermission(context.Background(),
				&provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "3",
				},
				"permissionid",
			)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "permissionID and itemID do not match")))

		})
		It("logs a permission and item id mismatch as debug with the item and permission ids", func() {
			service, logBuf := newLoggedService()

			// not a public link, but a user share on another resource than the one in the request
			getPublicShareResponse.Status = status.NewNotFound(context.Background(), "")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&getPublicShareResponse, nil)
			getShareResponse.Share.ResourceId = &provider.ResourceId{StorageId: "3", SpaceId: "4", OpaqueId: "5"}
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(&getShareResponse, nil)

			itemID := &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"}
			err := service.DeletePermission(ctx, itemID, "permissionid")
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "permissionID and itemID do not match")))
			// nothing is removed after the mismatch
			for _, method := range []string{"RemoveShare", "RemovePublicShare", "RemoveOCMShare"} {
				gatewayClient.AssertNotCalled(GinkgoT(), method, mock.Anything, mock.Anything)
			}

			entries := logEntries(logBuf)
			var mismatchEntries []map[string]any
			for _, entry := range entries {
				if entry["level"] == "debug" && entry["itemID"] == storagespace.FormatResourceID(itemID) && entry["permissionID"] == "permissionid" {
					mismatchEntries = append(mismatchEntries, entry)
				}
			}
			Expect(mismatchEntries).To(HaveLen(1))
			Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
		})
	})
	Describe("DeleteSpaceRootPermission", func() {
		It("records a DeleteSpaceRootPermission span as the parent of the DeletePermission span", func() {
			sr, tp := newSpanRecorder()
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			spaceRoot := &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status: status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{
					{Id: &provider.StorageSpaceId{OpaqueId: "2"}, SpaceType: "project", Root: spaceRoot},
				},
			}, nil)
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(&link.GetPublicShareResponse{
				Status: status.NewNotFound(ctx, ""),
			}, nil)
			gatewayClient.On("RemoveShare", mock.Anything, mock.Anything).Return(&collaboration.RemoveShareResponse{Status: status.NewOK(ctx)}, nil)

			err = service.DeleteSpaceRootPermission(context.Background(), &provider.ResourceId{StorageId: "1", SpaceId: "2"}, "shareid")
			Expect(err).ToNot(HaveOccurred())
			// the space was resolved and DeletePermission removed the space permission
			gatewayClient.AssertCalled(GinkgoT(), "RemoveShare", mock.Anything, mock.MatchedBy(func(req *collaboration.RemoveShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "shareid"
			}))

			spans := map[string]sdktrace.ReadOnlySpan{}
			for _, s := range sr.Ended() {
				spans[s.Name()] = s
			}
			Expect(spans).To(HaveKey("DeleteSpaceRootPermission"))
			Expect(spans).To(HaveKey("DeletePermission"))
			parent := spans["DeleteSpaceRootPermission"].SpanContext()
			child := spans["DeletePermission"]
			Expect(parent.IsValid()).To(BeTrue())
			Expect(child.SpanContext().IsValid()).To(BeTrue())
			Expect(child.Parent().SpanID()).To(Equal(parent.SpanID()))
			Expect(child.SpanContext().TraceID()).To(Equal(parent.TraceID()))
		})
	})
	Describe("UpdatePermission", func() {
		var (
			driveItemPermission           libregraph.Permission
			getShareMockResponse          *collaboration.GetShareResponse
			getPublicShareMockResponse    *link.GetPublicShareResponse
			updateShareMockResponse       *collaboration.UpdateShareResponse
			updatePublicShareMockResponse *link.UpdatePublicShareResponse
		)
		const TestLinkName = "Test Link"
		BeforeEach(func() {
			ctx = revactx.ContextSetUser(context.Background(), currentUser)
			driveItemPermission = libregraph.Permission{}

			share := &collaboration.Share{
				Id: &collaboration.ShareId{
					OpaqueId: "permissionid",
				},
				ResourceId: driveItemId,
				Grantee: &provider.Grantee{
					Type: provider.GranteeType_GRANTEE_TYPE_USER,
					Id: &provider.Grantee_UserId{
						UserId: &userpb.UserId{
							OpaqueId: "userid",
						},
					},
				},
				Permissions: &collaboration.SharePermissions{
					Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions(),
				},
			}
			getShareMockResponse = &collaboration.GetShareResponse{
				Status: status.NewOK(ctx),
				Share:  share,
			}

			updateShareMockResponse = &collaboration.UpdateShareResponse{
				Status: status.NewOK(ctx),
				Share:  share,
			}

			updatePublicShareMockResponse = &link.UpdatePublicShareResponse{
				Status: status.NewOK(ctx),
				Share:  &link.PublicShare{DisplayName: TestLinkName},
			}

			getPublicShareMockResponse = &link.GetPublicShareResponse{
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
			statResponse = &provider.StatResponse{
				Status: status.NewOK(ctx),
				Info: &provider.ResourceInfo{
					Id:   driveItemId,
					Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER,
				},
			}

			grantMapJSON, _ := json.Marshal(
				map[string]*provider.ResourcePermissions{
					"userid": roleconversions.NewSpaceViewerRole().CS3ResourcePermissions(),
				},
			)
			spaceOpaque := &types.Opaque{
				Map: map[string]*types.OpaqueEntry{
					"grants": {
						Decoder: "json",
						Value:   grantMapJSON,
					},
				},
			}
			listSpacesResponse = &provider.ListStorageSpacesResponse{
				Status: status.NewOK(ctx),
				StorageSpaces: []*provider.StorageSpace{
					{
						Id: &provider.StorageSpaceId{
							OpaqueId: "2",
						},
						Opaque: spaceOpaque,
					},
				},
			}
		})
		It("fails when no share is found", func() {
			getShareMockResponse.Share = nil
			getShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)

			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(HaveOccurred())
			Expect(res).To(BeZero())
		})
		It("logs a gateway selection failure while listing space root shares as error in the helper and the operation with the item and permission ids", func() {
			gatewayErr := errors.New("no gateway available")
			spaceRootID := &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}
			// getPermissionByID and the public link lookup get a client, the space root share listing does not
			gatewaySelector.EXPECT().Next().Unset()
			gatewaySelector.EXPECT().Next().Return(gatewayClient, nil).Times(2)
			gatewaySelector.EXPECT().Next().Return(nil, gatewayErr).Times(1)
			service, logBuf := newLoggedService()
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)

			_, err := service.UpdatePermission(ctx, spaceRootID, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, gatewayErr.Error())))
			// the lookup stops at the failed space root share listing, nothing is updated
			for _, method := range []string{"ListShares", "UpdateShare", "UpdatePublicShare", "Stat"} {
				gatewayClient.AssertNotCalled(GinkgoT(), method, mock.Anything, mock.Anything)
			}

			entries := logEntries(logBuf)
			// the helper and the operation both log the reason, the operation with the item and permission ids
			Expect(entriesAtLevel(entries, "error")).To(ConsistOf(
				gatewayHelperEntry("could not select next gateway client", gatewayErr.Error()),
				SatisfyAll(
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(spaceRootID)),
					HaveKeyWithValue("permissionID", "permissionid"),
					HaveKeyWithValue("error", err.Error()),
				),
			))
			for _, payloadKey := range []string{"share", "shares", "res", "response", "request", "Body", "body"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
		})
		It("logs the identity backend error when resolving a user share grantee fails", func() {
			service, logBuf := newLoggedService()

			const identityBackendMessage = "ldap search failed for cn=private-group-dn-42,ou=groups"
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			getShareMockResponse.Share.Grantee = &provider.Grantee{
				Type: provider.GranteeType_GRANTEE_TYPE_GROUP,
				Id:   &provider.Grantee_GroupId{GroupId: &grouppb.GroupId{OpaqueId: "groupid"}},
			}
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(getShareMockResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetGroup", mock.Anything, mock.Anything).Return(&grouppb.GetGroupResponse{
				Status: status.NewInternal(context.Background(), identityBackendMessage),
			}, nil)

			_, err := service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			// the returned error is unchanged: the share conversion error
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException,
				errorcode.New(errorcode.GeneralException, identityBackendMessage).Error())))
			// the user share grantee was looked up, nothing is updated
			gatewayClient.AssertCalled(GinkgoT(), "GetGroup", mock.Anything, mock.MatchedBy(func(req *grouppb.GetGroupRequest) bool {
				return req.GetGroupId().GetOpaqueId() == "groupid"
			}))
			for _, method := range []string{"UpdateShare", "UpdatePublicShare"} {
				gatewayClient.AssertNotCalled(GinkgoT(), method, mock.Anything, mock.Anything)
			}

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(driveItemId)),
				HaveKeyWithValue("permissionID", "permissionid"),
				HaveKeyWithValue("error", err.Error()),
			))
			for _, payloadKey := range []string{"share", "shares", "group", "res", "response", "request", "Body", "body"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
		})
		DescribeTable("logs public link update failures as error with the item and permission ids",
			func(setup func(), expectedErr error, helperEntries int) {
				setup()
				service, logBuf := newLoggedService()

				sharingLink := libregraph.NewSharingLink()
				sharingLink.SetLibreGraphDisplayName(TestLinkName)
				driveItemPermission.SetLink(*sharingLink)
				_, err := service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
				Expect(err).To(MatchError(expectedErr))
				// the permission was resolved as a public link, user and ocm shares are not touched
				gatewayClient.AssertCalled(GinkgoT(), "GetPublicShare", mock.Anything, mock.Anything)
				for _, method := range []string{"GetShare", "UpdateShare", "GetOCMShare"} {
					gatewayClient.AssertNotCalled(GinkgoT(), method, mock.Anything, mock.Anything)
				}

				entries := logEntries(logBuf)
				// the operation's entry carries the ids, a helper that cannot select a gateway logs the reason
				var operationEntries, helpers []map[string]any
				for _, entry := range entriesAtLevel(entries, "error") {
					if _, ok := entry["itemID"]; ok {
						operationEntries = append(operationEntries, entry)
					} else {
						helpers = append(helpers, entry)
					}
				}
				Expect(operationEntries).To(HaveLen(1))
				Expect(operationEntries[0]).To(SatisfyAll(
					HaveKeyWithValue("itemID", storagespace.FormatResourceID(driveItemId)),
					HaveKeyWithValue("permissionID", "permissionid"),
					HaveKeyWithValue("error", expectedErr.Error()),
				))
				Expect(helpers).To(HaveLen(helperEntries))
				for _, entry := range helpers {
					Expect(entry).To(gatewayHelperEntry("could not select next gateway client", "no gateway available"))
				}
				for _, payloadKey := range []string{"share", "link", "Link", "res", "response", "request", "Body", "body", "password"} {
					Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
				}
				// the public link token must never be logged
				raw, err := json.Marshal(entries)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(raw)).ToNot(ContainSubstring(`"token"`))
			},
			Entry("UpdatePublicShare returns an internal error",
				func() {
					gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
					gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
					gatewayClient.On("UpdatePublicShare", mock.Anything, mock.Anything).Return(&link.UpdatePublicShareResponse{
						Status: status.NewInternal(ctx, "link manager unavailable"),
					}, nil)
				},
				errorcode.New(errorcode.GeneralException, "link manager unavailable").WithOrigin(errorcode.ErrorOriginCS3), 0),
			Entry("gateway selection fails in the public link update",
				func() {
					// getPermissionByID and the public link lookup get a client, the link update does not
					gatewaySelector.EXPECT().Next().Unset()
					gatewaySelector.EXPECT().Next().Return(gatewayClient, nil).Times(2)
					gatewaySelector.EXPECT().Next().Return(nil, errors.New("no gateway available")).Times(1)
					gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
				},
				errorcode.New(errorcode.GeneralException, "no gateway available"), 1),
		)
		It("logs a missing public link item at debug level with the backend stat message", func() {
			service, logBuf := newLoggedService()

			// the permission resolves to a public link, but its item can no longer be stat'ed;
			// decomposedfs reports a missing node as "<parent id>/<name>"
			statMessage := "stat: 7c3e9a1b-parent/private-financial-report-2026.xlsx"
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
				Status: status.NewNotFound(ctx, statMessage),
			}, nil)

			sharingLink := libregraph.NewSharingLink()
			sharingLink.SetLibreGraphDisplayName(TestLinkName)
			driveItemPermission.SetLink(*sharingLink)
			_, err := service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, statMessage).WithOrigin(errorcode.ErrorOriginCS3)))
			gatewayClient.AssertCalled(GinkgoT(), "Stat", mock.Anything, mock.Anything)
			// nothing is updated after the failed stat
			for _, method := range []string{"UpdatePublicShare", "UpdateShare", "UpdateOCMShare"} {
				gatewayClient.AssertNotCalled(GinkgoT(), method, mock.Anything, mock.Anything)
			}

			entries := logEntries(logBuf)
			var debugEntries []map[string]any
			for _, entry := range entries {
				if entry["level"] == "debug" && entry["itemID"] == storagespace.FormatResourceID(driveItemId) && entry["permissionID"] == "permissionid" {
					debugEntries = append(debugEntries, entry)
				}
			}
			Expect(debugEntries).To(HaveLen(1))
			// the debug entry carries the backend message, so the missing item can be identified
			Expect(debugEntries[0]).To(HaveKeyWithValue("error", ContainSubstring(statMessage)))
			Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
		})
		It("logs a user share update failure once as error with the item and permission ids", func() {
			service, logBuf := newLoggedService()

			// resolved as a user share on the item, whose update fails
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(getShareMockResponse, nil)
			gatewayClient.On("UpdateShare", mock.Anything, mock.Anything).Return(&collaboration.UpdateShareResponse{
				Status: status.NewInternal(ctx, "share manager unavailable"),
			}, nil)

			driveItemPermission.SetExpirationDateTime(time.Now().Add(time.Hour))
			_, err := service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException, "share manager unavailable").WithOrigin(errorcode.ErrorOriginCS3)))
			gatewayClient.AssertCalled(GinkgoT(), "UpdateShare", mock.Anything, mock.Anything)

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(driveItemId)),
				HaveKeyWithValue("permissionID", "permissionid"),
				HaveKeyWithValue("errorCode", "generalException"),
				HaveKeyWithValue("error", err.Error()),
			))
			for _, payloadKey := range []string{"share", "shares", "res", "response", "request", "Body", "body"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
		})
		It("logs and traces a failed ocm share update once with the cause of both update attempts", func() {
			var logBuf bytes.Buffer
			sr, tp := newSpanRecorder()
			cfg.IncludeOCMSharees = true
			service, err := svc.NewDriveItemPermissionsService(log.Logger{Logger: zerolog.New(&logBuf)}, gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			// the permission is neither a public link nor a user share, but an ocm share
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			getShareMockResponse.Share = nil
			getShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(getShareMockResponse, nil)
			gatewayClient.On("GetOCMShare", mock.Anything, mock.Anything).Return(&ocm.GetOCMShareResponse{
				Status: status.NewOK(ctx),
				Share:  &ocm.Share{Id: &ocm.ShareId{OpaqueId: "permissionid"}, ResourceId: driveItemId},
			}, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			// the user share update attempt does not find the share, the ocm share update fails
			gatewayClient.On("UpdateShare", mock.Anything, mock.Anything).Return(&collaboration.UpdateShareResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)
			gatewayClient.On("UpdateOCMShare", mock.Anything, mock.Anything).Return(&ocm.UpdateOCMShareResponse{
				Status: status.NewInternal(ctx, "ocm share manager unavailable"),
			}, nil)

			driveItemPermission.SetExpirationDateTime(time.Now().Add(time.Hour))
			_, err = service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			// the returned error is unchanged: the error of the user share update attempt
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "not found").WithOrigin(errorcode.ErrorOriginCS3)))
			gatewayClient.AssertCalled(GinkgoT(), "UpdateOCMShare", mock.Anything, mock.Anything)

			entries := logEntries(&logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not update permission"),
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(driveItemId)),
				HaveKeyWithValue("permissionID", "permissionid"),
				HaveKeyWithValue("errorCode", "itemNotFound"),
				HaveKeyWithValue("ocmErrorCode", "generalException"),
				// the texts of both the failed ocm update and the returned not found are logged
				HaveKeyWithValue("ocmError", errorcode.New(errorcode.GeneralException, "ocm share manager unavailable").WithOrigin(errorcode.ErrorOriginCS3).Error()),
				HaveKeyWithValue("error", err.Error()),
			))

			// the span reports the failed ocm update by its code only
			span := endedSpan(sr, "UpdatePermission", "ocm share manager unavailable")
			Expect(span.Status().Code).To(Equal(codes.Error))
			Expect(span.Status().Description).To(Equal("generalException"))
			Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "itemNotFound")))
		})
		It("does not mark the span when neither update attempt finds the share", func() {
			sr, tp := newSpanRecorder()
			cfg.IncludeOCMSharees = true
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			getShareMockResponse.Share = nil
			getShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(getShareMockResponse, nil)
			gatewayClient.On("GetOCMShare", mock.Anything, mock.Anything).Return(&ocm.GetOCMShareResponse{
				Status: status.NewOK(ctx),
				Share:  &ocm.Share{Id: &ocm.ShareId{OpaqueId: "permissionid"}, ResourceId: driveItemId},
			}, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("UpdateShare", mock.Anything, mock.Anything).Return(&collaboration.UpdateShareResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)
			gatewayClient.On("UpdateOCMShare", mock.Anything, mock.Anything).Return(&ocm.UpdateOCMShareResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)

			driveItemPermission.SetExpirationDateTime(time.Now().Add(time.Hour))
			_, err = service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "not found").WithOrigin(errorcode.ErrorOriginCS3)))
			gatewayClient.AssertCalled(GinkgoT(), "UpdateOCMShare", mock.Anything, mock.Anything)

			span := endedSpan(sr, "UpdatePermission")
			Expect(span.Status().Code).To(Equal(codes.Unset))
			Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "itemNotFound")))
		})
		It("logs the identity backend error when converting the updated user share fails", func() {
			service, logBuf := newLoggedService()

			// resolved as a user share on the item, the updated share is returned with a
			// group grantee that has not been looked up (and cached) before
			const identityBackendMessage = "ldap search failed for cn=private-group-dn-42,ou=groups"
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(getShareMockResponse, nil)
			updateShareMockResponse.Share = &collaboration.Share{
				Id:         &collaboration.ShareId{OpaqueId: "permissionid"},
				ResourceId: driveItemId,
				Grantee: &provider.Grantee{
					Type: provider.GranteeType_GRANTEE_TYPE_GROUP,
					Id:   &provider.Grantee_GroupId{GroupId: &grouppb.GroupId{OpaqueId: "groupid"}},
				},
			}
			gatewayClient.On("UpdateShare", mock.Anything, mock.Anything).Return(updateShareMockResponse, nil)
			gatewayClient.On("GetGroup", mock.Anything, mock.Anything).Return(&grouppb.GetGroupResponse{
				Status: status.NewInternal(ctx, identityBackendMessage),
			}, nil)

			driveItemPermission.SetExpirationDateTime(time.Now().Add(time.Hour))
			_, err := service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			// the returned error is unchanged: the identity error wrapped by the share conversion
			Expect(err).To(MatchError(errorcode.New(errorcode.GeneralException,
				errorcode.New(errorcode.GeneralException, identityBackendMessage).Error())))
			gatewayClient.AssertCalled(GinkgoT(), "GetGroup", mock.Anything, mock.Anything)

			entries := logEntries(logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(driveItemId)),
				HaveKeyWithValue("permissionID", "permissionid"),
				HaveKeyWithValue("errorCode", "generalException"),
				HaveKeyWithValue("error", err.Error()),
			))
		})
		It("does not log a permission that is not found by any lookup as error", func() {
			service, logBuf := newLoggedService()

			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			getShareMockResponse.Share = nil
			getShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(getShareMockResponse, nil)

			_, err := service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			Expect(errorcode.IsErrorCode(err, errorcode.ItemNotFound)).To(BeTrue())

			Expect(logEntries(logBuf)).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
		})
		It("logs and traces a share lookup failure hidden by the ocm fallback once as error", func() {
			var logBuf bytes.Buffer
			sr, tp := newSpanRecorder()
			cfg.IncludeOCMSharees = true
			service, err := svc.NewDriveItemPermissionsService(log.Logger{Logger: zerolog.New(&logBuf)}, gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			// the user share lookup fails in the share manager, the ocm fallback does not know the permission
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			getShareMockResponse.Share = nil
			getShareMockResponse.Status = status.NewInternal(ctx, "lookup failed for /Users/alice/Private/salary-2026.ods")
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(getShareMockResponse, nil)
			gatewayClient.On("GetOCMShare", mock.Anything, mock.Anything).Return(&ocm.GetOCMShareResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)

			_, err = service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			// the ocm fallback result is still returned
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "not found").WithOrigin(errorcode.ErrorOriginCS3)))
			gatewayClient.AssertCalled(GinkgoT(), "GetOCMShare", mock.Anything, mock.Anything)

			entries := logEntries(&logBuf)
			errorEntries := entriesAtLevel(entries, "error")
			Expect(errorEntries).To(HaveLen(1))
			Expect(errorEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("message", "could not resolve permission"),
				HaveKeyWithValue("itemID", storagespace.FormatResourceID(driveItemId)),
				HaveKeyWithValue("permissionID", "permissionid"),
				HaveKeyWithValue("errorCode", "itemNotFound"),
				HaveKeyWithValue("lookupErrorCode", "generalException"),
				// the texts of both the hidden infrastructure failure and the returned not found are logged
				HaveKeyWithValue("lookupError", errorcode.New(errorcode.GeneralException, "lookup failed for /Users/alice/Private/salary-2026.ods").WithOrigin(errorcode.ErrorOriginCS3).Error()),
				HaveKeyWithValue("error", err.Error()),
			))

			// the span reports the hidden infrastructure failure by its code only
			span := endedSpan(sr, "UpdatePermission", "salary-2026.ods", "lookup failed")
			Expect(span.Status().Code).To(Equal(codes.Error))
			Expect(span.Status().Description).To(Equal("generalException"))
			Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "itemNotFound")))
		})
		It("does not mark the span when neither permission lookup finds the permission", func() {
			sr, tp := newSpanRecorder()
			cfg.IncludeOCMSharees = true
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			getShareMockResponse.Share = nil
			getShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetShare", mock.Anything, mock.Anything).Return(getShareMockResponse, nil)
			gatewayClient.On("GetOCMShare", mock.Anything, mock.Anything).Return(&ocm.GetOCMShareResponse{
				Status: status.NewNotFound(ctx, "not found"),
			}, nil)

			_, err = service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.ItemNotFound, "not found").WithOrigin(errorcode.ErrorOriginCS3)))

			span := endedSpan(sr, "UpdatePermission")
			Expect(span.Status().Code).To(Equal(codes.Unset))
			Expect(span.Attributes()).To(ContainElement(attribute.String("error.type", "itemNotFound")))
		})
		It("fails to update permission when the resourceID mismatches with the shared resource's id", func() {
			getShareMockResponse.Share = nil
			getShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			getPublicShareMockResponse.Share.ResourceId = &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "4",
			}
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTime(time.Now().Add(time.Hour))
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "permissionID and itemID do not match")))
			Expect(res).To(BeZero())
		})
		It("logs a permission and item id mismatch as debug with the item and permission ids", func() {
			service, logBuf := newLoggedService()

			// the link belongs to another item than the one in the request
			getPublicShareMockResponse.Share.ResourceId = &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "4"}
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTime(time.Now().Add(time.Hour))
			_, err := service.UpdatePermission(ctx, driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "permissionID and itemID do not match")))
			// nothing is updated after the mismatch
			for _, method := range []string{"Stat", "UpdatePublicShare", "UpdateShare", "UpdateOCMShare"} {
				gatewayClient.AssertNotCalled(GinkgoT(), method, mock.Anything, mock.Anything)
			}

			entries := logEntries(logBuf)
			var mismatchEntries []map[string]any
			for _, entry := range entries {
				if entry["level"] == "debug" && entry["itemID"] == storagespace.FormatResourceID(driveItemId) && entry["permissionID"] == "permissionid" {
					mismatchEntries = append(mismatchEntries, entry)
				}
			}
			Expect(mismatchEntries).To(HaveLen(1))
			Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
		})
		It("succeeds when trying to update a link permission with displayname", func() {
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("Stat", mock.Anything, mock.MatchedBy(func(req *provider.StatRequest) bool {
				return utils.ResourceIDEqual(req.GetRef().GetResourceId(), driveItemId) && req.GetRef().GetPath() == "."
			})).Return(statResponse, nil)

			gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					if req.GetRef().GetId().GetOpaqueId() == "permissionid" {
						return req.Update.GetDisplayName() == TestLinkName
					}
					return false
				}),
			).Return(updatePublicShareMockResponse, nil)

			link := libregraph.NewSharingLink()
			link.SetLibreGraphDisplayName(TestLinkName)

			driveItemPermission.SetLink(*link)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Link).ToNot(BeNil())
			Expect(res.Link.GetLibreGraphDisplayName() == TestLinkName)
		})
		It("records an UpdateSpaceRootPermission span as the parent of the UpdatePermission span", func() {
			sr, tp := newSpanRecorder()
			service, err := svc.NewDriveItemPermissionsService(log.NewLogger(), gatewaySelector, cache, cfg, tp)
			Expect(err).ToNot(HaveOccurred())

			spaceRoot := &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}
			listSpacesResponse.StorageSpaces[0].SpaceType = "project"
			listSpacesResponse.StorageSpaces[0].Root = spaceRoot
			getPublicShareMockResponse.Share.ResourceId = spaceRoot
			statResponse.Info.Id = spaceRoot
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listSpacesResponse, nil)
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("UpdatePublicShare", mock.Anything, mock.Anything).Return(updatePublicShareMockResponse, nil)

			sharingLink := libregraph.NewSharingLink()
			sharingLink.SetLibreGraphDisplayName(TestLinkName)
			driveItemPermission.SetLink(*sharingLink)
			res, err := service.UpdateSpaceRootPermission(context.Background(), &provider.ResourceId{StorageId: "1", SpaceId: "2"}, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			// the space was resolved and UpdatePermission updated the link on the space root
			Expect(res.Link).ToNot(BeNil())
			Expect(res.Link.GetLibreGraphDisplayName()).To(Equal(TestLinkName))
			gatewayClient.AssertCalled(GinkgoT(), "UpdatePublicShare", mock.Anything, mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid" && req.GetUpdate().GetDisplayName() == TestLinkName
			}))

			spans := map[string]sdktrace.ReadOnlySpan{}
			for _, s := range sr.Ended() {
				spans[s.Name()] = s
			}
			Expect(spans).To(HaveKey("UpdateSpaceRootPermission"))
			Expect(spans).To(HaveKey("UpdatePermission"))
			parent := spans["UpdateSpaceRootPermission"].SpanContext()
			child := spans["UpdatePermission"]
			Expect(parent.IsValid()).To(BeTrue())
			Expect(child.SpanContext().IsValid()).To(BeTrue())
			Expect(child.Parent().SpanID()).To(Equal(parent.SpanID()))
			Expect(child.SpanContext().TraceID()).To(Equal(parent.TraceID()))
		})
		It("updates the expiration date", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)

			expiration := time.Now().Add(time.Hour)
			updateShareMockResponse.Share.Expiration = utils.TimeToTS(expiration)
			gatewayClient.On("UpdateShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.UpdateShareRequest) bool {
					if req.GetShare().GetId().GetOpaqueId() == "permissionid" {
						return expiration.Equal(utils.TSToTime(req.GetShare().GetExpiration()))
					}
					return false
				}),
			).Return(updateShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTime(expiration)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.GetExpirationDateTime().Equal(expiration)).To(BeTrue())
		})
		It("deletes the expiration date", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)
			gatewayClient.On("UpdateShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.UpdateShareRequest) bool {
					return req.GetShare().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updateShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTimeNil()
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			_, ok := res.GetExpirationDateTimeOk()
			Expect(ok).To(BeFalse())
		})
		It("fails to update the share permissions for a file share when setting a space specific role", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)

			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)

			driveItemPermission.SetRoles([]string{unifiedrole.UnifiedRoleSpaceViewerID})
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(res).To(BeZero())
		})
		It("fails to update the space permissions for a space share when setting a file specific role", func() {
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)

			spaceId := &provider.ResourceId{
				StorageId: "1",
				SpaceId:   "2",
				OpaqueId:  "2",
			}
			statResponse.Info.Id = spaceId
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)

			gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(&collaboration.ListSharesResponse{
				Status: status.NewOK(ctx),
				Shares: []*collaboration.Share{
					{
						Id:         &collaboration.ShareId{OpaqueId: "spaceShareId"},
						ResourceId: spaceId,
						Grantee: &provider.Grantee{
							Type: provider.GranteeType_GRANTEE_TYPE_USER,
							Id: &provider.Grantee_UserId{
								UserId: &userpb.UserId{OpaqueId: "userid"},
							},
						},
						Permissions: &collaboration.SharePermissions{
							Permissions: roleconversions.NewSpaceViewerRole().CS3ResourcePermissions(),
						},
					},
				},
			}, nil)
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)

			driveItemPermission.SetRoles([]string{unifiedrole.UnifiedRoleFileEditorID})
			res, err := driveItemPermissionsService.UpdatePermission(ctx, spaceId, "spaceShareId", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "role not applicable to this resource")))
			Expect(res).To(BeZero())
		})
		It("updates the share permissions when changing the resource permission actions", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			getPublicShareMockResponse.Share = nil
			getPublicShareMockResponse.Status = status.NewNotFound(ctx, "not found")
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)

			gatewayClient.On("GetShare", mock.Anything, mock.MatchedBy(func(req *collaboration.GetShareRequest) bool {
				return req.GetRef().GetId().GetOpaqueId() == "permissionid"
			})).Return(getShareMockResponse, nil)

			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)

			updateShareMockResponse.Share.Permissions = &collaboration.SharePermissions{
				Permissions: &provider.ResourcePermissions{
					GetPath: true,
				},
			}
			gatewayClient.On("UpdateShare",
				mock.Anything,
				mock.MatchedBy(func(req *collaboration.UpdateShareRequest) bool {
					return req.GetShare().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updateShareMockResponse, nil)

			driveItemPermission.SetLibreGraphPermissionsActions([]string{unifiedrole.DriveItemPathRead})
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			_, ok := res.GetRolesOk()
			Expect(ok).To(BeFalse())
			_, ok = res.GetLibreGraphPermissionsActionsOk()
			Expect(ok).To(BeTrue())
		})
		It("updates the expiration date on a public share", func() {
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)

			expiration := time.Now().UTC().Add(time.Hour)
			updatePublicShareMockResponse.Share.Expiration = utils.TimeToTS(expiration)
			gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updatePublicShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTime(expiration)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.GetExpirationDateTime().Equal(expiration)).To(BeTrue())
		})
		It("updates the permissions on a public share", func() {
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)

			newLink := libregraph.NewSharingLink()
			newLinkType, err := libregraph.NewSharingLinkTypeFromValue("edit")
			Expect(err).ToNot(HaveOccurred())
			newLink.SetType(*newLinkType)

			updatePublicShareMockResponse.Share.Permissions = &link.PublicSharePermissions{
				Permissions: linktype.NewFolderEditLinkPermissionSet().Permissions,
			}
			gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updatePublicShareMockResponse, nil)

			driveItemPermission.SetLink(*newLink)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			linkType := res.Link.GetType()
			Expect(string(linkType)).To(Equal("edit"))
		})
		It("updates the public share to internal link", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			getPublicShareMockResponse.Share = &link.PublicShare{
				Id: &link.PublicShareId{
					OpaqueId: "permissionid",
				},
				ResourceId: &provider.ResourceId{
					StorageId: "1",
					SpaceId:   "2",
					OpaqueId:  "3",
				},
				PasswordProtected: true,
				Permissions: &link.PublicSharePermissions{
					Permissions: linktype.NewFileEditLinkPermissionSet().GetPermissions(),
				},
				Token: "token",
			}
			gatewayClient.On("GetPublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.GetPublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(getPublicShareMockResponse, nil)

			newLink := libregraph.NewSharingLink()
			newLinkType, err := libregraph.NewSharingLinkTypeFromValue("internal")
			Expect(err).ToNot(HaveOccurred())
			newLink.SetType(*newLinkType)

			updatePublicShareMockResponse.Share.Permissions = &link.PublicSharePermissions{
				Permissions: linktype.NewInternalLinkPermissionSet().Permissions,
			}
			gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			).Return(updatePublicShareMockResponse, nil)

			driveItemPermission.SetLink(*newLink)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).ToNot(HaveOccurred())
			linkType := res.Link.GetType()
			Expect(string(linkType)).To(Equal("internal"))
			pp, hasPP := res.GetHasPasswordOk()
			Expect(hasPP).To(Equal(true))
			Expect(*pp).To(Equal(false))
		})
		It("fails when updating the expiration date on a public share", func() {
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(statResponse, nil)
			gatewayClient.On("GetPublicShare", mock.Anything, mock.Anything).Return(getPublicShareMockResponse, nil)
			expiration := time.Now().UTC().AddDate(0, 0, -1)
			updatePublicShareMock := gatewayClient.On("UpdatePublicShare",
				mock.Anything,
				mock.MatchedBy(func(req *link.UpdatePublicShareRequest) bool {
					return req.GetRef().GetId().GetOpaqueId() == "permissionid"
				}),
			)

			updatePublicShareMockResponse.Share = nil
			updatePublicShareMockResponse.Status = status.NewFailedPrecondition(ctx, nil, "expiration date is in the past")
			updatePublicShareMock.Return(updatePublicShareMockResponse, nil)

			driveItemPermission.SetExpirationDateTime(expiration)
			res, err := driveItemPermissionsService.UpdatePermission(context.Background(), driveItemId, "permissionid", driveItemPermission)
			Expect(err).To(MatchError(errorcode.New(errorcode.InvalidRequest, "expiration date is in the past").WithOrigin(errorcode.ErrorOriginCS3)))
			Expect(res).To(BeZero())
		})
	})

})

var _ = Describe("DriveItemPermissionsApi", func() {
	var (
		mockProvider *mocks.DriveItemPermissionsProvider
		httpAPI      svc.DriveItemPermissionsApi
		rCTX         *chi.Context
		invite       libregraph.DriveItemInvite
	)

	BeforeEach(func() {
		logger := log.NewLogger()

		mockProvider = mocks.NewDriveItemPermissionsProvider(GinkgoT())
		api, err := svc.NewDriveItemPermissionsApi(mockProvider, logger, defaults.FullDefaultConfig())
		Expect(err).ToNot(HaveOccurred())

		httpAPI = api

		rCTX = chi.NewRouteContext()
		rCTX.URLParams.Add("driveID", "1$2")

		invite = libregraph.DriveItemInvite{
			Recipients: []libregraph.DriveRecipient{
				{
					ObjectId:                libregraph.PtrString("1"),
					LibreGraphRecipientType: libregraph.PtrString("user")},
			},
			Roles: []string{unifiedrole.UnifiedRoleViewerID},
		}
	})

	DescribeTable("logs link password service failures by cause with the resource and permission ids",
		func(spaceRoot bool, serviceErr error, expectedStatus int, level string) {
			var logBuf bytes.Buffer
			api, err := svc.NewDriveItemPermissionsApi(mockProvider, log.Logger{Logger: zerolog.New(&logBuf)}, defaults.FullDefaultConfig())
			Expect(err).ToNot(HaveOccurred())

			rCTX.URLParams.Add("permissionID", "permissionid")
			handler, method, ids := api.SetLinkPassword, "SetPublicLinkPassword", map[string]string{"itemID": "1$2!3"}
			if spaceRoot {
				handler, method, ids = api.SetSpaceRootLinkPassword, "SetPublicLinkPasswordOnSpaceRoot", map[string]string{"driveID": "1$2"}
			} else {
				rCTX.URLParams.Add("itemID", "1$2!3")
			}
			mockProvider.On(method, mock.Anything, mock.Anything, "permissionid", "s3cr3t-P4ss!").
				Return(libregraph.Permission{}, serviceErr).Once()

			responseRecorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"password":"s3cr3t-P4ss!"}`)).
				WithContext(context.WithValue(context.WithValue(context.Background(), chi.RouteCtxKey, rCTX), middleware.RequestIDKey, "request-42"))
			request.Header.Set("Authorization", "Bearer secret-token-xyz")
			handler(responseRecorder, request)

			// the HTTP response is the rendered service error, as before
			Expect(responseRecorder.Code).To(Equal(expectedStatus))
			graphErr, ok := errorcode.ToError(serviceErr)
			Expect(ok).To(BeTrue())
			jsonData := gjson.Get(responseRecorder.Body.String(), "error")
			Expect(jsonData.Get("code").String()).To(Equal(graphErr.GetCode().String()))
			Expect(jsonData.Get("message").String()).To(Equal(strings.TrimPrefix(serviceErr.Error(), graphErr.GetCode().String()+": ")))
			// the service operation was attempted with the parsed ids and password
			mockProvider.AssertExpectations(GinkgoT())

			entries := logEntries(&logBuf)
			var failureEntries []map[string]any
			for _, entry := range entries {
				matches := entry["level"] == level && entry["permissionID"] == "permissionid"
				for k, v := range ids {
					matches = matches && entry[k] == v
				}
				if matches {
					failureEntries = append(failureEntries, entry)
				}
			}
			Expect(failureEntries).To(HaveLen(1))
			// the failure is logged with its error code and error text, whatever its level
			Expect(failureEntries[0]).To(SatisfyAll(
				HaveKeyWithValue("request-id", "request-42"),
				HaveKeyWithValue("errorCode", graphErr.GetCode().String()),
				HaveKeyWithValue("error", serviceErr.Error()),
			))
			if level == "error" {
				Expect(entriesAtLevel(entries, "error")).To(HaveLen(1))
			} else {
				Expect(entries).ToNot(ContainElement(HaveKeyWithValue("level", BeElementOf("error", "warn"))))
			}
			for _, payloadKey := range []string{"password", "Body", "body", "request", "response", "token", "Authorization"} {
				Expect(entries).ToNot(ContainElement(HaveKey(payloadKey)))
			}
			// secrets are never logged: neither the password nor the bearer token
			raw, err := json.Marshal(entries)
			Expect(err).ToNot(HaveOccurred())
			for _, secret := range []string{"s3cr3t-P4ss!", "secret-token-xyz"} {
				Expect(string(raw)).ToNot(ContainSubstring(secret))
			}
		},
		Entry("SetLinkPassword: internal error", false,
			errorcode.New(errorcode.GeneralException, "link manager unavailable"), http.StatusInternalServerError, "error"),
		Entry("SetLinkPassword: link not found", false,
			errorcode.New(errorcode.ItemNotFound, "link not found"), http.StatusNotFound, "debug"),
		Entry("SetSpaceRootLinkPassword: internal error", true,
			errorcode.New(errorcode.GeneralException, "link manager unavailable"), http.StatusInternalServerError, "error"),
		Entry("SetSpaceRootLinkPassword: link not found", true,
			errorcode.New(errorcode.ItemNotFound, "link not found"), http.StatusNotFound, "debug"),
	)

	checkDriveIDAndItemIDValidation := func(handler http.HandlerFunc) {
		rCTX.URLParams.Add("itemID", "3$4!5")

		responseRecorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/", nil).
			WithContext(
				context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
			)

		handler(responseRecorder, request)

		Expect(responseRecorder.Code).To(Equal(http.StatusUnprocessableEntity))

		jsonData := gjson.Get(responseRecorder.Body.String(), "error")
		Expect(jsonData.Get("message").String()).To(Equal("invalid driveID or itemID"))
	}

	Describe("Invite", func() {
		It("validates the driveID and itemID url param", func() {
			checkDriveIDAndItemIDValidation(httpAPI.Invite)
		})

		It("return an error when the Invite provider errors", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)

			onInvite := mockProvider.On("Invite", mock.Anything, mock.Anything, mock.Anything)

			onInvite.Return(func(ctx context.Context, resourceID *provider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error) {
				return libregraph.Permission{}, errors.New("any")
			}).Once()

			httpAPI.Invite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		})

		It("call the Invite provider with the correct arguments", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			onInvite := mockProvider.On("Invite", mock.Anything, mock.Anything, mock.Anything)
			onInvite.Return(func(ctx context.Context, resourceID *provider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error) {
				Expect(storagespace.FormatResourceID(resourceID)).To(Equal("1$2!3"))
				return libregraph.Permission{}, nil
			}).Once()

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.Invite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		})

		It("fails with unknown or disable role", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			responseRecorder := httptest.NewRecorder()
			invite.Roles = []string{unifiedrole.UnifiedRoleViewerID, unifiedrole.UnifiedRoleSecureViewerID}
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.Invite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
		})
	})
	Describe("SpaceRootInvite", func() {
		It("call the Invite provider with the correct arguments", func() {
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			onInvite := mockProvider.On("SpaceRootInvite", mock.Anything, mock.Anything, mock.Anything)
			onInvite.Return(func(ctx context.Context, driveID *provider.ResourceId, invite libregraph.DriveItemInvite) (libregraph.Permission, error) {
				Expect(storagespace.FormatResourceID(driveID)).To(Equal("1$2"))
				return libregraph.Permission{}, nil
			}).Once()

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.SpaceRootInvite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		})
		It("fails with an empty driveid", func() {
			rCTX.URLParams.Add("driveID", "")
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.SpaceRootInvite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusUnprocessableEntity))
		})

		It("fails with unknown or disable role", func() {
			responseRecorder := httptest.NewRecorder()
			invite.Roles = []string{unifiedrole.UnifiedRoleViewerID, unifiedrole.UnifiedRoleSecureViewerID}
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.SpaceRootInvite(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
		})
	})
	Describe("ListPermissions", func() {
		It("calls the ListPermissions provider with the correct arguments", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			mockProvider.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(func(ctx context.Context, itemid *provider.ResourceId, opt svc.ListPermissionsQueryOptions) (libregraph.CollectionOfPermissionsWithAllowedValues, error) {
					Expect(storagespace.FormatResourceID(itemid)).To(Equal("1$2!3"))
					return libregraph.CollectionOfPermissionsWithAllowedValues{}, nil
				}).Once()

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.ListPermissions(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		})
		It("fails with an empty itemid", func() {
			responseRecorder := httptest.NewRecorder()
			inviteJson, err := json.Marshal(invite)
			Expect(err).ToNot(HaveOccurred())

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(inviteJson)).
				WithContext(
					context.WithValue(context.Background(), chi.RouteCtxKey, rCTX),
				)
			httpAPI.ListPermissions(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("GetPermission", func() {
		get := func() *httptest.ResponseRecorder {
			responseRecorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil).
				WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCTX))
			httpAPI.GetPermission(responseRecorder, request)
			return responseRecorder
		}

		It("passes the item and permission ID to the provider and renders the permission", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			rCTX.URLParams.Add("permissionID", "perm%2Fid")
			mockProvider.On("GetPermission", mock.Anything, mock.Anything, mock.Anything).
				Return(func(ctx context.Context, itemID *provider.ResourceId, permissionID string) (libregraph.Permission, error) {
					Expect(storagespace.FormatResourceID(itemID)).To(Equal("1$2!3"))
					Expect(permissionID).To(Equal("perm/id"))
					return libregraph.Permission{Id: libregraph.PtrString("perm/id")}, nil
				}).Once()

			rr := get()
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(gjson.Get(rr.Body.String(), "id").String()).To(Equal("perm/id"))
		})

		It("rejects an itemID that does not belong to the drive", func() {
			rCTX.URLParams.Add("itemID", "3$4!5")
			rCTX.URLParams.Add("permissionID", "permid")
			Expect(get().Code).To(Equal(http.StatusNotFound))
		})

		It("rejects an empty permissionID", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			Expect(get().Code).To(Equal(http.StatusBadRequest))
		})

		It("renders a not found error of the provider", func() {
			rCTX.URLParams.Add("itemID", "1$2!3")
			rCTX.URLParams.Add("permissionID", "permid")
			mockProvider.On("GetPermission", mock.Anything, mock.Anything, mock.Anything).
				Return(libregraph.Permission{}, errorcode.New(errorcode.ItemNotFound, "permission not found")).Once()

			Expect(get().Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("GetSpaceRootPermission", func() {
		It("rejects an invalid driveID", func() {
			rCTX = chi.NewRouteContext()
			rCTX.URLParams.Add("permissionID", "permid")
			responseRecorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil).
				WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCTX))
			httpAPI.GetSpaceRootPermission(responseRecorder, request)

			Expect(responseRecorder.Code).To(Equal(http.StatusUnprocessableEntity))
		})
	})
})

var _ = Describe("DriveItemPermissions routes", func() {
	var (
		graphService  svc.Service
		gatewayClient *cs3mocks.GatewayAPIClient
		ctx           context.Context
		rr            *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		ctx = revactx.ContextSetUser(context.Background(), &userpb.User{Id: &userpb.UserId{OpaqueId: "user"}})
		rr = httptest.NewRecorder()

		cfg := defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = ""
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector := pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		logger := log.NewLogger()
		m, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })

		var err error
		graphService, err = svc.NewService(
			svc.Config(cfg),
			svc.Metrics(m),
			svc.WithGatewaySelector(gatewaySelector),
			svc.EventsPublisher(&mocks.Publisher{}),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("GET .../items/{itemID}/permissions/{permissionID} returns the matching permission", func() {
		itemID := &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"}
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
			Status: status.NewOK(ctx),
			Info: &provider.ResourceInfo{
				Id:            itemID,
				Type:          provider.ResourceType_RESOURCE_TYPE_CONTAINER,
				PermissionSet: roleconversions.NewEditorRole().CS3ResourcePermissions(),
			},
		}, nil)
		gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(&collaboration.ListSharesResponse{Status: status.NewOK(ctx)}, nil)
		viewLink := func(id string) *link.PublicShare {
			return &link.PublicShare{
				Id:          &link.PublicShareId{OpaqueId: id},
				Token:       id,
				ResourceId:  itemID,
				Permissions: &link.PublicSharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
				Ctime:       &types.Timestamp{Seconds: uint64(time.Now().Unix())},
			}
		}
		gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(&link.ListPublicSharesResponse{
			Status: status.NewOK(ctx),
			Share:  []*link.PublicShare{viewLink("aaaaaaaaaaaaaaa"), viewLink("bbbbbbbbbbbbbbb")},
		}, nil)

		r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/drives/1$2/items/1$2!3/permissions/bbbbbbbbbbbbbbb", nil).WithContext(ctx)
		graphService.ServeHTTP(rr, r)

		Expect(rr.Code).To(Equal(http.StatusOK))
		Expect(gjson.Get(rr.Body.String(), "id").String()).To(Equal("bbbbbbbbbbbbbbb"))
		Expect(gjson.Get(rr.Body.String(), "link.type").String()).To(Equal("view"))
	})

	It("GET .../root/permissions/{permissionID} returns the matching permission of a project space", func() {
		rootID := &provider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}
		gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
			Status: status.NewOK(ctx),
			StorageSpaces: []*provider.StorageSpace{{
				Id:        &provider.StorageSpaceId{OpaqueId: "1$2"},
				SpaceType: "project",
				Root:      rootID,
			}},
		}, nil)
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
			Status: status.NewOK(ctx),
			Info: &provider.ResourceInfo{
				Id:            rootID,
				Type:          provider.ResourceType_RESOURCE_TYPE_CONTAINER,
				PermissionSet: roleconversions.NewManagerRole().CS3ResourcePermissions(),
			},
		}, nil)
		gatewayClient.On("ListShares", mock.Anything, mock.Anything).Return(&collaboration.ListSharesResponse{Status: status.NewOK(ctx)}, nil)
		gatewayClient.On("ListPublicShares", mock.Anything, mock.Anything).Return(&link.ListPublicSharesResponse{
			Status: status.NewOK(ctx),
			Share: []*link.PublicShare{{
				Id:          &link.PublicShareId{OpaqueId: "aaaaaaaaaaaaaaa"},
				Token:       "aaaaaaaaaaaaaaa",
				ResourceId:  rootID,
				Permissions: &link.PublicSharePermissions{Permissions: roleconversions.NewViewerRole().CS3ResourcePermissions()},
			}},
		}, nil)

		r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/drives/1$2/root/permissions/aaaaaaaaaaaaaaa", nil).WithContext(ctx)
		graphService.ServeHTTP(rr, r)

		Expect(rr.Code).To(Equal(http.StatusOK))
		Expect(gjson.Get(rr.Body.String(), "id").String()).To(Equal("aaaaaaaaaaaaaaa"))
	})
})

func guestGrantee(mail string) *provider.Grantee {
	return &provider.Grantee{
		Type: provider.GranteeType_GRANTEE_TYPE_USER,
		Id: &provider.Grantee_UserId{
			UserId: &userpb.UserId{Type: userpb.UserType_USER_TYPE_GUEST, OpaqueId: mail},
		},
	}
}
