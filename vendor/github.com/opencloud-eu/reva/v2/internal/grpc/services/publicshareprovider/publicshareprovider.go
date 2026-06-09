// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package publicshareprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/password"
	"github.com/opencloud-eu/reva/v2/pkg/permission"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
)

const getUserCtxErrMsg = "error getting user from context"

func init() {
	rgrpc.Register("publicshareprovider", NewDefault)
}

type config struct {
	Driver                         string                            `mapstructure:"driver"`
	Drivers                        map[string]map[string]interface{} `mapstructure:"drivers"`
	GatewayAddr                    string                            `mapstructure:"gateway_addr"`
	AllowedPathsForShares          []string                          `mapstructure:"allowed_paths_for_shares"`
	EnableExpiredSharesCleanup     bool                              `mapstructure:"enable_expired_shares_cleanup"`
	WriteableShareMustHavePassword bool                              `mapstructure:"writeable_share_must_have_password"`
	PublicShareMustHavePassword    bool                              `mapstructure:"public_share_must_have_password"`
	PasswordPolicy                 map[string]interface{}            `mapstructure:"password_policy"`
}

type passwordPolicy struct {
	MinCharacters          int                 `mapstructure:"min_characters"`
	MinLowerCaseCharacters int                 `mapstructure:"min_lowercase_characters"`
	MinUpperCaseCharacters int                 `mapstructure:"min_uppercase_characters"`
	MinDigits              int                 `mapstructure:"min_digits"`
	MinSpecialCharacters   int                 `mapstructure:"min_special_characters"`
	BannedPasswordsList    map[string]struct{} `mapstructure:"banned_passwords_list"`
}

func (c *config) init() {
	if c.Driver == "" {
		c.Driver = "json"
	}
}

type service struct {
	conf                  *config
	sm                    publicshare.Manager
	gatewaySelector       pool.Selectable[gateway.GatewayAPIClient]
	allowedPathsForShares []*regexp.Regexp
	passwordValidator     password.Validator
}

func getShareManager(c *config) (publicshare.Manager, error) {
	if f, ok := registry.NewFuncs[c.Driver]; ok {
		return f(c.Drivers[c.Driver])
	}
	return nil, errtypes.NotFound("driver not found: " + c.Driver)
}

// TODO(labkode): add ctx to Close.
func (s *service) Close() error {
	return nil
}
func (s *service) UnprotectedEndpoints() []string {
	return []string{"/cs3.sharing.link.v1beta1.LinkAPI/GetPublicShareByToken"}
}

func (s *service) Register(ss *grpc.Server) {
	link.RegisterLinkAPIServer(ss, s)
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding config")
		return nil, err
	}
	return c, nil
}

func parsePasswordPolicy(m map[string]interface{}) (*passwordPolicy, error) {
	p := &passwordPolicy{}
	if err := mapstructure.Decode(m, p); err != nil {
		err = errors.Wrap(err, "error decoding password policy config")
		return nil, err
	}
	return p, nil
}

// New creates a new public share provider svc initialized from defaults
func NewDefault(m map[string]interface{}, ss *grpc.Server, _ *zerolog.Logger) (rgrpc.Service, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}
	p, err := parsePasswordPolicy(c.PasswordPolicy)
	if err != nil {
		return nil, err
	}

	c.init()

	sm, err := getShareManager(c)
	if err != nil {
		return nil, err
	}

	gatewaySelector, err := pool.GatewaySelector(sharedconf.GetGatewaySVC(c.GatewayAddr))
	if err != nil {
		return nil, err
	}
	return New(gatewaySelector, sm, c, p)
}

// New creates a new user share provider svc
func New(gatewaySelector pool.Selectable[gateway.GatewayAPIClient], sm publicshare.Manager, c *config, p *passwordPolicy) (rgrpc.Service, error) {
	allowedPathsForShares := make([]*regexp.Regexp, 0, len(c.AllowedPathsForShares))
	for _, s := range c.AllowedPathsForShares {
		regex, err := regexp.Compile(s)
		if err != nil {
			return nil, err
		}
		allowedPathsForShares = append(allowedPathsForShares, regex)
	}

	service := &service{
		conf:                  c,
		sm:                    sm,
		gatewaySelector:       gatewaySelector,
		allowedPathsForShares: allowedPathsForShares,
		passwordValidator:     newPasswordPolicy(p),
	}

	return service, nil
}

