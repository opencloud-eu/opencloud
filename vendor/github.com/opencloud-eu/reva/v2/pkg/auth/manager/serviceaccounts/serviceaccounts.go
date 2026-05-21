package serviceaccounts

import (
	"context"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/rs/zerolog"

	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/pkg/errors"
)

type conf struct {
	ServiceUsers []serviceuser `mapstructure:"service_accounts"`
}

type serviceuser struct {
	ID     string `mapstructure:"id"`
	Secret string `mapstructure:"secret"`
}

type manager struct {
	authenticate func(userID, secret string) error
}

func init() {
	registry.Register("serviceaccounts", New)
}

// Configure parses the map conf
func (m *manager) Configure(config map[string]interface{}) error {
	c := &conf{}
	if err := mapstructure.Decode(config, c); err != nil {
		return errors.Wrap(err, "error decoding conf")
	}
	// only inmem authenticator for now
	a := &inmemAuthenticator{make(map[string]string)}
	for _, s := range c.ServiceUsers {
		a.m[s.ID] = s.Secret
	}
	m.authenticate = a.Authenticate
	return nil
}

// New creates a new manager for the 'service' authentication
func New(conf map[string]any, _ *zerolog.Logger) (auth.Manager, error) {
	m := &manager{}
	err := m.Configure(conf)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Authenticate authenticates the service account
func (m *manager) Authenticate(ctx context.Context, userID string, secret string) (*userpb.User, map[string]*authpb.Scope, error) {
	if err := m.authenticate(userID, secret); err != nil {
		return nil, nil, err
	}
	scope, err := scope.AddOwnerScope(nil)
	if err != nil {
		return nil, nil, err
	}
	return &userpb.User{
		// TODO: more details for service users?
		Id: &userpb.UserId{
			OpaqueId: userID,
			Type:     userpb.UserType_USER_TYPE_SERVICE,
			Idp:      "none",
		},
	}, scope, nil
}

type inmemAuthenticator struct {
	m map[string]string
}

func (a *inmemAuthenticator) Authenticate(userID string, secret string) error {
	if secret == "" || a.m[userID] == "" {
		return errors.New("unknown user")
	}
	if a.m[userID] == secret {
		return nil
	}
	return errors.New("secrets do not match")
}
