package cache

import (
	"context"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3User "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

// mockGatewaySelector is a mock implementation of pool.Selectable[gateway.GatewayAPIClient]
type mockGatewaySelector struct {
	client gateway.GatewayAPIClient
}

func (m *mockGatewaySelector) Next(opts ...pool.Option) (gateway.GatewayAPIClient, error) {
	return m.client, nil
}

var _ = Describe("Cache", func() {
	var (
		ctx            context.Context
		idc            IdentityCache
		mockGwSelector pool.Selectable[gateway.GatewayAPIClient]
	)

	BeforeEach(func() {
		// Create a mock gateway selector (client can be nil for cached tests)
		mockGwSelector = &mockGatewaySelector{
			client: nil,
		}

		idc = NewIdentityCache(
			IdentityCacheWithGatewaySelector(mockGwSelector),
		)
		ctx = context.Background()
	})

	Describe("GetUser", func() {
		It("should return no error", func() {
			alan := &cs3User.User{
				Id: &cs3User.UserId{
					OpaqueId: "alan",
					TenantId: "",
				},
				DisplayName: "Alan",
			}
			// Persist the user to the cache for 1 hour
			idc.users.Set(alan.GetId().GetTenantId()+"|"+alan.GetId().GetOpaqueId(), alan, time.Hour)

			// getting the cache item in cache.go line 103 does not work
			ru, err := idc.GetUser(ctx, "", "alan")
			Expect(err).To(BeNil())
			Expect(ru).ToNot(BeNil())
			Expect(ru.GetId()).To(Equal(alan.GetId().GetOpaqueId()))
			Expect(ru.GetDisplayName()).To(Equal(alan.GetDisplayName()))
		})

		It("should return the correct user if two users with the same uid and different tennant ids exist", func() {
			alan1 := &cs3User.User{
				Id: &cs3User.UserId{
					OpaqueId: "alan",
					TenantId: "1234",
				},
				DisplayName: "Alan1",
			}

			alan2 := &cs3User.User{
				Id: &cs3User.UserId{
					OpaqueId: "alan",
					TenantId: "5678",
				},
				DisplayName: "Alan2",
			}
			// Persist the user to the cache for 1 hour
			idc.users.Set(alan1.GetId().GetTenantId()+"|"+alan1.GetId().GetOpaqueId(), alan1, time.Hour)
			idc.users.Set(alan2.GetId().GetTenantId()+"|"+alan2.GetId().GetOpaqueId(), alan2, time.Hour)
			ru, err := idc.GetUser(ctx, "5678", "alan")
			Expect(err).To(BeNil())
			Expect(ru.GetDisplayName()).To(Equal(alan2.GetDisplayName()))
			ru, err = idc.GetUser(ctx, "1234", "alan")
			Expect(err).To(BeNil())
			Expect(ru.GetDisplayName()).To(Equal(alan1.GetDisplayName()))
		})

		It("should not return an error, if the tenant id does match", func() {
			alan := &cs3User.User{
				Id: &cs3User.UserId{
					OpaqueId: "alan",
					TenantId: "1234",
				},
				DisplayName: "Alan",
			}
			// Persist the user to the cache for 1 hour
			cu := idc.users.Set(alan.GetId().GetTenantId()+"|"+alan.GetId().GetOpaqueId(), alan, time.Hour)
			// Test if element has been persisted in the cache
			Expect(cu.Value().GetId().GetOpaqueId()).To(Equal(alan.GetId().GetOpaqueId()))
			ru, err := idc.GetUser(ctx, "1234", "alan")
			Expect(err).To(BeNil())
			Expect(ru.GetDisplayName()).To(Equal(alan.GetDisplayName()))
		})
	})
})
