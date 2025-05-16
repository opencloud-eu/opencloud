package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	utils "github.com/opencloud-eu/reva/v2/pkg/utils"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"go-micro.dev/v4/selector"
)

type cs3backend struct {
	graphSelector selector.Selector
	Options
}

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	logger              log.Logger
	gatewaySelector     pool.Selectable[gateway.GatewayAPIClient]
	selector            selector.Selector
	machineAuthAPIKey   string
	oidcISS             string
	serviceAccount      config.ServiceAccount
	autoProvisionClaims config.AutoProvisionClaims
}

var (
	errGroupNotFound = errors.New("group not found")
)

// WithLogger sets the logger option
func WithLogger(l log.Logger) Option {
	return func(o *Options) {
		o.logger = l
	}
}

// WithRevaGatewaySelector set the gatewaySelector option
func WithRevaGatewaySelector(selectable pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.gatewaySelector = selectable
	}
}

// WithSelector set the Selector option
func WithSelector(selector selector.Selector) Option {
	return func(o *Options) {
		o.selector = selector
	}
}

// WithMachineAuthAPIKey configures the machine auth API key
func WithMachineAuthAPIKey(ma string) Option {
	return func(o *Options) {
		o.machineAuthAPIKey = ma
	}
}

// WithOIDCissuer set the OIDC issuer URL
func WithOIDCissuer(oidcISS string) Option {
	return func(o *Options) {
		o.oidcISS = oidcISS
	}
}

// WithServiceAccount configures the service account creator to use
func WithServiceAccount(c config.ServiceAccount) Option {
	return func(o *Options) {
		o.serviceAccount = c
	}
}

func WithAutoProvisionClaims(claims config.AutoProvisionClaims) Option {
	return func(o *Options) {
		o.autoProvisionClaims = claims
	}
}

// NewCS3UserBackend creates a user-provider which fetches users from a CS3 UserBackend
func NewCS3UserBackend(opts ...Option) UserBackend {
	opt := Options{}
	for _, o := range opts {
		o(&opt)
	}

	b := cs3backend{
		Options:       opt,
		graphSelector: opt.selector,
	}

	return &b
}

func (c *cs3backend) GetUserByClaims(ctx context.Context, claim, value string) (*cs3.User, string, error) {
	gatewayClient, err := c.gatewaySelector.Next()
	if err != nil {
		return nil, "", fmt.Errorf("could not obtain gatewayClient: %s", err)
	}

	res, err := gatewayClient.Authenticate(ctx, &gateway.AuthenticateRequest{
		Type:         "machine",
		ClientId:     claim + ":" + value,
		ClientSecret: c.machineAuthAPIKey,
	})

	switch {
	case err != nil:
		return nil, "", fmt.Errorf("could not get user by claim %v with value %v: %w", claim, value, err)
	case res.Status.Code != rpcv1beta1.Code_CODE_OK:
		if res.Status.Code == rpcv1beta1.Code_CODE_NOT_FOUND {
			return nil, "", ErrAccountNotFound
		}
		return nil, "", fmt.Errorf("could not get user by claim %v with value %v : %s ", claim, value, res.GetStatus().GetMessage())
	}

	user := res.User

	return user, res.GetToken(), nil
}

func (c *cs3backend) Authenticate(ctx context.Context, username string, password string) (*cs3.User, string, error) {
	gatewayClient, err := c.gatewaySelector.Next()
	if err != nil {
		return nil, "", fmt.Errorf("could not obtain gatewayClient: %s", err)
	}

	res, err := gatewayClient.Authenticate(ctx, &gateway.AuthenticateRequest{
		Type:         "basic",
		ClientId:     username,
		ClientSecret: password,
	})

	switch {
	case err != nil:
		return nil, "", fmt.Errorf("could not authenticate with username and password user: %s, %w", username, err)
	case res.Status.Code != rpcv1beta1.Code_CODE_OK:
		return nil, "", fmt.Errorf("could not authenticate with username and password user: %s, got code: %d", username, res.GetStatus().GetCode())
	}

	return res.User, res.Token, nil
}

