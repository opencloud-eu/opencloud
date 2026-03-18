package service_test

import (
	"context"

	"github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/config"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/invitations"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/service/v0"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service", func() {
	var (
		testSvc service.Service
		ctx     context.Context
		invite  *invitations.Invitation
	)
	BeforeEach(func() {
		ctx = context.Background()
		var err error
		cfg := &config.Config{
			UseBackEndForInvitations: false,
			Persistance: config.Persistance{
				Store: "memory",
			},
		}

		testSvc, err = service.New(
			service.Logger(log.NewLogger()),
			service.Config(cfg),
		)
		if err != nil {
			panic(err)
		}

		invite = &invitations.Invitation{
			InvitedUserEmailAddress: "test@example.com",
		}
	})

	Describe("Invite", func() {
		It("should return an error if no user is authorized", func() {
			inv, err := testSvc.Invite(ctx, invite)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(service.ErrUnauthorized))
			Expect(inv).To(BeNil())
		})
		It("should return an invitation", func() {
			ctx = revactx.ContextSetUser(ctx, &userv1beta1.User{
				Id: &userv1beta1.UserId{
					OpaqueId: "testuser",
				},
			})
			inv, err := testSvc.Invite(ctx, invite)
			Expect(err).ToNot(HaveOccurred())
			Expect(inv).ToNot(BeNil())
		})
		It("should return an error if the email is invalid", func() {
			invite := &invitations.Invitation{
				InvitedUserEmailAddress: "test",
			}
			inv, err := testSvc.Invite(ctx, invite)
			Expect(err).To(HaveOccurred())
			Expect(inv).To(BeNil())
		})
	})
})
