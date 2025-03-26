package svc

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"

	cs3permissions "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/leonelquinteros/gotext"
	"github.com/opencloud-eu/opencloud/pkg/l10n"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/roles"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/config"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/settings"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
	metastore "github.com/opencloud-eu/opencloud/services/settings/pkg/store/metadata"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	merrors "go-micro.dev/v4/errors"
	"go-micro.dev/v4/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:embed l10n/locale
var _translationFS embed.FS

var _domain = "settings"

// Service represents a service.
type Service struct {
	id      string
	config  *config.Config
	logger  log.Logger
	manager settings.Manager
}

// NewService returns a service implementation for Service.
func NewService(cfg *config.Config, logger log.Logger) settings.ServiceHandler {
	service := Service{
		id:     "opencloud-settings",
		config: cfg,
		logger: logger,
	}

	service.manager = metastore.New(cfg)
	return service
}

// CheckPermission implements the CS3 API Permssions service.
// It's used to check if a subject (user or group) has a permission.
func (g Service) CheckPermission(ctx context.Context, req *cs3permissions.CheckPermissionRequest) (*cs3permissions.CheckPermissionResponse, error) {
	spec := req.GetSubjectRef().GetSpec()

	var accountID string
	switch ref := spec.(type) {
	case *cs3permissions.SubjectReference_UserId:
		accountID = ref.UserId.GetOpaqueId()
	case *cs3permissions.SubjectReference_GroupId:
		accountID = ref.GroupId.GetOpaqueId()
	}

	assignments, err := g.manager.ListRoleAssignments(accountID)
	if err != nil {
		return &cs3permissions.CheckPermissionResponse{
			Status: status.NewInternal(ctx, err.Error()),
		}, nil
	}

	roleIDs := make([]string, 0, len(assignments))
	for _, a := range assignments {
		roleIDs = append(roleIDs, a.GetRoleId())
	}

	permission, err := g.manager.ReadPermissionByName(req.GetPermission(), roleIDs)
	if err != nil {
		if !errors.Is(err, settings.ErrNotFound) {
			return &cs3permissions.CheckPermissionResponse{
				Status: status.NewInternal(ctx, err.Error()),
			}, nil
		}
	}

	if permission == nil {
		return &cs3permissions.CheckPermissionResponse{
			Status: &rpcv1beta1.Status{
				Code: rpcv1beta1.Code_CODE_PERMISSION_DENIED,
			},
		}, nil
	}

	return &cs3permissions.CheckPermissionResponse{
		Status: status.NewOK(ctx),
	}, nil
}

// TODO: check permissions on every request

// SaveBundle implements the BundleServiceHandler interface
func (g Service) SaveBundle(ctx context.Context, req *settingssvc.SaveBundleRequest, res *settingssvc.SaveBundleResponse) error {
	cleanUpResource(ctx, req.GetBundle().GetResource())
	if err := g.checkStaticPermissionsByBundleType(ctx, req.GetBundle().GetType()); err != nil {
		return err
	}
	if validationError := validateSaveBundle(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}

	r, err := g.manager.WriteBundle(req.GetBundle())
	if err != nil {
		return merrors.BadRequest(g.id, "%s", err)
	}
	res.Bundle = r
	return nil
}

// GetBundle implements the BundleServiceHandler interface
func (g Service) GetBundle(ctx context.Context, req *settingssvc.GetBundleRequest, res *settingssvc.GetBundleResponse) error {
	if validationError := validateGetBundle(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	bundle, err := g.manager.ReadBundle(req.GetBundleId())
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}
	filteredBundle := g.getFilteredBundle(g.getRoleIDs(ctx), bundle)
	if len(filteredBundle.GetSettings()) == 0 {
		err = fmt.Errorf("could not read bundle: %s", req.GetBundleId())
		return merrors.NotFound(g.id, "%s", err)
	}
	res.Bundle = filteredBundle
	return nil
}