// CreateUserFromClaims creates a new user via libregraph users API, taking the
// attributes from the provided `claims` map. On success it returns the new
// user. If the user already exist this is not considered an error and the
// function will just return the existing user.
func (c *cs3backend) CreateUserFromClaims(ctx context.Context, claims map[string]interface{}) (*cs3.User, error) {
	gatewayClient, err := c.gatewaySelector.Next()
	if err != nil {
		c.logger.Error().Err(err).Msg("could not select next gateway client")
		return nil, err
	}
	newctx := context.Background()
	authRes, err := gatewayClient.Authenticate(newctx, &gateway.AuthenticateRequest{
		Type:         "serviceaccounts",
		ClientId:     c.serviceAccount.ServiceAccountID,
		ClientSecret: c.serviceAccount.ServiceAccountSecret,
	})
	if err != nil {
		return nil, err
	}
	if authRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		return nil, fmt.Errorf("error authenticating service user: %s", authRes.GetStatus().GetMessage())
	}

	lgClient, err := c.setupLibregraphClient(newctx, authRes.GetToken())
	if err != nil {
		c.logger.Error().Err(err).Msg("Error setting up libregraph client.")
		return nil, err
	}

	newUser, err := c.libregraphUserFromClaims(claims)
	if err != nil {
		c.logger.Error().Err(err).Interface("claims", claims).Msg("Error creating user from claims")
		return nil, fmt.Errorf("error creating user from claims: %w", err)
	}

	req := lgClient.UsersApi.CreateUser(newctx).User(newUser)

	created, resp, err := req.Execute()
	defer resp.Body.Close()

	var reread bool
	if err != nil {
		if resp == nil {
			return nil, err
		}

		// If the user already exists here, some other request did already create it in parallel.
		// So just issue a Debug message and ignore the libregraph error otherwise
		var lerr error
		if reread, lerr = c.isAlreadyExists(resp); lerr != nil {
			c.logger.Error().Err(lerr).Msg("extracting error from ibregraph response body failed.")
			return nil, err
		}
		if !reread {
			c.logger.Error().Err(err).Msg("Error creating user")
			return nil, err
		}
	}

	// User has been created meanwhile, re-read it to get the user id
	if reread {
		c.logger.Debug().Msg("User already exist, re-reading via libregraph")
		gureq := lgClient.UserApi.GetUser(newctx, newUser.GetOnPremisesSamAccountName())
		created, resp, err = gureq.Execute()
		defer resp.Body.Close()
		if err != nil {
			c.logger.Error().Err(err).Msg("Error trying to re-read user from graphAPI")
			return nil, err
		}
	}

	cs3UserCreated := c.cs3UserFromLibregraph(newctx, created)

	return &cs3UserCreated, nil
}

func (c cs3backend) UpdateUserIfNeeded(ctx context.Context, user *cs3.User, claims map[string]interface{}) error {
	newUser, err := c.libregraphUserFromClaims(claims)
	if err != nil {
		c.logger.Error().Err(err).Interface("claims", claims).Msg("Error converting claims to user")
		return fmt.Errorf("error converting claims to updated user: %w", err)
	}

	// Check if the user needs to be updated, only updates of "displayName" and "mail" are supported
	// currently.
	userupdate := libregraph.NewUserUpdate()
	switch {
	case newUser.GetDisplayName() != user.GetDisplayName():
		userupdate.SetDisplayName(newUser.GetDisplayName())
	case newUser.GetMail() != user.GetMail():
		userupdate.SetMail(newUser.GetMail())
	}

	if userupdate.HasDisplayName() || userupdate.HasMail() {
		return c.updateLibregraphUser(user.GetId().GetOpaqueId(), *userupdate)
	}

	return nil
}

