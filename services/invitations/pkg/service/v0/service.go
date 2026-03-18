package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/backends/keycloak"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/config"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/invitations"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"go-micro.dev/v4/store"
)

// Service defines the extension handlers.
type Service interface {
	// Invite creates a new invitation. Invitation adds an external user to the organization.
	//
	// When creating a new invitation you have several options available:
	// 1. On invitation creation, Microsoft Graph can automatically send an
	//    invitation email directly to the invited user, or your app can use
	//    the inviteRedeemUrl returned in the creation response to craft your
	//    own invitation (through your communication mechanism of choice) to
	//    the invited user. If you decide to have Microsoft Graph send an
	//    invitation email automatically, you can control the content and
	//    language of the email using invitedUserMessageInfo.
	// 2. When the user is invited, a user entity (of userType Guest) is
	//    created and can now be used to control access to resources. The
	//    invited user has to go through the redemption process to access any
	//    resources they have been invited to.
	Invite(ctx context.Context, invitation *invitations.Invitation) (*invitations.Invitation, error)
	List(ctx context.Context, userId string) ([]*invitations.Invitation, error)
	GetByInvitedEmail(ctx context.Context, email string) (*invitations.Invitation, error)
}

// Backend defines the behaviour of a user backend.
type Backend interface {
	// CreateUser creates a user in the backend and returns an identifier string.
	CreateUser(ctx context.Context, invitation *invitations.Invitation) (string, error)
	// CanSendMail should return true if the backend can send mail
	CanSendMail() bool
	// SendMail sends a mail to the user with details on how to reedeem the invitation.
	SendMail(ctx context.Context, identifier string) error
}

// New returns a new instance of Service
func New(opts ...Option) (Service, error) {
	options := newOptions(opts...)

	// Harcode keycloak backend for now, but this should be configurable in the future.
	backend := keycloak.New(
		options.Logger,
		options.Config.Keycloak.BasePath,
		options.Config.Keycloak.ClientID,
		options.Config.Keycloak.ClientSecret,
		options.Config.Keycloak.ClientRealm,
		options.Config.Keycloak.UserRealm,
		options.Config.Keycloak.InsecureSkipVerify,
	)

	s := svc{
		log:     options.Logger,
		config:  options.Config,
		backend: backend,
	}
	switch options.Config.Persistance.Store {
	case "nat-js-kv":
		panic("not implemented yet")
	case "memory":
		s.persistance = store.NewMemoryStore()
	default:
		panic("unknown store")
	}

	if err := s.persistance.Init(store.Table("invitations")); err != nil {
		s.log.Error().Err(err).Msg("error initializing store")
		return nil, err
	}
	return s, nil
}

type svc struct {
	config  *config.Config
	log     log.Logger
	backend Backend

	persistance store.Store
}

// Invite implements the service interface
func (s svc) Invite(ctx context.Context, invitation *invitations.Invitation) (*invitations.Invitation, error) {
	if invitation == nil {
		return nil, ErrBadRequest
	}

	if invitation.InvitedUserEmailAddress == "" {
		return nil, ErrMissingEmail
	}

	if s.config.UseBackEndForInvitations {
		id, err := s.backend.CreateUser(ctx, invitation)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrBackend, err)
		}

		// As we only have a single backend, and that backend supports email, we don't have
		// any code to handle mailing ourself yet.
		if s.backend.CanSendMail() {
			err := s.backend.SendMail(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrBackend, err)
			}
		}
	}

	// get logged in user
	u, ok := revactx.ContextGetUser(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}

	// serialize invitation
	invitationBytes, err := json.Marshal(invitation)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSerialization, err)
	}

	// persist invitation
	err = s.persistance.Write(&store.Record{
		Key: uuid.New().String(),
		// TODO: Fixme: persisting metadata does not help if you can not read it
		Metadata: map[string]interface{}{
			"invitedUserEmailAddress": invitation.InvitedUserEmailAddress,
			"inviterUserId":           u.GetId(),
		},
		Value: invitationBytes})

	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPersistence, err)
	}

	return invitation, nil
}

// List implements the service interface
func (s svc) List(ctx context.Context, userId string) ([]*invitations.Invitation, error) {
	fmt.Println("list invitations for user", userId)
	// get logged in user
	_, ok := revactx.ContextGetUser(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}

	invitationsKeys, err := s.persistance.List()

	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPersistence, err)
	}

	invSlice := []*invitations.Invitation{}

	for _, key := range invitationsKeys {
		// TODO: fixme: we can not read the metadata back again (WTF)
		invs, err := s.persistance.Read(key)
		if err != nil {
			// we cannot get the item, probably deleted in the meantime
			continue
		}
		for _, value := range invs {
			inv := &invitations.Invitation{}
			err := json.Unmarshal(value.Value, inv)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrSerialization, err)
			}
			invSlice = append(invSlice, inv)
		}
	}
	return invSlice, nil
}

// GetByInvitedEmail implements the service interface
func (s svc) GetByInvitedEmail(ctx context.Context, email string) (*invitations.Invitation, error) {
	// TODO: implement
	panic("implement me")
	return nil, nil
}