// ListBundles implements the BundleServiceHandler interface
func (g Service) ListBundles(ctx context.Context, req *settingssvc.ListBundlesRequest, res *settingssvc.ListBundlesResponse) error {
	// fetch all bundles
	if validationError := validateListBundles(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	bundles, err := g.manager.ListBundles(settingsmsg.Bundle_TYPE_DEFAULT, req.GetBundleIds())
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}
	roleIDs := g.getRoleIDs(ctx)

	// find user locale
	var locale string
	if u, ok := ctxpkg.ContextGetUser(ctx); ok {
		var err error
		locale, err = g.getUserLocale(ctx, u.GetId().GetOpaqueId())
		if err != nil {
			g.logger.Error().Err(err).Str("userid", u.GetId().GetOpaqueId()).Msg("failed to get user locale")
		}
	}

	// filter settings in bundles that are allowed according to roles
	var filteredBundles []*settingsmsg.Bundle
	for _, bundle := range bundles {
		filteredBundle := g.getFilteredBundle(roleIDs, bundle)
		if len(filteredBundle.GetSettings()) > 0 {
			t := l10n.NewTranslatorFromCommonConfig(g.config.DefaultLanguage, _domain, g.config.TranslationPath, _translationFS, "l10n/locale").Locale(locale)
			filteredBundles = append(filteredBundles, translateBundle(filteredBundle, t))
		}
	}

	res.Bundles = filteredBundles
	return nil
}

func (g Service) getFilteredBundle(roleIDs []string, bundle *settingsmsg.Bundle) *settingsmsg.Bundle {
	// check if full bundle is whitelisted
	bundleResource := &settingsmsg.Resource{
		Type: settingsmsg.Resource_TYPE_BUNDLE,
		Id:   bundle.GetId(),
	}
	if g.hasPermission(
		roleIDs,
		bundleResource,
		[]settingsmsg.Permission_Operation{settingsmsg.Permission_OPERATION_READ, settingsmsg.Permission_OPERATION_READWRITE},
		settingsmsg.Permission_CONSTRAINT_OWN,
	) {
		return bundle
	}

	// filter settings based on permissions
	var filteredSettings []*settingsmsg.Setting
	for _, setting := range bundle.GetSettings() {
		settingResource := &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   setting.GetId(),
		}
		if g.hasPermission(
			roleIDs,
			settingResource,
			[]settingsmsg.Permission_Operation{settingsmsg.Permission_OPERATION_READ, settingsmsg.Permission_OPERATION_READWRITE},
			settingsmsg.Permission_CONSTRAINT_OWN,
		) {
			filteredSettings = append(filteredSettings, setting)
		}
	}
	bundle.Settings = filteredSettings
	return bundle
}

// AddSettingToBundle implements the BundleServiceHandler interface
func (g Service) AddSettingToBundle(ctx context.Context, req *settingssvc.AddSettingToBundleRequest, res *settingssvc.AddSettingToBundleResponse) error {
	cleanUpResource(ctx, req.GetSetting().GetResource())
	if err := g.checkStaticPermissionsByBundleID(ctx, req.GetBundleId()); err != nil {
		return err
	}
	if validationError := validateAddSettingToBundle(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}

	r, err := g.manager.AddSettingToBundle(req.GetBundleId(), req.GetSetting())
	if err != nil {
		return merrors.BadRequest(g.id, "%s", err)
	}
	res.Setting = r
	return nil
}

// RemoveSettingFromBundle implements the BundleServiceHandler interface
func (g Service) RemoveSettingFromBundle(ctx context.Context, req *settingssvc.RemoveSettingFromBundleRequest, _ *emptypb.Empty) error {
	if err := g.checkStaticPermissionsByBundleID(ctx, req.GetBundleId()); err != nil {
		return err
	}
	if validationError := validateRemoveSettingFromBundle(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}

	if err := g.manager.RemoveSettingFromBundle(req.GetBundleId(), req.GetSettingId()); err != nil {
		return merrors.BadRequest(g.id, "%s", err)
	}

	return nil
}

// SaveValue implements the ValueServiceHandler interface
func (g Service) SaveValue(ctx context.Context, req *settingssvc.SaveValueRequest, res *settingssvc.SaveValueResponse) error {
	req.Value.AccountUuid = getValidatedAccountUUID(ctx, req.GetValue().GetAccountUuid())
	if !g.isCurrentUser(ctx, req.GetValue().GetAccountUuid()) {
		return merrors.Forbidden(g.id, "can't save value for another user")
	}

	cleanUpResource(ctx, req.GetValue().GetResource())
	// TODO: we need to check, if the authenticated user has permission to write the value for the specified resource (e.g. global, file with id xy, ...)
	if validationError := validateSaveValue(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError.Error())
	}
	r, err := g.manager.WriteValue(req.GetValue())
	if err != nil {
		return merrors.BadRequest(g.id, "%s", err)
	}
	valueWithIdentifier, err := g.getValueWithIdentifier(r)
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}
	res.Value = valueWithIdentifier
	return nil
}