func newPasswordPolicy(c *passwordPolicy) password.Validator {
	if c == nil {
		return password.NewPasswordPolicy(0, 0, 0, 0, 0, nil)
	}
	return password.NewPasswordPolicy(
		c.MinCharacters,
		c.MinLowerCaseCharacters,
		c.MinUpperCaseCharacters,
		c.MinDigits,
		c.MinSpecialCharacters,
		c.BannedPasswordsList,
	)
}

func (s *service) isPathAllowed(path string) bool {
	if len(s.allowedPathsForShares) == 0 {
		return true
	}
	for _, reg := range s.allowedPathsForShares {
		if reg.MatchString(path) {
			return true
		}
	}
	return false
}

func (s *service) CreatePublicShare(ctx context.Context, req *link.CreatePublicShareRequest) (*link.CreatePublicShareResponse, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("publicshareprovider", "create").Msg("create public share")

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	isInternalLink := grants.PermissionsEqual(req.GetGrant().GetPermissions().GetPermissions(), &provider.ResourcePermissions{})

	sRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{ResourceId: req.GetResourceInfo().GetId()}})
	if err != nil {
		log.Err(err).Interface("resource_id", req.GetResourceInfo().GetId()).Msg("failed to stat resource to share")
		return &link.CreatePublicShareResponse{
			Status: status.NewInternal(ctx, "failed to stat resource to share"),
		}, err
	}

	// all users can create internal links
	if !isInternalLink {
		// check if the user has the permission in the user role
		ok, err := utils.CheckPermission(ctx, permission.WritePublicLink, gatewayClient)
		if err != nil {
			return &link.CreatePublicShareResponse{
				Status: status.NewInternal(ctx, "failed check user permission to write public link"),
			}, err
		}
		if !ok {
			return &link.CreatePublicShareResponse{
				Status: status.NewPermissionDenied(ctx, nil, "no permission to create public links"),
			}, nil
		}
	}

	// check that user has share permissions
	if !isInternalLink && !sRes.GetInfo().GetPermissionSet().AddGrant {
		return &link.CreatePublicShareResponse{
			Status: status.NewInvalidArg(ctx, "no share permission"),
		}, nil
	}

	// check if the user can share with the desired permissions. For internal links this is skipped,
	// users can always create internal links provided they have the AddGrant permission, which was already
	// checked above
	if !isInternalLink && !conversions.SufficientCS3Permissions(sRes.GetInfo().GetPermissionSet(), req.GetGrant().GetPermissions().GetPermissions()) {
		return &link.CreatePublicShareResponse{
			Status: status.NewInvalidArg(ctx, "insufficient permissions to create that kind of share"),
		}, nil
	}

	// validate path
	if !s.isPathAllowed(req.GetResourceInfo().GetPath()) {
		return &link.CreatePublicShareResponse{
			Status: status.NewFailedPrecondition(ctx, nil, "share creation is not allowed for the specified path"),
		}, nil
	}

	// check that this is a not a personal space root
	if req.GetResourceInfo().GetId().GetOpaqueId() == req.GetResourceInfo().GetId().GetSpaceId() &&
		req.GetResourceInfo().GetSpace().GetSpaceType() == "personal" {
		return &link.CreatePublicShareResponse{
			Status: status.NewInvalidArg(ctx, "cannot create link on personal space root"),
		}, nil
	}

	// quick link returns the existing one if already present
	quickLink, err := checkQuicklink(req.GetResourceInfo())
	if err != nil {
		return &link.CreatePublicShareResponse{
			Status: status.NewInvalidArg(ctx, "invalid quicklink value"),
		}, nil
	}
	if quickLink {
		f := []*link.ListPublicSharesRequest_Filter{publicshare.ResourceIDFilter(req.GetResourceInfo().GetId())}
		req := link.ListPublicSharesRequest{Filters: f}
		res, err := s.ListPublicShares(ctx, &req)
		if err != nil || res.GetStatus().GetCode() != rpc.Code_CODE_OK {
			return &link.CreatePublicShareResponse{
				Status: status.NewInternal(ctx, "could not list public links"),
			}, nil
		}
		for _, l := range res.GetShare() {
			if l.Quicklink {
				return &link.CreatePublicShareResponse{
					Status: status.NewOK(ctx),
					Share:  l,
				}, nil
			}
		}
	}

	grant := req.GetGrant()

	// validate expiration date
	if grant.GetExpiration() != nil {
		expirationDateTime := utils.TSToTime(grant.GetExpiration()).UTC()
		if expirationDateTime.Before(time.Now().UTC()) {
			msg := fmt.Sprintf("expiration date is in the past: %s", expirationDateTime.Format(time.RFC3339))
			return &link.CreatePublicShareResponse{
				Status: status.NewInvalidArg(ctx, msg),
			}, nil
		}
	}

	// enforce password if needed
	setPassword := grant.GetPassword()
	if !isInternalLink && enforcePassword(false, grant.GetPermissions().GetPermissions(), s.conf) && len(setPassword) == 0 {
		return &link.CreatePublicShareResponse{
			Status: status.NewInvalidArg(ctx, "password protection is enforced"),
		}, nil
	}

	// validate password policy
	if len(setPassword) > 0 {
		if err := s.passwordValidator.Validate(setPassword); err != nil {
			return &link.CreatePublicShareResponse{
				Status: status.NewInvalidArg(ctx, err.Error()),
			}, nil
		}
	}

	user := ctxpkg.ContextMustGetUser(ctx)
	res := &link.CreatePublicShareResponse{}
	share, err := s.sm.CreatePublicShare(ctx, user, req.GetResourceInfo(), req.GetGrant())
	switch {
	case err != nil:
		log.Error().Err(err).Interface("request", req).Msg("could not write public share")
		res.Status = status.NewInternal(ctx, "error persisting public share:"+err.Error())
	default:
		res.Status = status.NewOK(ctx)
		res.Share = share
		res.Opaque = utils.AppendPlainToOpaque(nil, "resourcename", sRes.GetInfo().GetName())
	}

	return res, nil
}

