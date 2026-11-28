package svc_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	cs3mocks "github.com/opencloud-eu/reva/v3/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("AssignTags", func() {
	var (
		cfg           *config.Config
		gatewayClient *cs3mocks.GatewayAPIClient
		tape          *httptest.ResponseRecorder

		// the resource that gets the tags, with new random IDs for each test
		storageID, spaceID, itemID string
		resourceID                 string
	)

	// isResource matches a reference to the resource that gets the tags
	isResource := func(ref *provider.Reference) bool {
		id := ref.GetResourceId()
		return id.GetStorageId() == storageID && id.GetSpaceId() == spaceID && id.GetOpaqueId() == itemID
	}

	// assignTags sends a tag assignment for the resource, which already has the
	// given tags, to a graph service with the current configuration
	assignTags := func(existing string, tags ...string) {
		logger := log.NewLogger()
		metrics, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })
		gatewaySelector := mocks.NewSelectable[gateway.GatewayAPIClient](GinkgoT())
		gatewaySelector.EXPECT().Next().Return(gatewayClient, nil).Maybe()

		svc, err := service.NewService(
			service.Config(cfg),
			service.Metrics(metrics),
			service.WithIdentityBackend(&identitymocks.Backend{}),
			service.WithGatewaySelector(gatewaySelector),
		)
		Expect(err).ToNot(HaveOccurred())

		gatewayClient.EXPECT().Stat(mock.Anything, mock.MatchedBy(func(req *provider.StatRequest) bool {
			return isResource(req.GetRef())
		})).Return(&provider.StatResponse{
			Status: status.NewOK(nil),
			Info: &provider.ResourceInfo{
				Id:                &provider.ResourceId{StorageId: storageID, SpaceId: spaceID, OpaqueId: itemID},
				PermissionSet:     &provider.ResourcePermissions{InitiateFileUpload: true},
				ArbitraryMetadata: &provider.ArbitraryMetadata{Metadata: map[string]string{"tags": existing}},
			},
		}, nil)

		body, err := json.Marshal(libregraph.TagAssignment{ResourceId: resourceID, Tags: tags})
		Expect(err).ToNot(HaveOccurred())
		svc.AssignTags(tape, httptest.NewRequest(http.MethodPut, "/graph/v1.0/extensions/org.libregraph/tags", bytes.NewReader(body)))
	}

	// expectStoredTags expects the tags of the resource to be set to the given list
	expectStoredTags := func(tags string) {
		gatewayClient.EXPECT().SetArbitraryMetadata(mock.Anything, mock.MatchedBy(func(req *provider.SetArbitraryMetadataRequest) bool {
			return isResource(req.GetRef()) && req.GetArbitraryMetadata().GetMetadata()["tags"] == tags
		})).Return(&provider.SetArbitraryMetadataResponse{Status: status.NewOK(nil)}, nil)
	}

	BeforeEach(func() {
		storageID, spaceID, itemID = uuid.NewString(), uuid.NewString(), uuid.NewString()
		resourceID = storageID + "$" + spaceID + "!" + itemID

		gatewayClient = cs3mocks.NewGatewayAPIClient(GinkgoT())
		tape = httptest.NewRecorder()

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, these tests don't use LDAP
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}
		cfg.Validation.MaxTagLength = 5
	})

	It("limits tags to 100 characters by default", func() {
		Expect(defaults.FullDefaultConfig().Validation.MaxTagLength).To(Equal(100))
	})

	It("stores tags up to the maximum length", func() {
		expectStoredTags("abcde,new,old")
		assignTags("old", "abcde", "new")
		Expect(tape.Code).To(Equal(http.StatusOK))
	})

	It("rejects the whole assignment if a tag is too long, and stores nothing", func() {
		// no SetArbitraryMetadata expected: the mock fails the test if it is called
		assignTags("old", "ok", "abcdef")
		Expect(tape.Code).To(Equal(http.StatusBadRequest))
		Expect(tape.Body.String()).To(ContainSubstring(`\"abcdef\"`))
		Expect(tape.Body.String()).ToNot(ContainSubstring(`\"ok\"`))
	})

	It("counts characters, not bytes", func() {
		expectStoredTags("äöüßé")
		assignTags("", "äöüßé")
		Expect(tape.Code).To(Equal(http.StatusOK))
	})

	It("doesn't count the whitespace around a tag", func() {
		expectStoredTags("abcde")
		assignTags("", "  abcde  ")
		Expect(tape.Code).To(Equal(http.StatusOK))
	})

	It("doesn't limit the length if the maximum is 0", func() {
		long := strings.Repeat("x", 1000)
		cfg.Validation.MaxTagLength = 0
		expectStoredTags(long)
		assignTags("", long)
		Expect(tape.Code).To(Equal(http.StatusOK))
	})

	It("still rejects an assignment without new tags", func() {
		assignTags("abc", "abc")
		Expect(tape.Code).To(Equal(http.StatusBadRequest))
	})
})
