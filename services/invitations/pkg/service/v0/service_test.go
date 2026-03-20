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
		inviter *userv1beta1.User
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

		inviter = &userv1beta1.User{
			DisplayName: "testuser",
			Mail:        "testuser@example.org",
			Id: &userv1beta1.UserId{
				OpaqueId: "testuser",
				Type:     userv1beta1.UserType_USER_TYPE_PRIMARY,
			},
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
			ctx = revactx.ContextSetUser(ctx, inviter)
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

	Describe("List", func() {
		It("should return a list of invitations", func() {
			ctx = revactx.ContextSetUser(ctx, inviter)

			// create a slice of invitations
			invitations := []*invitations.Invitation{
				{
					InvitedUserEmailAddress: "test@example.org",
				},
				{
					InvitedUserEmailAddress: "test2@example.org",
				},
			}
			for _, inv := range invitations {
				_, err := testSvc.Invite(ctx, inv)
				Expect(err).ToNot(HaveOccurred())
			}

			inv, err := testSvc.List(ctx, inviter.GetId().GetOpaqueId())
			Expect(err).ToNot(HaveOccurred())
			Expect(inv).ToNot(BeNil())
			Expect(len(inv)).To(Equal(2))
		})
	})
	It("should return one invitation when two are set with different inviters", func() {
		ctx = revactx.ContextSetUser(ctx, inviter)
		_, err := testSvc.Invite(ctx, invite)
		Expect(err).ToNot(HaveOccurred())
		inviter.Id.OpaqueId = "testuser2"
		_, err = testSvc.Invite(ctx, invite)
		Expect(err).ToNot(HaveOccurred())
		invitations, err := testSvc.List(ctx, inviter.GetId().GetOpaqueId())
		Expect(err).ToNot(HaveOccurred())
		Expect(len(invitations)).To(Equal(1))
	})

	Describe("GetByInvitedEmail", func() {
		It("should return an invitation", func() {
			ctx = revactx.ContextSetUser(ctx, &userv1beta1.User{
				Id: &userv1beta1.UserId{
					OpaqueId: "testuser",
				},
			})
			inv, err := testSvc.GetByInvitedEmail(ctx, "test@example.org")
			Expect(err).ToNot(HaveOccurred())
			Expect(inv).ToNot(BeNil())
		})
	})
})
