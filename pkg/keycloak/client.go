// Package keycloak is a package for keycloak utility functions.
package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/Nerzal/gocloak/v13"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

// Some attribute constants.
// TODO: Make these configurable in the future.
const (
	_idAttr       = "OPENCLOUD_ID"
	_userTypeAttr = "OPENCLOUD_USER_TYPE"
)

// ConcreteClient represents a concrete implementation of a keycloak client
type ConcreteClient struct {
	keycloak     GoCloak
	clientID     string
	clientSecret string
	realm        string
	baseURL      string
}

// New instantiates a new keycloak.Backend with a default gocloak client.
func New(
	baseURL, clientID, clientSecret, realm string,
	insecureSkipVerify bool,
) *ConcreteClient {
	gc := gocloak.NewClient(baseURL)
	restyClient := gc.RestyClient()
	restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: insecureSkipVerify}) //nolint:gosec
	return NewWithClient(gc, baseURL, clientID, clientSecret, realm)
}

// NewWithClient instantiates a new keycloak.Backend with a custom
func NewWithClient(
	gocloakClient GoCloak,
	baseURL, clientID, clientSecret, realm string,
) *ConcreteClient {
	return &ConcreteClient{
		keycloak:     gocloakClient,
		baseURL:      baseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		realm:        realm,
	}
}

// CreateUser creates a user from a libregraph user and returns its *keycloak* ID.
// TODO: For now we only call this from the invitation service where all the attributes are set correctly.
//
//	For more wider use, do some sanity checking on the user instance.
func (c *ConcreteClient) CreateUser(ctx context.Context, realm string, user *libregraph.User, userActions []UserAction) (string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}

	req := gocloak.User{
		Email:     user.Mail,
		Enabled:   user.AccountEnabled,
		Username:  &user.OnPremisesSamAccountName,
		FirstName: user.GivenName,
		LastName:  user.Surname,
		Attributes: &map[string][]string{
			_idAttr:       {user.GetId()},
			_userTypeAttr: {user.GetUserType()},
		},
		RequiredActions: convertUserActions(userActions),
	}
	return c.keycloak.CreateUser(ctx, token.AccessToken, realm, req)
}

// SendActionsMail sends a mail to the user with userID instructing them to do the actions defined in userActions.
func (c *ConcreteClient) SendActionsMail(ctx context.Context, realm, userID string, userActions []UserAction) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}
	params := gocloak.ExecuteActionsEmail{
		UserID:  &userID,
		Actions: convertUserActions(userActions),
	}

	return c.keycloak.ExecuteActionsEmail(ctx, token.AccessToken, realm, params)
}

// getUserByParams looks up a user by the given parameters.
func (c *ConcreteClient) getUserByParams(ctx context.Context, realm string, params gocloak.GetUsersParams) (*libregraph.User, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	users, err := c.keycloak.GetUsers(ctx, token.AccessToken, realm, params)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("no users found")
	}

	if len(users) > 1 {
		return nil, fmt.Errorf("%d users found", len(users))
	}

	return c.keycloakUserToLibregraph(users[0]), nil
}

// GetUserByUsername looks up a user by username.
func (c *ConcreteClient) GetUserByUsername(ctx context.Context, realm, username string) (*libregraph.User, error) {
	return c.getUserByParams(ctx, realm, gocloak.GetUsersParams{
		Username: &username,
	})
}

// GetPIIReport returns a structure with all the PII for the user.
func (c *ConcreteClient) GetPIIReport(ctx context.Context, realm, username string) (*PIIReport, error) {
	u, err := c.GetUserByUsername(ctx, realm, username)
	if err != nil {
		return nil, err
	}

	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	keycloakID, err := c.getKeyCloakID(u)
	if err != nil {
		return nil, err
	}

	sessions, err := c.keycloak.GetUserSessions(ctx, token.AccessToken, realm, keycloakID)
	if err != nil {
		return nil, err
	}

	return &PIIReport{
		UserData: u,
		Sessions: sessions,
	}, nil
}

// getToken gets a fresh token for the request.
// TODO: set a token on the struct and check if it's still valid before requesting a new one.
func (c *ConcreteClient) getToken(ctx context.Context) (*gocloak.JWT, error) {
	token, err := c.keycloak.LoginClient(ctx, c.clientID, c.clientSecret, c.realm)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	rRes, err := c.keycloak.RetrospectToken(ctx, token.AccessToken, c.clientID, c.clientSecret, c.realm)
	if err != nil {
		return nil, fmt.Errorf("failed to retrospect token: %w", err)
	}

	if !*rRes.Active {
		return nil, fmt.Errorf("token is not active")
	}

	return token, nil
}

func (c *ConcreteClient) keycloakUserToLibregraph(u *gocloak.User) *libregraph.User {
	var ldapID string
	var userType *string

	if u.Attributes != nil {
		attrs := *u.Attributes
		ldapIDs, ok := attrs[_idAttr]
		if ok {
			ldapID = ldapIDs[0]
		}

		userTypes, ok := attrs[_userTypeAttr]
		if ok {
			userType = &userTypes[0]
		}
	}

	return &libregraph.User{
		Id:             &ldapID,
		Mail:           u.Email,
		GivenName:      u.FirstName,
		Surname:        u.LastName,
		AccountEnabled: u.Enabled,
		UserType:       userType,
		Identities: []libregraph.ObjectIdentity{
			{
				Issuer:           &c.baseURL,
				IssuerAssignedId: u.ID,
			},
		},
	}
}

func (c *ConcreteClient) getKeyCloakID(u *libregraph.User) (string, error) {
	for _, i := range u.Identities {
		if *i.Issuer == c.baseURL {
			return *i.IssuerAssignedId, nil
		}
	}
	return "", fmt.Errorf("could not find identity for issuer: %s", c.baseURL)
}

func convertUserActions(userActions []UserAction) *[]string {
	stringActions := make([]string, len(userActions))
	for i, a := range userActions {
		stringActions[i] = userActionsToString[a]
	}
	return &stringActions
}
