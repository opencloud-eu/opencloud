package service

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/invitations"
)

// NewLogging returns a service that logs messages.
func NewLogging(next Service, logger log.Logger) Service {
	return logging{
		next:   next,
		logger: logger,
	}
}

type logging struct {
	next   Service
	logger log.Logger
}

func (l logging) List(ctx context.Context, userId string) ([]*invitations.Invitation, error) {
	l.logger.Debug().
		Interface("invitation", "list").
		Msg("List")

	return l.next.List(ctx, userId)
}

func (l logging) GetByInvitedEmail(ctx context.Context, email string) (*invitations.Invitation, error) {
	l.logger.Debug().
		Interface("invitation", email).
		Msg("Get")

	return l.next.GetByInvitedEmail(ctx, email)
}

// Invite implements the Service interface.
func (l logging) Invite(ctx context.Context, invitation *invitations.Invitation) (*invitations.Invitation, error) {
	l.logger.Debug().
		Interface("invitation", invitation).
		Msg("Invite")

	return l.next.Invite(ctx, invitation)
}
