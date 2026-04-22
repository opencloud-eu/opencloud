package svc

import (
	"errors"
	"fmt"
	"regexp"

	validation "github.com/invopop/validation"
	"github.com/invopop/validation/is"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
)

var (
	regexForAccountUUID = regexp.MustCompile(`^[A-Za-z0-9\-_.+@:]+$`)
	requireAccountID    = []validation.Rule{
		// use rule for validation error message consistency (".. must not be blank" on empty strings)
		validation.Required,
		validation.Match(regexForAccountUUID),
	}
	regexForKeys        = regexp.MustCompile(`^[A-Za-z0-9\-_]*$`)
	requireAlphanumeric = []validation.Rule{
		validation.Required,
		validation.Match(regexForKeys),
	}
)

func validateSaveBundle(req *settingssvc.SaveBundleRequest) error {
	if err := validation.ValidateStruct(
		req.Bundle,
		validation.Field(&req.Bundle.Id, validation.When(req.Bundle.Id != "", is.UUID)),
		validation.Field(&req.Bundle.Name, requireAlphanumeric...),
		validation.Field(&req.Bundle.Type, validation.NotIn(settingsmsg.Bundle_TYPE_UNKNOWN)),
		validation.Field(&req.Bundle.Extension, requireAlphanumeric...),
		validation.Field(&req.Bundle.DisplayName, validation.Required),
		validation.Field(&req.Bundle.Settings, validation.Required),
	); err != nil {
		return err
	}
	if err := validateResource(req.Bundle.Resource); err != nil {
		return err
	}
	for i := range req.Bundle.Settings {
		if err := validateSetting(req.Bundle.Settings[i]); err != nil {
			return err
		}
	}
	return nil
}

func validateGetBundle(req *settingssvc.GetBundleRequest) error {
	return validation.Validate(&req.BundleId, is.UUID)
}

func validateListBundles(req *settingssvc.ListBundlesRequest) error {
	return nil
}

func validateAddSettingToBundle(req *settingssvc.AddSettingToBundleRequest) error {
	if err := validation.ValidateStruct(req, validation.Field(&req.BundleId, is.UUID)); err != nil {
		return err
	}
	return validateSetting(req.Setting)
}

func validateRemoveSettingFromBundle(req *settingssvc.RemoveSettingFromBundleRequest) error {
	return validation.ValidateStruct(
		req,
		validation.Field(&req.BundleId, is.UUID),
		validation.Field(&req.SettingId, is.UUID),
	)
}

func validateSaveValue(req *settingssvc.SaveValueRequest) error {
	if err := validation.ValidateStruct(
		req.Value,
		validation.Field(&req.Value.Id, validation.When(req.Value.Id != "", is.UUID)),
		validation.Field(&req.Value.BundleId, is.UUID),
		validation.Field(&req.Value.SettingId, is.UUID),
		validation.Field(&req.Value.AccountUuid, requireAccountID...),
	); err != nil {
		return err
	}

	if err := validateResource(req.Value.Resource); err != nil {
		return err
	}

	if v := req.GetValue().GetCollectionValue().GetValues(); v != nil {
		for i := range v {
			if err := validation.Validate(v[i].GetKey(), validation.Required); err != nil {
				return fmt.Errorf("collectionValue.values[%d].key %w", i, err)
			}
		}
	}

	// TODO: validate values against the respective setting. need to check if constraints of the setting are fulfilled.
	return nil
}

func validateGetValue(req *settingssvc.GetValueRequest) error {
	return validation.Validate(req.Id, is.UUID)
}

func validateGetValueByUniqueIdentifiers(req *settingssvc.GetValueByUniqueIdentifiersRequest) error {
	return validation.ValidateStruct(
		req,
		validation.Field(&req.SettingId, is.UUID),
		validation.Field(&req.AccountUuid, requireAccountID...),
	)
}