// GetValue implements the ValueServiceHandler interface
func (g Service) GetValue(_ context.Context, req *settingssvc.GetValueRequest, res *settingssvc.GetValueResponse) error {
	if validationError := validateGetValue(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	r, err := g.manager.ReadValue(req.GetId())
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}
	valueWithIdentifier, err := g.getValueWithIdentifier(r)
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}
	res.Value = valueWithIdentifier
	return nil
}

// GetValueByUniqueIdentifiers implements the ValueService interface
func (g Service) GetValueByUniqueIdentifiers(ctx context.Context, req *settingssvc.GetValueByUniqueIdentifiersRequest, res *settingssvc.GetValueResponse) error {
	req.AccountUuid = getValidatedAccountUUID(ctx, req.GetAccountUuid())
	if !g.isCurrentUser(ctx, req.GetAccountUuid()) {
		return merrors.Forbidden(g.id, "can't get value of another user")
	}
	if validationError := validateGetValueByUniqueIdentifiers(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	v, err := g.manager.ReadValueByUniqueIdentifiers(req.GetAccountUuid(), req.GetSettingId())
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}

	if v.GetBundleId() != "" {
		valueWithIdentifier, err := g.getValueWithIdentifier(v)
		if err != nil {
			return merrors.NotFound(g.id, "%s", err)
		}

		res.Value = valueWithIdentifier
	}
	return nil
}

// ListValues implements the ValueServiceHandler interface
func (g Service) ListValues(ctx context.Context, req *settingssvc.ListValuesRequest, res *settingssvc.ListValuesResponse) error {
	req.AccountUuid = getValidatedAccountUUID(ctx, req.GetAccountUuid())
	if !g.isCurrentUser(ctx, req.GetAccountUuid()) {
		return merrors.Forbidden(g.id, "can't list values of another user")
	}

	if validationError := validateListValues(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	values, err := g.manager.ListValues(req.GetBundleId(), req.GetAccountUuid())
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}
	result := make([]*settingsmsg.ValueWithIdentifier, 0, len(values))
	for _, value := range values {
		valueWithIdentifier, err := g.getValueWithIdentifier(value)
		if err == nil {
			result = append(result, valueWithIdentifier)
		}
	}
	res.Values = result
	return nil
}

// ListRoles implements the RoleServiceHandler interface
func (g Service) ListRoles(_ context.Context, req *settingssvc.ListBundlesRequest, res *settingssvc.ListBundlesResponse) error {
	//accountUUID := getValidatedAccountUUID(c, "me")
	if validationError := validateListRoles(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	r, err := g.manager.ListBundles(settingsmsg.Bundle_TYPE_ROLE, req.GetBundleIds())
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}
	// TODO: only allow listing roles when user has account/role/... management permissions
	res.Bundles = r
	return nil
}

// ListRoleAssignments implements the RoleServiceHandler interface
func (g Service) ListRoleAssignments(ctx context.Context, req *settingssvc.ListRoleAssignmentsRequest, res *settingssvc.ListRoleAssignmentsResponse) error {
	req.AccountUuid = getValidatedAccountUUID(ctx, req.GetAccountUuid())
	if validationError := validateListRoleAssignments(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	r, err := g.manager.ListRoleAssignments(req.GetAccountUuid())
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}
	res.Assignments = r
	return nil
}