func (s *service) RemovePublicShare(ctx context.Context, req *link.RemovePublicShareRequest) (*link.RemovePublicShareResponse, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	log := appctx.GetLogger(ctx)
	log.Info().Str("publicshareprovider", "remove").Msg("remove public share")

	user := ctxpkg.ContextMustGetUser(ctx)
	ps, err := s.sm.GetPublicShare(ctx, user, req.GetRef(), false)
	if err != nil {
		return &link.RemovePublicShareResponse{
			Status: status.NewInternal(ctx, "error loading public share"),
		}, err
	}

	sRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{ResourceId: ps.ResourceId}})
	if err != nil {
		log.Err(err).Interface("resource_id", ps.ResourceId).Msg("failed to stat shared resource")
		return &link.RemovePublicShareResponse{
			Status: status.NewInternal(ctx, "failed to stat shared resource"),
		}, err
	}
	if !publicshare.IsCreatedByUser(ps, user) {
		if !sRes.GetInfo().GetPermissionSet().RemoveGrant {
			return &link.RemovePublicShareResponse{
				Status: status.NewPermissionDenied(ctx, nil, "no permission to delete public share"),
			}, err
		}
	}
	err = s.sm.RevokePublicShare(ctx, user, req.Ref)
	if err != nil {
		return &link.RemovePublicShareResponse{
			Status: status.NewInternal(ctx, "error deleting public share"),
		}, err
	}
	o := utils.AppendJSONToOpaque(nil, "resourceid", ps.GetResourceId())
	o = utils.AppendPlainToOpaque(o, "resourcename", sRes.GetInfo().GetName())
	return &link.RemovePublicShareResponse{
		Opaque: o,
		Status: status.NewOK(ctx),
	}, nil
}

func (s *service) GetPublicShareByToken(ctx context.Context, req *link.GetPublicShareByTokenRequest) (*link.GetPublicShareByTokenResponse, error) {
	log := appctx.GetLogger(ctx)
	log.Debug().Msg("getting public share by token")

	// there are 2 passes here, and the second request has no password
	found, err := s.sm.GetPublicShareByToken(ctx, req.GetToken(), req.GetAuthentication(), req.GetSign())
	switch v := err.(type) {
	case nil:
		return &link.GetPublicShareByTokenResponse{
			Status: status.NewOK(ctx),
			Share:  found,
		}, nil
	case errtypes.InvalidCredentials:
		return &link.GetPublicShareByTokenResponse{
			Status: status.NewPermissionDenied(ctx, v, "wrong password"),
		}, nil
	case errtypes.NotFound:
		return &link.GetPublicShareByTokenResponse{
			Status: status.NewNotFound(ctx, "unknown token"),
		}, nil
	default:
		return &link.GetPublicShareByTokenResponse{
			Status: status.NewInternal(ctx, "unexpected error"),
		}, nil
	}
}

