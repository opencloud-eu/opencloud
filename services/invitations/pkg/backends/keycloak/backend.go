// Package keycloak offers an invitation backend for the invitation service.
package keycloak

import (
	"context"

	"github.com/google/uuid"
	"github.com/opencloud-eu/opencloud/pkg/keycloak"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/invitations"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

const (
	userType = "Guest"
)

var userRequiredActions = []keycloak.UserAction{
	keycloak.UserActionUpdatePassword,
	keycloak.UserActionVerifyEmail,
}

// Backend represents the keycloak backend.
type Backend struct {
	logger    log.Logger
	client    keycloak.Client
	userRealm string
}

// New instantiates a new keycloak.Backend with a default gocloak client.
func New(
	logger log.Logger,
	baseURL, clientID, clientSecret, clientRealm, userRealm string,
	insecureSkipVerify bool,
) *Backend {
	logger = log.Logger{
		Logger: logger.With().
			Str("invitationBackend", "keycloak").
			Str("clientID", clientID).
			Str("clientRealm", clientRealm).
			Str("userRealm", userRealm).
			Logger(),
	}
	client := keycloak.New(baseURL, clientID, clientSecret, clientRealm, insecureSkipVerify)
	return NewWithClient(logger, client, userRealm)
}

// NewWithClient creates a new backend with the supplied keycloak client.
func NewWithClient(
	logger log.Logger,
	client keycloak.Client,
	userRealm string,
) *Backend {
	return &Backend{
		logger:    logger,
		client:    client,
		userRealm: userRealm,
	}
}

// CreateUser creates a user in the keycloak backend.
func (b Backend) CreateUser(ctx context.Context, invitation *invitations.Invitation) (string, error) {
	u := uuid.New()

	b.logger.Info().
		Str("email", invitation.InvitedUserEmailAddress).
		Msg("Creating new user")
	user := &libregraph.User{
		Mail:                     &invitation.InvitedUserEmailAddress,
		AccountEnabled:           boolP(true),
		OnPremisesSamAccountName: invitation.InvitedUserEmailAddress,
		Id:                       stringP(u.String()),
		UserType:                 stringP(userType),
	}

	id, err := b.client.CreateUser(ctx, b.userRealm, user, userRequiredActions)
	if err != nil {
		b.logger.Error().
			Str("userID", u.String()).
			Str("email", invitation.InvitedUserEmailAddress).
			Err(err).
			Msg("Failed to create user")
		return "", err
	}

	return id, nil
}

// CanSendMail returns true because keycloak does allow sending mail.
func (b Backend) CanSendMail() bool { return true }

// SendMail sends a mail to the user with details on how to redeem the invitation.
func (b Backend) SendMail(ctx context.Context, id string) error {
	return b.client.SendActionsMail(ctx, b.userRealm, id, userRequiredActions)
}

func boolP(b bool) *bool       { return &b }
func stringP(s string) *string { return &s }