// ListRoleAssignmentsFiltered implements the RoleServiceHandler interface. Who made this up? And why is everyone copying it? So this methods lists role assignments filtered by account or role.
func (g Service) ListRoleAssignmentsFiltered(ctx context.Context, req *settingssvc.ListRoleAssignmentsFilteredRequest, res *settingssvc.ListRoleAssignmentsResponse) error {
	if validationError := validateListRoleAssignmentsFiltered(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	filters := req.GetFilters()

	var r []*settingsmsg.UserRoleAssignment
	var err error
	switch filters[0].GetType() {
	case settingsmsg.UserRoleAssignmentFilter_TYPE_ACCOUNT:
		accountUUID := getValidatedAccountUUID(ctx, filters[0].GetAccountUuid())
		r, err = g.manager.ListRoleAssignments(accountUUID)
	case settingsmsg.UserRoleAssignmentFilter_TYPE_ROLE:
		roleID := filters[0].GetRoleId()
		r, err = g.manager.ListRoleAssignmentsByRole(roleID)
	}
	if err != nil {
		return merrors.NotFound(g.id, "%s", err)
	}
	res.Assignments = r
	return nil
}

// AssignRoleToUser implements the RoleServiceHandler interface
func (g Service) AssignRoleToUser(ctx context.Context, req *settingssvc.AssignRoleToUserRequest, res *settingssvc.AssignRoleToUserResponse) error {
	req.AccountUuid = getValidatedAccountUUID(ctx, req.GetAccountUuid())
	if validationError := validateAssignRoleToUser(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}

	ownAccountUUID, ok := metadata.Get(ctx, middleware.AccountID)
	if !ok {
		g.logger.Debug().Str("id", g.id).Msg("user not in context")
		return merrors.InternalServerError(g.id, "user not in context")
	}

	switch {
	case ownAccountUUID == req.GetAccountUuid():
		// Allow users to assign themself to the user or user light role
		// deny any other attempt to change	the user's own assignment
		if r, err := g.manager.ListRoleAssignments(req.GetAccountUuid()); err == nil && len(r) > 0 {
			return merrors.Forbidden(g.id, "Changing own role assignment forbidden")
		}
		if req.GetRoleId() != defaults.BundleUUIDRoleUser && req.GetRoleId() != defaults.BundleUUIDRoleUserLight {
			return merrors.Forbidden(g.id, "Changing own role assignment forbidden")
		}
		g.logger.Debug().Str("userid", ownAccountUUID).Msg("Self-assignment for default 'user' role permitted")
	case g.canManageRoles(ctx):
	default:
		return merrors.Forbidden(g.id, "user has no role management permission")
	}

	r, err := g.manager.WriteRoleAssignment(req.GetAccountUuid(), req.GetRoleId())
	if err != nil {
		return merrors.BadRequest(g.id, "%s", err)
	}
	res.Assignment = r
	return nil
}

// RemoveRoleFromUser implements the RoleServiceHandler interface
func (g Service) RemoveRoleFromUser(ctx context.Context, req *settingssvc.RemoveRoleFromUserRequest, _ *emptypb.Empty) error {
	if !g.canManageRoles(ctx) {
		return merrors.Forbidden(g.id, "user has no role management permission")
	}

	if validationError := validateRemoveRoleFromUser(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}

	ownAccountUUID, ok := metadata.Get(ctx, middleware.AccountID)
	if !ok {
		g.logger.Debug().Str("id", g.id).Msg("user not in context")
		return merrors.InternalServerError(g.id, "user not in context")
	}

	al, err := g.manager.ListRoleAssignments(ownAccountUUID)
	if err != nil {
		g.logger.Debug().Err(err).Str("id", g.id).Msg("ListRoleAssignments failed")
		return merrors.InternalServerError(g.id, "%s", err)
	}

	for _, a := range al {
		if a.GetId() == req.GetId() {
			g.logger.Debug().Str("id", g.id).Msg("Removing own role assignment forbidden")
			return merrors.Forbidden(g.id, "Removing own role assignment forbidden")
		}
	}

	if err := g.manager.RemoveRoleAssignment(req.GetId()); err != nil {
		return merrors.BadRequest(g.id, "%s", err)
	}
	return nil
}

// ListPermissions implements the PermissionServiceHandler interface
func (g Service) ListPermissions(ctx context.Context, req *settingssvc.ListPermissionsRequest, res *settingssvc.ListPermissionsResponse) error {
	ownAccountUUID, ok := metadata.Get(ctx, middleware.AccountID)
	if !ok {
		g.logger.Debug().Str("id", g.id).Msg("user not in context")
		return merrors.InternalServerError(g.id, "user not in context")
	}

	if ownAccountUUID != req.GetAccountUuid() {
		return merrors.NotFound(g.id, "user not found: %s", req.GetAccountUuid())
	}

	assignments, err := g.manager.ListRoleAssignments(req.GetAccountUuid())
	if err != nil {
		return err
	}

	// deduplicate role ids
	roleIDs := map[string]struct{}{}
	for _, a := range assignments {
		roleIDs[a.GetRoleId()] = struct{}{}
	}

	// deduplicate permission names
	permissionNames := map[string]struct{}{}
	for roleID := range roleIDs {
		bundle, err := g.manager.ReadBundle(roleID)
		if err != nil {
			if !errors.Is(err, settings.ErrNotFound) {
				return err
			}
			continue
		}

		if bundle != nil {
			for _, setting := range bundle.GetSettings() {
				permissionNames[formatPermissionName(setting)] = struct{}{}
			}
		}
	}

	res.Permissions = make([]string, 0, len(permissionNames))
	for p := range permissionNames {
		res.Permissions = append(res.Permissions, p)
	}

	return nil
}

// ListPermissionsByResource implements the PermissionServiceHandler interface
func (g Service) ListPermissionsByResource(ctx context.Context, req *settingssvc.ListPermissionsByResourceRequest, res *settingssvc.ListPermissionsByResourceResponse) error {
	if validationError := validateListPermissionsByResource(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	permissions, err := g.manager.ListPermissionsByResource(req.GetResource(), g.getRoleIDs(ctx))
	if err != nil {
		return merrors.BadRequest(g.id, "%s", err)
	}
	res.Permissions = permissions
	return nil
}

// GetPermissionByID implements the PermissionServiceHandler interface
func (g Service) GetPermissionByID(ctx context.Context, req *settingssvc.GetPermissionByIDRequest, res *settingssvc.GetPermissionByIDResponse) error {
	if validationError := validateGetPermissionByID(req); validationError != nil {
		return merrors.BadRequest(g.id, "%s", validationError)
	}
	permission, err := g.manager.ReadPermissionByID(req.GetPermissionId(), g.getRoleIDs(ctx))
	if err != nil {
		return merrors.BadRequest(g.id, "%s", err)
	}
	if permission == nil {
		return merrors.NotFound(g.id, "%s", fmt.Errorf("permission %s not found in roles", req.GetPermissionId()))
	}
	res.Permission = permission
	return nil
}

// cleanUpResource makes sure that the account uuid of the authenticated user is injected if needed.
func cleanUpResource(ctx context.Context, resource *settingsmsg.Resource) {
	if resource != nil && resource.GetType() == settingsmsg.Resource_TYPE_USER {
		resource.Id = getValidatedAccountUUID(ctx, resource.GetId())
	}
}

// getValidatedAccountUUID converts `me` into an actual account uuid from the context, if possible.
// the result of this function will always be a valid lower-case UUID or an empty string.
func getValidatedAccountUUID(ctx context.Context, accountUUID string) string {
	if accountUUID == "me" {
		if ownAccountUUID, ok := metadata.Get(ctx, middleware.AccountID); ok {
			accountUUID = ownAccountUUID
		}
	}
	if accountUUID == "me" {
		// no matter what happens above, an accountUUID of `me` must not be passed on. Clear it instead.
		accountUUID = ""
	}
	return accountUUID
}

// getRoleIDs extracts the roleIDs of the authenticated user from the context.
func (g Service) getRoleIDs(ctx context.Context) []string {
	if ownRoleIDs, ok := roles.ReadRoleIDsFromContext(ctx); ok {
		return ownRoleIDs
	}
	if accountID, ok := metadata.Get(ctx, middleware.AccountID); ok {
		assignments, err := g.manager.ListRoleAssignments(accountID)
		if err != nil {
			g.logger.Info().Err(err).Str("userid", accountID).Msg("failed to get roles for user")
			return nil
		}

		ownRoleIDs := make([]string, 0, len(assignments))
		for _, a := range assignments {
			ownRoleIDs = append(ownRoleIDs, a.GetRoleId())
		}
		return ownRoleIDs
	}
	g.logger.Info().Msg("failed to get accountID from context")
	return nil
}

func (g Service) getValueWithIdentifier(value *settingsmsg.Value) (*settingsmsg.ValueWithIdentifier, error) {
	bundle, err := g.manager.ReadBundle(value.GetBundleId())
	if err != nil {
		return nil, err
	}
	setting, err := g.manager.ReadSetting(value.GetSettingId())
	if err != nil {
		return nil, err
	}
	return &settingsmsg.ValueWithIdentifier{
		Identifier: &settingsmsg.Identifier{
			Extension: bundle.GetExtension(),
			Bundle:    bundle.GetName(),
			Setting:   setting.GetName(),
		},
		Value: value,
	}, nil
}

func (g Service) hasStaticPermission(ctx context.Context, permissionID string) bool {
	roleIDs, ok := roles.ReadRoleIDsFromContext(ctx)
	if !ok {
		// TODO add system role for internal requests.
		// - at least the proxy needs to look up account info
		// - glauth needs to make bind requests
		// tracked as OCIS-454

		accountID, ok := metadata.Get(ctx, middleware.AccountID)
		if !ok {
			return false
		}
		assignments, err := g.manager.ListRoleAssignments(accountID)
		if err != nil {
			return false
		}

		// deduplicate roleids
		uniqueRoleIDs := make(map[string]struct{})
		for _, a := range assignments {
			uniqueRoleIDs[a.GetRoleId()] = struct{}{}
		}
		roleIDs = make([]string, 0, len(uniqueRoleIDs))
		for a := range uniqueRoleIDs {
			roleIDs = append(roleIDs, a)
		}
	}
	p, err := g.manager.ReadPermissionByID(permissionID, roleIDs)
	return err == nil && p != nil
}

func (g Service) checkStaticPermissionsByBundleID(ctx context.Context, bundleID string) error {
	bundle, err := g.manager.ReadBundle(bundleID)
	if err != nil {
		return merrors.NotFound(g.id, "bundle not found: %s", err)
	}
	return g.checkStaticPermissionsByBundleType(ctx, bundle.GetType())
}

func (g Service) checkStaticPermissionsByBundleType(ctx context.Context, bundleType settingsmsg.Bundle_Type) error {
	if bundleType == settingsmsg.Bundle_TYPE_ROLE {
		if !g.hasStaticPermission(ctx, RoleManagementPermissionID) {
			return merrors.Forbidden(g.id, "user has no role management permission")
		}
		return nil
	}
	if !g.hasStaticPermission(ctx, SettingsManagementPermissionID) {
		return merrors.Forbidden(g.id, "user has no settings management permission")
	}
	return nil
}

func (g Service) isCurrentUser(ctx context.Context, accountID string) bool {
	ownAccountID, ok := metadata.Get(ctx, middleware.AccountID)
	if !ok {
		return false
	}
	return accountID == ownAccountID
}

func (g Service) canManageRoles(ctx context.Context) bool {
	return g.hasStaticPermission(ctx, RoleManagementPermissionID)
}

func (g Service) getUserLocale(ctx context.Context, userID string) (string, error) {
	var resp settingssvc.GetValueResponse
	err := g.GetValueByUniqueIdentifiers(
		ctx,
		&settingssvc.GetValueByUniqueIdentifiersRequest{
			AccountUuid: userID,
			SettingId:   defaults.SettingUUIDProfileLanguage,
		},
		&resp,
	)
	if err != nil {
		return "", err
	}
	val := resp.GetValue().GetValue().GetListValue().GetValues()
	if len(val) == 0 {
		return "", errors.New("no language setting found")
	}
	return val[0].GetStringValue(), nil
}

func formatPermissionName(setting *settingsmsg.Setting) string {
	constraint := strings.TrimPrefix(setting.GetPermissionValue().GetConstraint().String(), "CONSTRAINT_")
	return setting.GetName() + "." + strings.ToLower(constraint)
}

func translateBundle(bundle *settingsmsg.Bundle, t *gotext.Locale) *settingsmsg.Bundle {
	for i, set := range bundle.GetSettings() {
		switch set.GetId() {
		default:
			continue
		case defaults.SettingUUIDProfileEmailSendingInterval:
			// translate interval names ('Instant', 'Daily', 'Weekly', 'Never')
			value := set.GetSingleChoiceValue()
			for i, v := range value.GetOptions() {
				value.Options[i].DisplayValue = t.Get(v.GetDisplayValue(), []interface{}{}...)
			}
			set.Value = &settingsmsg.Setting_SingleChoiceValue{SingleChoiceValue: value}
			fallthrough
		case defaults.SettingUUIDProfileEventShareCreated,
			defaults.SettingUUIDProfileEventShareRemoved,
			defaults.SettingUUIDProfileEventShareExpired,
			defaults.SettingUUIDProfileEventSpaceShared,
			defaults.SettingUUIDProfileEventSpaceUnshared,
			defaults.SettingUUIDProfileEventSpaceMembershipExpired,
			defaults.SettingUUIDProfileEventSpaceDisabled,
			defaults.SettingUUIDProfileEventSpaceDeleted:
			// translate event names ('Share Received', 'Share Removed', ...)
			set.DisplayName = t.Get(set.GetDisplayName(), []interface{}{}...)
			// translate event descriptions ('Notify me when I receive a share', ...)
			set.Description = t.Get(set.GetDescription(), []interface{}{}...)
			bundle.Settings[i] = set
		}
	}
	return bundle
}
