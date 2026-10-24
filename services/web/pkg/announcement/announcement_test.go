package announcement_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3permissions "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/nats-io/nats.go/jetstream"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/web/mocks"
	"github.com/opencloud-eu/opencloud/services/web/pkg/announcement"
)

func newGatewaySelector(allowed bool) pool.Selectable[gateway.GatewayAPIClient] {
	code := cs3rpc.Code_CODE_OK
	name := "announcement-test-allowed"
	if !allowed {
		code = cs3rpc.Code_CODE_PERMISSION_DENIED
		name = "announcement-test-denied"
	}

	client := &cs3mocks.GatewayAPIClient{}
	client.On("CheckPermission", mock.Anything, mock.Anything).Return(
		&cs3permissions.CheckPermissionResponse{Status: &cs3rpc.Status{Code: code}}, nil)

	// pool.GetSelector caches by name, so allow/deny must use distinct names
	return pool.GetSelector[gateway.GatewayAPIClient](
		name,
		"eu.opencloud.api.gateway",
		func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient { return client },
	)
}

func withUser(r *http.Request) *http.Request {
	return r.WithContext(revactx.ContextSetUser(r.Context(), &userpb.User{
		Id: &userpb.UserId{OpaqueId: "user"},
	}))
}

func newService(store *announcement.Store, allowed bool) announcement.Service {
	svc, err := announcement.NewService(announcement.ServiceOptions{}.
		WithLogger(log.NopLogger()).
		WithStore(store).
		WithGatewaySelector(newGatewaySelector(allowed)))
	Expect(err).ToNot(HaveOccurred())
	return svc
}

var _ = Describe("Store", func() {
	It("reads the stored announcement", func() {
		entry := mocks.NewKeyValueEntry(GinkgoT())
		entry.EXPECT().Value().Return([]byte(`{"enabled":true,"bannerText":"hello","infoText":"world"}`))
		kv := mocks.NewKeyValue(GinkgoT())
		kv.EXPECT().Get(mock.Anything, "announcement").Return(entry, nil)

		got, err := announcement.NewStore(kv).Get(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Enabled).To(BeTrue())
		Expect(got.BannerText).To(Equal("hello"))
		Expect(got.InfoText).To(Equal("world"))
	})

	It("returns the zero value when unset", func() {
		kv := mocks.NewKeyValue(GinkgoT())
		kv.EXPECT().Get(mock.Anything, "announcement").Return(nil, jetstream.ErrKeyNotFound)

		got, err := announcement.NewStore(kv).Get(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(got.BannerText).To(BeEmpty())
	})

	It("writes the announcement", func() {
		kv := mocks.NewKeyValue(GinkgoT())
		kv.EXPECT().Put(mock.Anything, "announcement", mock.Anything).Return(uint64(1), nil)

		Expect(announcement.NewStore(kv).Set(context.Background(), announcement.Announcement{BannerText: "hello"})).To(Succeed())
	})

	It("deletes the announcement", func() {
		kv := mocks.NewKeyValue(GinkgoT())
		kv.EXPECT().Delete(mock.Anything, "announcement").Return(nil)

		Expect(announcement.NewStore(kv).Delete(context.Background())).To(Succeed())
	})

	It("treats deleting a missing announcement as a no-op", func() {
		kv := mocks.NewKeyValue(GinkgoT())
		kv.EXPECT().Delete(mock.Anything, "announcement").Return(jetstream.ErrKeyNotFound)

		Expect(announcement.NewStore(kv).Delete(context.Background())).To(Succeed())
	})
})

var _ = Describe("Service", func() {
	Describe("NewService", func() {
		It("fails when options are missing", func() {
			_, err := announcement.NewService(announcement.ServiceOptions{})
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when options are valid", func() {
			_, err := announcement.NewService(announcement.ServiceOptions{}.
				WithStore(announcement.NewStore(mocks.NewKeyValue(GinkgoT()))).
				WithGatewaySelector(newGatewaySelector(true)))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Get", func() {
		It("returns the full stored announcement when permitted", func() {
			entry := mocks.NewKeyValueEntry(GinkgoT())
			entry.EXPECT().Value().Return([]byte(`{"enabled":true,"bannerText":"hello","infoText":"world"}`))
			kv := mocks.NewKeyValue(GinkgoT())
			kv.EXPECT().Get(mock.Anything, "announcement").Return(entry, nil)

			req := withUser(httptest.NewRequest(http.MethodGet, "/announcement", nil))
			resp := httptest.NewRecorder()

			newService(announcement.NewStore(kv), true).Get(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))
			var got announcement.Announcement
			Expect(json.Unmarshal(resp.Body.Bytes(), &got)).To(Succeed())
			Expect(got.Enabled).To(BeTrue())
			Expect(got.BannerText).To(Equal("hello"))
			Expect(got.InfoText).To(Equal("world"))
		})

		It("is forbidden without permission", func() {
			req := withUser(httptest.NewRequest(http.MethodGet, "/announcement", nil))
			resp := httptest.NewRecorder()

			newService(announcement.NewStore(mocks.NewKeyValue(GinkgoT())), false).Get(resp, req)

			Expect(resp.Code).To(Equal(http.StatusForbidden))
		})
	})

	Describe("Set", func() {
		It("persists the message when permitted", func() {
			kv := mocks.NewKeyValue(GinkgoT())
			kv.EXPECT().Put(mock.Anything, "announcement", mock.Anything).Return(uint64(1), nil)

			req := withUser(httptest.NewRequest(http.MethodPut, "/announcement", strings.NewReader(`{"bannerText":"hello"}`)))
			resp := httptest.NewRecorder()

			newService(announcement.NewStore(kv), true).Set(resp, req)

			Expect(resp.Code).To(Equal(http.StatusNoContent))
		})

		It("is forbidden without permission", func() {
			req := withUser(httptest.NewRequest(http.MethodPut, "/announcement", strings.NewReader(`{"bannerText":"hello"}`)))
			resp := httptest.NewRecorder()

			newService(announcement.NewStore(mocks.NewKeyValue(GinkgoT())), false).Set(resp, req)

			Expect(resp.Code).To(Equal(http.StatusForbidden))
		})

		It("rejects an invalid body", func() {
			req := withUser(httptest.NewRequest(http.MethodPut, "/announcement", strings.NewReader(`not json`)))
			resp := httptest.NewRecorder()

			newService(announcement.NewStore(mocks.NewKeyValue(GinkgoT())), true).Set(resp, req)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("rejects an oversized body", func() {
			body := `{"bannerText":"` + strings.Repeat("a", 300000) + `"}`
			req := withUser(httptest.NewRequest(http.MethodPut, "/announcement", strings.NewReader(body)))
			resp := httptest.NewRecorder()

			newService(announcement.NewStore(mocks.NewKeyValue(GinkgoT())), true).Set(resp, req)

			Expect(resp.Code).To(Equal(http.StatusRequestEntityTooLarge))
		})
	})

	Describe("Set with an empty banner text", func() {
		It("removes the stored announcement", func() {
			kv := mocks.NewKeyValue(GinkgoT())
			kv.EXPECT().Delete(mock.Anything, "announcement").Return(nil)

			req := withUser(httptest.NewRequest(http.MethodPut, "/announcement", strings.NewReader(`{"enabled":false,"bannerText":"","infoText":""}`)))
			resp := httptest.NewRecorder()
			newService(announcement.NewStore(kv), true).Set(resp, req)

			Expect(resp.Code).To(Equal(http.StatusNoContent))
		})
	})
})