func validateListValues(req *settingssvc.ListValuesRequest) error {
	return validation.ValidateStruct(
		req,
		validation.Field(&req.BundleId, validation.When(req.BundleId != "", is.UUID)),
		validation.Field(&req.AccountUuid, validation.When(req.AccountUuid != "", validation.Match(regexForAccountUUID))),
	)
}

func validateListRoles(req *settingssvc.ListBundlesRequest) error {
	return nil
}

func validateListRoleAssignments(req *settingssvc.ListRoleAssignmentsRequest) error {
	return validation.Validate(req.AccountUuid, requireAccountID...)
}

func validateListRoleAssignmentsFiltered(req *settingssvc.ListRoleAssignmentsFilteredRequest) error {
	return validation.ValidateStruct(
		req,
		validation.Field(&req.Filters,
			validation.Required,
			validation.Length(1, 1),
			validation.Each(validation.By(validateUserRoleAssignmentFilter)),
		),
	)
}

func validateUserRoleAssignmentFilter(values any) error {
	filter, ok := values.(*settingsmsg.UserRoleAssignmentFilter)
	if !ok {
		return errors.New("expected UserRoleAssignmentFilter")
	}
	return validation.ValidateStruct(
		filter,
		validation.Field(&filter.Type,
			validation.Required,
			validation.In(settingsmsg.UserRoleAssignmentFilter_TYPE_ACCOUNT, settingsmsg.UserRoleAssignmentFilter_TYPE_ROLE),
		),
		validation.Field(&filter.Term,
			validation.When(
				filter.Type == settingsmsg.UserRoleAssignmentFilter_TYPE_ACCOUNT,
				validation.By(validateFilterAccountUUID),
			),
			validation.When(
				filter.Type == settingsmsg.UserRoleAssignmentFilter_TYPE_ROLE,
				validation.By(validateFilterRoleID),
			),
		),
	)
}

func validateFilterRoleID(value any) error {
	roleTerm, ok := value.(*settingsmsg.UserRoleAssignmentFilter_RoleId)
	if !ok {
		return errors.New("expected UserRoleAssignmentFilter_RoleId")
	}
	return validation.Validate(&roleTerm.RoleId, is.UUID)
}

func validateFilterAccountUUID(value any) error {
	accountTerm, ok := value.(*settingsmsg.UserRoleAssignmentFilter_AccountUuid)
	if !ok {
		return errors.New("expected UserRoleAssignmentFilter_AccountUuid")
	}
	return validation.Validate(&accountTerm.AccountUuid, requireAccountID...)
}

func validateAssignRoleToUser(req *settingssvc.AssignRoleToUserRequest) error {
	return validation.ValidateStruct(
		req,
		validation.Field(&req.AccountUuid, requireAccountID...),
		validation.Field(&req.RoleId, is.UUID),
	)
}

func validateRemoveRoleFromUser(req *settingssvc.RemoveRoleFromUserRequest) error {
	return validation.ValidateStruct(
		req,
		validation.Field(&req.Id, is.UUID),
	)
}

func validateListPermissionsByResource(req *settingssvc.ListPermissionsByResourceRequest) error {
	return validateResource(req.Resource)
}

func validateGetPermissionByID(req *settingssvc.GetPermissionByIDRequest) error {
	return validation.ValidateStruct(
		req,
		validation.Field(&req.PermissionId, requireAlphanumeric...),
	)
}

// validateResource is an internal helper for validating the content of a resource.
func validateResource(resource *settingsmsg.Resource) error {
	if err := validation.Validate(&resource, validation.Required); err != nil {
		return err
	}
	return validation.Validate(&resource, validation.NotIn(settingsmsg.Resource_TYPE_UNKNOWN))
}

// validateSetting is an internal helper for validating the content of a setting.
func validateSetting(setting *settingsmsg.Setting) error {
	// TODO: make sanity checks, like for int settings, min <= default <= max.
	if err := validation.ValidateStruct(
		setting,
		validation.Field(&setting.Id, validation.When(setting.Id != "", is.UUID)),
		validation.Field(&setting.Name, requireAlphanumeric...),
	); err != nil {
		return err
	}
	return validateResource(setting.Resource)
}
