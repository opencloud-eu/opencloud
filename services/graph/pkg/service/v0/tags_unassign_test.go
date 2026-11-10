package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("UnassignTags", func() {
	var (
		svc             service.Service
		gatewayClient   *cs3mocks.GatewayAPIClient
		eventsPublisher *mocks.Publisher
		tape            *httptest.ResponseRecorder

		// the resource that loses the tags, with new random IDs for each test
		storageID, spaceID, itemID string
		resourceID                 string
	)

	// isResource matches a reference to the resource that loses the tags
	isResource := func(ref *provider.Reference) bool {
		id := ref.GetResourceId()
		return id.GetStorageId() == storageID && id.GetSpaceId() == spaceID && id.GetOpaqueId() == itemID
	}

	// resourceHasTags lets the resource have the given tags when it is stat'ed
	resourceHasTags := func(existing string) {
		gatewayClient.EXPECT().Stat(mock.Anything, mock.MatchedBy(func(req *provider.StatRequest) bool {
			return isResource(req.GetRef())
		})).Return(&provider.StatResponse{
			Status: status.NewOK(nil),
			Info: &provider.ResourceInfo{
				Id:                &provider.ResourceId{StorageId: storageID, SpaceId: spaceID, OpaqueId: itemID},
				Owner:             &userpb.UserId{OpaqueId: "owner"},
				PermissionSet:     &provider.ResourcePermissions{InitiateFileUpload: true},
				ArbitraryMetadata: &provider.ArbitraryMetadata{Metadata: map[string]string{"tags": existing}},
			},
		}, nil)
	}

	// expectStoredTags expects the tags of the resource to be set to the given list
	expectStoredTags := func(tags string) {
		gatewayClient.EXPECT().SetArbitraryMetadata(mock.Anything, mock.MatchedBy(func(req *provider.SetArbitraryMetadataRequest) bool {
			return isResource(req.GetRef()) && req.GetArbitraryMetadata().GetMetadata()["tags"] == tags
		})).Return(&provider.SetArbitraryMetadataResponse{Status: status.NewOK(nil)}, nil)
	}

	// expectEvent expects a TagsRemoved event for the resource with the given
	// tags, and lets publishing it return err
	expectEvent := func(tags string, err error) {
		eventsPublisher.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(ev events.TagsRemoved) bool {
			return ev.Tags == tags && isResource(ev.Ref) && ev.Executant.GetOpaqueId() == "executant"
		}), mock.Anything).Return(err)
	}

	// unassignTags asks to remove the given tags from the resource
	unassignTags := func(tags ...string) {
		body, err := json.Marshal(libregraph.TagUnassignment{ResourceId: resourceID, Tags: tags})
		Expect(err).ToNot(HaveOccurred())
		ctx := revactx.ContextSetUser(context.Background(), &userpb.User{Id: &userpb.UserId{OpaqueId: "executant"}})
		r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/extensions/org.libregraph/tags", bytes.NewReader(body)).WithContext(ctx)
		svc.UnassignTags(tape, r)
	}

	BeforeEach(func() {
		storageID, spaceID, itemID = uuid.NewString(), uuid.NewString(), uuid.NewString()
		resourceID = storageID + "$" + spaceID + "!" + itemID

		// both mocks fail the test on calls that weren't expected, and on
		// expected calls that didn't happen
		gatewayClient = cs3mocks.NewGatewayAPIClient(GinkgoT())
		eventsPublisher = mocks.NewPublisher(GinkgoT())
		tape = httptest.NewRecorder()

		cfg := defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, these tests don't use LDAP
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		logger := log.NewLogger()
		metrics, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })
		gatewaySelector := mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
		gatewaySelector.EXPECT().Next().Return(gatewayClient, nil).Maybe()

		var err error
		svc, err = service.NewService(
			service.Config(cfg),
			service.Metrics(metrics),
			service.WithIdentityBackend(&identitymocks.Backend{}),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(eventsPublisher),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("removes a tag the resource has", func() {
		resourceHasTags("a,b")
		expectStoredTags("b")
		expectEvent("a", nil)
		unassignTags("a")
		Expect(tape.Code).To(Equal(http.StatusOK))
	})

	It("succeeds for a tag the resource doesn't have, without changing it, and still sends the event", func() {
		resourceHasTags("a,b")
		// no SetArbitraryMetadata expected: the mock fails the test if it is called
		expectEvent("x", nil)
		unassignTags("x")
		Expect(tape.Code).To(Equal(http.StatusOK))
	})

	It("succeeds for a resource without any tags", func() {
		resourceHasTags("")
		expectEvent("x", nil)
		unassignTags("x")
		Expect(tape.Code).To(Equal(http.StatusOK))
	})

	It("removes the tags the resource has, and sends the event for all requested tags", func() {
		resourceHasTags("a,b,c")
		expectStoredTags("b")
		expectEvent("a,x,c", nil)
		unassignTags("a", "x", "c")
		Expect(tape.Code).To(Equal(http.StatusOK))
	})

	It("rejects a request without tags", func() {
		// no gateway call and no event expected
		unassignTags()
		Expect(tape.Code).To(Equal(http.StatusBadRequest))
	})

	It("sends no event if the tags can't be stored", func() {
		resourceHasTags("a,b")
		gatewayClient.EXPECT().SetArbitraryMetadata(mock.Anything, mock.Anything).
			Return(&provider.SetArbitraryMetadataResponse{Status: status.NewInternal(context.Background(), "failure")}, nil)
		unassignTags("a")
		Expect(tape.Code).To(Equal(http.StatusInternalServerError))
	})

	It("keeps the removal if the event can't be sent", func() {
		resourceHasTags("a,b")
		expectStoredTags("b")
		expectEvent("a", errors.New("no connection to the event bus"))
		unassignTags("a")
		Expect(tape.Code).To(Equal(http.StatusOK))
	})
})