func (s *service) GetPublicShare(ctx context.Context, req *link.GetPublicShareRequest) (*link.GetPublicShareResponse, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("publicshareprovider", "get").Msg("get public share")

	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		log.Error().Msg(getUserCtxErrMsg)
	}

	ps, err := s.sm.GetPublicShare(ctx, u, req.Ref, req.GetSign())
	switch {
	case err != nil:
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, err.Error())
		default:
			st = status.NewInternal(ctx, err.Error())
		}
		return &link.GetPublicShareResponse{
			Status: st,
		}, nil
	case ps == nil:
		return &link.GetPublicShareResponse{
			Status: status.NewNotFound(ctx, "not found"),
		}, nil
	default:
		return &link.GetPublicShareResponse{
			Status: status.NewOK(ctx),
			Share:  ps,
		}, nil
	}
}

func (s *service) ListPublicShares(ctx context.Context, req *link.ListPublicSharesRequest) (*link.ListPublicSharesResponse, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("publicshareprovider", "list").Msg("list public share")
	user, _ := ctxpkg.ContextGetUser(ctx)

	shares, err := s.sm.ListPublicShares(ctx, user, req.Filters, req.GetSign())
	if err != nil {
		log.Err(err).Msg("error listing shares")
		return &link.ListPublicSharesResponse{
			Status: status.NewInternal(ctx, "error listing public shares"),
		}, nil
	}

	res := &link.ListPublicSharesResponse{
		Status: status.NewOK(ctx),
		Share:  shares,
	}
	return res, nil
}