// SyncGroupMemberships maintains a users group memberships based on an OIDC claim
func (c cs3backend) SyncGroupMemberships(ctx context.Context, user *cs3.User, claims map[string]interface{}) error {
	gatewayClient, err := c.gatewaySelector.Next()
	if err != nil {
		c.logger.Error().Err(err).Msg("could not select next gateway client")
		return err
	}
	newctx := context.Background()
	token, err := utils.GetServiceUserToken(newctx, gatewayClient, c.serviceAccount.ServiceAccountID, c.serviceAccount.ServiceAccountSecret)
	if err != nil {
		c.logger.Error().Err(err).Msg("Error getting token for service user")
		return err
	}

	lgClient, err := c.setupLibregraphClient(newctx, token)
	if err != nil {
		c.logger.Error().Err(err).Msg("Error setting up libregraph client")
		return err
	}

	lgUser, resp, err := lgClient.UserApi.GetUser(newctx, user.GetId().GetOpaqueId()).Expand([]string{"memberOf"}).Execute()
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to lookup user via libregraph")
		return err
	}

	currentGroups := lgUser.GetMemberOf()
	currentGroupSet := make(map[string]struct{})
	for _, group := range currentGroups {
		currentGroupSet[group.GetDisplayName()] = struct{}{}
	}

	newGroupSet := make(map[string]struct{})
	if groups, ok := claims[c.autoProvisionClaims.Groups].([]interface{}); ok {
		for _, g := range groups {
			if group, ok := g.(string); ok {
				newGroupSet[group] = struct{}{}
			}
		}
	}

	for group := range newGroupSet {
		if _, exists := currentGroupSet[group]; !exists {
			c.logger.Debug().Str("group", group).Msg("adding user to group")
			// Check if group exists
			lgGroup, err := c.getLibregraphGroup(newctx, lgClient, group)
			switch {
			case errors.Is(err, errGroupNotFound):
				newGroup := libregraph.Group{}
				newGroup.SetDisplayName(group)
				req := lgClient.GroupsApi.CreateGroup(newctx).Group(newGroup)
				var resp *http.Response
				lgGroup, resp, err = req.Execute()
				if resp != nil {
					defer resp.Body.Close()
				}
				switch {
				case err == nil:
				// all good
				case resp == nil:
					return err
				default:
					// Ignore error if group already exists
					exists, lerr := c.isAlreadyExists(resp)
					switch {
					case lerr != nil:
						c.logger.Error().Err(lerr).Msg("extracting error from ibregraph response body failed.")
						return err
					case !exists:
						c.logger.Error().Err(err).Msg("Failed to create group via libregraph")
						return err
					default:
						// group has been created meanwhile, re-read it to get the group id
						lgGroup, err = c.getLibregraphGroup(newctx, lgClient, group)
						if err != nil {
							return err
						}
					}
				}
			case err != nil:
				return err
			}

			memberref := "https://localhost/graph/v1.0/users/" + user.GetId().GetOpaqueId()
			resp, err := lgClient.GroupApi.AddMember(newctx, lgGroup.GetId()).MemberReference(
				libregraph.MemberReference{
					OdataId: &memberref,
				},
			).Execute()
			if resp != nil {
				defer resp.Body.Close()
			}
			if err != nil {
				c.logger.Error().Err(err).Msg("Failed to add user to group via libregraph")
			}
		}
	}
	for current := range currentGroupSet {
		if _, exists := newGroupSet[current]; !exists {
			c.logger.Debug().Str("group", current).Msg("deleting user from group")
			lgGroup, err := c.getLibregraphGroup(newctx, lgClient, current)
			if err != nil {
				return err
			}
			resp, err := lgClient.GroupApi.DeleteMember(newctx, lgGroup.GetId(), user.GetId().GetOpaqueId()).Execute()
			if resp != nil {
				defer resp.Body.Close()
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c cs3backend) getLibregraphGroup(ctx context.Context, client *libregraph.APIClient, group string) (*libregraph.Group, error) {
	lgGroup, resp, err := client.GroupApi.GetGroup(ctx, group).Execute()
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		switch {
		case resp == nil:
			return nil, err
		case resp.StatusCode == http.StatusNotFound:
			return nil, errGroupNotFound
		case resp.StatusCode != http.StatusOK:
			return nil, err
		}
	}
	return lgGroup, nil
}

func (c cs3backend) updateLibregraphUser(userid string, user libregraph.UserUpdate) error {
	gatewayClient, err := c.gatewaySelector.Next()
	if err != nil {
		c.logger.Error().Err(err).Msg("could not select next gateway client")
		return err
	}
	newctx := context.Background()
	token, err := utils.GetServiceUserToken(newctx, gatewayClient, c.serviceAccount.ServiceAccountID, c.serviceAccount.ServiceAccountSecret)
	if err != nil {
		c.logger.Error().Err(err).Msg("Error getting token for service user")
		return err
	}

	lgClient, err := c.setupLibregraphClient(newctx, token)
	if err != nil {
		c.logger.Error().Err(err).Msg("Error setting up libregraph client")
		return err
	}

	req := lgClient.UserApi.UpdateUser(newctx, userid).UserUpdate(user)

	_, resp, err := req.Execute()
	defer resp.Body.Close()
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to update user via libregraph")
		return err
	}

	return nil
}

