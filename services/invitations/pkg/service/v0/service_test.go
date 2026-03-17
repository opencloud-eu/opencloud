package service_test

import (
	"context"

	"github.com/opencloud-eu/opencloud/services/invitations/pkg/invitations"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/service/v0"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service", func() {
	var (
		testSvc service.Service
		ctx     context.Context
	)
	BeforeEach(func() {
		ctx = context.Background()
		var err error
		testSvc, err = service.New()
		if err != nil {
			panic(err)
		}
	})

	Describe("Invite", func() {
		It("should return an invitation", func() {
			invite := &invitations.Invitation{
				InvitedUserEmailAddress: "test@example.com",
			}
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