func (s *service) UpdatePublicShare(ctx context.Context, req *link.UpdatePublicShareRequest) (*link.UpdatePublicShareResponse, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("publicshareprovider", "update").Msg("update public share")

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	user := ctxpkg.ContextMustGetUser(ctx)
	ps, err := s.sm.GetPublicShare(ctx, user, req.GetRef(), false)
	if err != nil {
		return &link.UpdatePublicShareResponse{
			Status: status.NewInternal(ctx, "error loading public share"),
		}, err
	}

	isInternalLink := isInternalLink(req, ps)

	// check if the user has the permission in the user role
	if !publicshare.IsCreatedByUser(ps, user) {
		canWriteLink, err := utils.CheckPermission(ctx, permission.WritePublicLink, gatewayClient)
		if err != nil {
			return &link.UpdatePublicShareResponse{
				Status: status.NewInternal(ctx, "error checking permission to write public share"),
			}, err
		}
		if !canWriteLink {
			return &link.UpdatePublicShareResponse{
				Status: status.NewPermissionDenied(ctx, nil, "no permission to update public share"),
			}, nil
		}
	}

	sRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{ResourceId: ps.ResourceId}})
	if err != nil {
		log.Err(err).Interface("resource_id", ps.ResourceId).Msg("failed to stat shared resource")
		return &link.UpdatePublicShareResponse{
			Status: status.NewInternal(ctx, "failed to stat shared resource"),
		}, err
	}
	if sRes.Status.Code != rpc.Code_CODE_OK {
		return &link.UpdatePublicShareResponse{
			Status: sRes.GetStatus(),
		}, nil

	}

	if !isInternalLink && !publicshare.IsCreatedByUser(ps, user) {
		if !sRes.GetInfo().GetPermissionSet().UpdateGrant {
			return &link.UpdatePublicShareResponse{
				Status: status.NewPermissionDenied(ctx, nil, "no permission to update public share"),
			}, err
		}
	}

	// check if the user can change the permissions to the desired permissions
	updatePermissions := req.GetUpdate().GetType() == link.UpdatePublicShareRequest_Update_TYPE_PERMISSIONS
	if updatePermissions &&
		!isInternalLink &&
		!conversions.SufficientCS3Permissions(
			sRes.GetInfo().GetPermissionSet(),
			req.GetUpdate().GetGrant().GetPermissions().GetPermissions(),
		) {
		return &link.UpdatePublicShareResponse{
			Status: status.NewInvalidArg(ctx, "insufficient permissions to update that kind of share"),
		}, nil
	}
	if updatePermissions {
		beforePerm, _ := json.Marshal(sRes.GetInfo().GetPermissionSet())
		afterPerm, _ := json.Marshal(req.GetUpdate().GetGrant().GetPermissions())
		log.Info().
			Str("shares", "update").
			Msgf("updating permissions from %v to: %v",
				string(beforePerm),
				string(afterPerm),
			)
	}

	grant := req.GetUpdate().GetGrant()

	// validate expiration date
	if grant.GetExpiration() != nil {
		expirationDateTime := utils.TSToTime(grant.GetExpiration()).UTC()
		if expirationDateTime.Before(time.Now().UTC()) {
			msg := fmt.Sprintf("expiration date is in the past: %s", expirationDateTime.Format(time.RFC3339))
			return &link.UpdatePublicShareResponse{
				Status: status.NewInvalidArg(ctx, msg),
			}, nil
		}
	}

	// enforce password if needed
	var canOptOut bool
	if !isInternalLink {
		canOptOut, err = utils.CheckPermission(ctx, permission.DeleteReadOnlyPassword, gatewayClient)
		if err != nil {
			return &link.UpdatePublicShareResponse{
				Status: status.NewInternal(ctx, err.Error()),
			}, nil
		}
	}

	updatePassword := req.GetUpdate().GetType() == link.UpdatePublicShareRequest_Update_TYPE_PASSWORD
	setPassword := grant.GetPassword()

	// we update permissions with an empty password and password is not set on the public share
	emptyPasswordInPermissionUpdate := len(setPassword) == 0 && updatePermissions && !ps.PasswordProtected

	// password is updated, we use the current permissions to check if the user can opt out
	if updatePassword && !isInternalLink && enforcePassword(canOptOut, ps.GetPermissions().GetPermissions(), s.conf) && len(setPassword) == 0 {
		return &link.UpdatePublicShareResponse{
			Status: status.NewInvalidArg(ctx, "password protection is enforced"),
		}, nil
	}

	// permissions are updated, we use the new permissions to check if the user can opt out
	if emptyPasswordInPermissionUpdate && !isInternalLink && enforcePassword(canOptOut, grant.GetPermissions().GetPermissions(), s.conf) && len(setPassword) == 0 {
		return &link.UpdatePublicShareResponse{
			Status: status.NewInvalidArg(ctx, "password protection is enforced"),
		}, nil
	}

	// validate password policy
	if updatePassword && len(setPassword) > 0 {
		if err := s.passwordValidator.Validate(setPassword); err != nil {
			return &link.UpdatePublicShareResponse{
				Status: status.NewInvalidArg(ctx, err.Error()),
			}, nil
		}
	}

	updateR, err := s.sm.UpdatePublicShare(ctx, user, req)
	if err != nil {
		return &link.UpdatePublicShareResponse{
			Status: status.NewInternal(ctx, err.Error()),
		}, nil
	}

	res := &link.UpdatePublicShareResponse{
		Status: status.NewOK(ctx),
		Share:  updateR,
		Opaque: utils.AppendPlainToOpaque(nil, "resourcename", sRes.GetInfo().GetName()),
	}
	return res, nil
}

func isInternalLink(req *link.UpdatePublicShareRequest, ps *link.PublicShare) bool {
	switch {
	case req.GetUpdate().GetType() == link.UpdatePublicShareRequest_Update_TYPE_PERMISSIONS:
		return grants.PermissionsEqual(req.GetUpdate().GetGrant().GetPermissions().GetPermissions(), &provider.ResourcePermissions{})
	default:
		return grants.PermissionsEqual(ps.GetPermissions().GetPermissions(), &provider.ResourcePermissions{})
	}
}

func enforcePassword(canOptOut bool, permissions *provider.ResourcePermissions, conf *config) bool {
	isReadOnly := conversions.SufficientCS3Permissions(conversions.NewViewerRole().CS3ResourcePermissions(), permissions)
	if isReadOnly && canOptOut {
		return false
	}

	if conf.PublicShareMustHavePassword {
		return true
	}

	return !isReadOnly && conf.WriteableShareMustHavePassword
}

func checkQuicklink(info *provider.ResourceInfo) (bool, error) {
	if info == nil {
		return false, nil
	}
	if m := info.GetArbitraryMetadata().GetMetadata(); m != nil {
		q, ok := m["quicklink"]
		// empty string would trigger an error in ParseBool()
		if !ok || q == "" {
			return false, nil
		}
		quickLink, err := strconv.ParseBool(q)
		if err != nil {
			return false, err
		}
		return quickLink, nil
	}
	return false, nil
}