func (c cs3backend) setupLibregraphClient(_ context.Context, cs3token string) (*libregraph.APIClient, error) {
	// Use micro registry to resolve next graph service endpoint
	next, err := c.graphSelector.Select("eu.opencloud.web.graph")
	if err != nil {
		c.logger.Debug().Err(err).Msg("setupLibregraphClient: error during Select")
		return nil, err
	}
	node, err := next()
	if err != nil {
		c.logger.Debug().Err(err).Msg("setupLibregraphClient: error getting next Node")
		return nil, err
	}
	lgconf := libregraph.NewConfiguration()
	lgconf.Servers = libregraph.ServerConfigurations{
		{
			URL: fmt.Sprintf("%s://%s/graph", node.Metadata["protocol"], node.Address),
		},
	}

	lgconf.DefaultHeader = map[string]string{revactx.TokenHeader: cs3token}
	return libregraph.NewAPIClient(lgconf), nil
}

func (c cs3backend) isAlreadyExists(resp *http.Response) (bool, error) {
	oDataErr := libregraph.NewOdataErrorWithDefaults()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Debug().Err(err).Msg("Error trying to read libregraph response")
		return false, err
	}
	err = json.Unmarshal(body, oDataErr)
	if err != nil {
		c.logger.Debug().Err(err).Msg("Error unmarshalling libregraph response")
		return false, err
	}

	c.logger.Warn().Str("OData Error", oDataErr.Error.Message).Msg("Error Response")

	if oDataErr.Error.Code == errorcode.NameAlreadyExists.String() {
		return true, nil
	}
	return false, nil
}

func (c cs3backend) libregraphUserFromClaims(claims map[string]interface{}) (libregraph.User, error) {
	user := libregraph.User{}
	if dn, ok := claims[c.autoProvisionClaims.DisplayName].(string); ok {
		user.SetDisplayName(dn)
	} else {
		return user, fmt.Errorf("missing claim '%s' (displayName)", c.autoProvisionClaims.DisplayName)
	}
	if username, ok := claims[c.autoProvisionClaims.Username].(string); ok {
		user.SetOnPremisesSamAccountName(username)
	} else {
		return user, fmt.Errorf("missing claim '%s' (username)", c.autoProvisionClaims.Username)
	}
	// Email is optional so we don't need an 'else' here
	if mail, ok := claims[c.autoProvisionClaims.Email].(string); ok {
		user.SetMail(mail)
	}

	sub, subExists := claims[oidc.Sub].(string)
	iss, issExists := claims[oidc.Iss].(string)

	if subExists && issExists {
		var objectIdentity libregraph.ObjectIdentity
		objectIdentity.SetIssuer(iss)
		objectIdentity.SetIssuerAssignedId(sub)
		user.Identities = append(user.Identities, objectIdentity)
	}

	return user, nil
}

func (c cs3backend) cs3UserFromLibregraph(_ context.Context, lu *libregraph.User) cs3.User {
	cs3id := cs3.UserId{
		Type: cs3.UserType_USER_TYPE_PRIMARY,
		Idp:  c.oidcISS,
	}

	cs3id.OpaqueId = lu.GetId()

	cs3user := cs3.User{
		Id: &cs3id,
	}
	cs3user.Username = lu.GetOnPremisesSamAccountName()
	cs3user.DisplayName = lu.GetDisplayName()
	cs3user.Mail = lu.GetMail()
	return cs3user
}
