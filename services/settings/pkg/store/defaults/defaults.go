package defaults

import (
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/config"
)

const (
	// BundleUUIDRoleAdmin represents the admin role
	BundleUUIDRoleAdmin = "71881883-1768-46bd-a24d-a356a2afdf7f"
	// BundleUUIDRoleSpaceAdmin represents the space admin role
	BundleUUIDRoleSpaceAdmin = "2aadd357-682c-406b-8874-293091995fdd"
	// BundleUUIDRoleUser represents the user role.
	BundleUUIDRoleUser = "d7beeea8-8ff4-406b-8fb6-ab2dd81e6b11"
	// BundleUUIDRoleUserLight represents the user light role.
	BundleUUIDRoleUserLight = "38071a68-456a-4553-846a-fa67bf5596cc"
	// BundleUUIDRoleGuest represents the guest role.
	BundleUUIDRoleGuest = "38071a68-456a-4553-846a-fa67bf5596cc"
	// BundleUUIDProfile represents the user profile.
	BundleUUIDProfile = "2a506de7-99bd-4f0d-994e-c38e72c28fd9"
	// BundleUUIDServiceAccount represents the service account role.
	BundleUUIDServiceAccount = "bcceed81-c610-49cc-ab77-39a024e8da12"
	// SettingUUIDProfileLanguage is the hardcoded setting UUID for the user profile language
	SettingUUIDProfileLanguage = "aa8cfbe5-95d4-4f7e-a032-c3c01f5f062f"
	// SettingUUIDProfileDisableNotifications is the hardcoded setting UUID for the disable notifications setting
	SettingUUIDProfileDisableNotifications = "33ffb5d6-cd07-4dc0-afb0-84f7559ae438"
	// SettingUUIDProfileAutoAcceptShares is the hardcoded setting UUID for the disable notifications setting
	SettingUUIDProfileAutoAcceptShares = "ec3ed4a3-3946-4efc-8f9f-76d38b12d3a9"

	// SettingUUIDProfileEmailSendingInterval is the hardcoded setting UUID for the email sending interval setting
	SettingUUIDProfileEmailSendingInterval = "08dec2fe-3f97-42a9-9d1b-500855e92f25"
	// SettingUUIDProfileEventShareCreated it the hardcoded setting UUID for the send in app setting
	SettingUUIDProfileEventShareCreated = "872d8ef6-6f2a-42ab-af7d-f53cc81d7046"
	// SettingUUIDProfileEventShareRemoved is the hardcoded setting UUID for the send in app setting
	SettingUUIDProfileEventShareRemoved = "d7484394-8321-4c84-9677-741ba71e1f80"
	// SettingUUIDProfileEventShareExpired is the hardcoded setting UUID for the send in app setting
	SettingUUIDProfileEventShareExpired = "e1aa0b7c-1b0f-4072-9325-c643c89fee4e"
	// SettingUUIDProfileEventSpaceShared is the hardcoded setting UUID for the send in app setting
	SettingUUIDProfileEventSpaceShared = "694d5ee1-a41c-448c-8d14-396b95d2a918"
	// SettingUUIDProfileEventSpaceUnshared is the hardcoded setting UUID for the send in app setting
	SettingUUIDProfileEventSpaceUnshared = "26c20e0e-98df-4483-8a77-759b3a766af0"
	// SettingUUIDProfileEventSpaceMembershipExpired is the hardcoded setting UUID for the send in app setting
	SettingUUIDProfileEventSpaceMembershipExpired = "7275921e-b737-4074-ba91-3c2983be3edd"
	// SettingUUIDProfileEventSpaceDisabled is the hardcoded setting UUID for the send in app setting
	SettingUUIDProfileEventSpaceDisabled = "eb5c716e-03be-42c6-9ed1-1105d24e109f"
	// SettingUUIDProfileEventSpaceDeleted is the hardcoded setting UUID for the send in app setting
	SettingUUIDProfileEventSpaceDeleted = "094ceca9-5a00-40ba-bb1a-bbc7bccd39ee"
	// SettingUUIDProfileEventPostprocessingStepFinished is the hardcoded setting UUID for the send in mail setting
	SettingUUIDProfileEventPostprocessingStepFinished = "fe0a3011-d886-49c8-b797-33d02fa426ef"
)

// GenerateBundlesDefaultRoles bootstraps the default roles.
func GenerateBundlesDefaultRoles() []*settingsmsg.Bundle {
	return []*settingsmsg.Bundle{
		generateBundleAdminRole(),
		generateBundleUserRole(),
		generateBundleUserLightRole(),
		generateBundleProfileRequest(),
		generateBundleSpaceAdminRole(),
	}
}

// GenerateDefaultProfileBundle return the default profile bundle.
func GenerateDefaultProfileBundle() *settingsmsg.Bundle {
	return generateBundleProfileRequest()
}

// ServiceAccountBundle returns the service account bundle
func ServiceAccountBundle() *settingsmsg.Bundle {
	return &settingsmsg.Bundle{
		Id:          BundleUUIDServiceAccount,
		Name:        "service-account",
		Type:        settingsmsg.Bundle_TYPE_ROLE,
		Extension:   "opencloud-roles",
		DisplayName: "Service Account",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Settings: []*settingsmsg.Setting{
			AccountManagementPermission(All),
			ChangeLogoPermission(All),
			CreatePublicLinkPermission(All),
			CreateSharePermission(All),
			CreateSpacesPermission(All),
			DeletePersonalSpacesPermission(All),
			DeleteProjectSpacesPermission(All),
			DeleteReadOnlyPublicLinkPasswordPermission(All),
			GroupManagementPermission(All),
			LanguageManagementPermission(All),
			ListFavoritesPermission(All),
			ListSpacesPermission(All),
			ManageSpacePropertiesPermission(All),
			RoleManagementPermission(All),
			SetPersonalSpaceQuotaPermission(All),
			SetProjectSpaceQuotaPermission(All),
			SettingsManagementPermission(All),
			SpaceAbilityPermission(All),
			WriteFavoritesPermission(All),
			// TODO: add more permissions? remove some?
		},
	}
}

func generateBundleAdminRole() *settingsmsg.Bundle {
	return &settingsmsg.Bundle{
		Id:          BundleUUIDRoleAdmin,
		Name:        "admin",
		Type:        settingsmsg.Bundle_TYPE_ROLE,
		Extension:   "opencloud-roles",
		DisplayName: "Admin",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Settings: []*settingsmsg.Setting{
			AccountManagementPermission(All),
			AutoAcceptSharesPermission(Own),
			ChangeLogoPermission(All),
			CreatePublicLinkPermission(All),
			CreateSharePermission(All),
			CreateSpacesPermission(All),
			DeletePersonalSpacesPermission(All),
			DeleteProjectSpacesPermission(All),
			DeleteReadOnlyPublicLinkPasswordPermission(All),
			DisableEmailNotificationsPermission(Own),
			ProfileEmailSendingIntervalPermission(Own),
			ProfileEventShareCreatedPermission(Own),
			ProfileEventShareRemovedPermission(Own),
			ProfileEventShareExpiredPermission(Own),
			ProfileEventSpaceSharedPermission(Own),
			ProfileEventSpaceUnsharedPermission(Own),
			ProfileEventSpaceMembershipExpiredPermission(Own),
			ProfileEventSpaceDisabledPermission(Own),
			ProfileEventSpaceDeletedPermission(Own),
			ProfileEventPostprocessingStepFinishedPermission(Own),
			GroupManagementPermission(All),
			LanguageManagementPermission(All),
			ListFavoritesPermission(Own),
			ListSpacesPermission(All),
			ManageSpacePropertiesPermission(All),
			RoleManagementPermission(All),
			SetPersonalSpaceQuotaPermission(All),
			SetProjectSpaceQuotaPermission(All),
			SettingsManagementPermission(All),
			SpaceAbilityPermission(All),
			WebOfficeManagementPermssion(All),
			WriteFavoritesPermission(Own),
		},
	}
}

func generateBundleSpaceAdminRole() *settingsmsg.Bundle {
	return &settingsmsg.Bundle{
		Id:          BundleUUIDRoleSpaceAdmin,
		Name:        "spaceadmin",
		Type:        settingsmsg.Bundle_TYPE_ROLE,
		Extension:   "opencloud-roles",
		DisplayName: "Space Admin",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Settings: []*settingsmsg.Setting{
			AutoAcceptSharesPermission(Own),
			CreatePublicLinkPermission(All),
			CreateSharePermission(All),
			CreateSpacesPermission(All),
			DeleteProjectSpacesPermission(All),
			DeleteReadOnlyPublicLinkPasswordPermission(All),
			DisableEmailNotificationsPermission(Own),
			ProfileEmailSendingIntervalPermission(Own),
			ProfileEventShareCreatedPermission(Own),
			ProfileEventShareRemovedPermission(Own),
			ProfileEventShareExpiredPermission(Own),
			ProfileEventSpaceSharedPermission(Own),
			ProfileEventSpaceUnsharedPermission(Own),
			ProfileEventSpaceMembershipExpiredPermission(Own),
			ProfileEventSpaceDisabledPermission(Own),
			ProfileEventSpaceDeletedPermission(Own),
			ProfileEventPostprocessingStepFinishedPermission(Own),
			LanguageManagementPermission(Own),
			ListFavoritesPermission(Own),
			ListSpacesPermission(All),
			ManageSpacePropertiesPermission(All),
			SelfManagementPermission(Own),
			SetProjectSpaceQuotaPermission(All),
			SpaceAbilityPermission(All),
			WriteFavoritesPermission(Own),
		},
	}
}

func generateBundleUserRole() *settingsmsg.Bundle {
	return &settingsmsg.Bundle{
		Id:          BundleUUIDRoleUser,
		Name:        "user",
		Type:        settingsmsg.Bundle_TYPE_ROLE,
		Extension:   "opencloud-roles",
		DisplayName: "User",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Settings: []*settingsmsg.Setting{
			AutoAcceptSharesPermission(Own),
			CreatePublicLinkPermission(All),
			CreateSharePermission(All),
			CreateSpacesPermission(Own),
			DisableEmailNotificationsPermission(Own),
			ProfileEmailSendingIntervalPermission(Own),
			ProfileEventShareCreatedPermission(Own),
			ProfileEventShareRemovedPermission(Own),
			ProfileEventShareExpiredPermission(Own),
			ProfileEventSpaceSharedPermission(Own),
			ProfileEventSpaceUnsharedPermission(Own),
			ProfileEventSpaceMembershipExpiredPermission(Own),
			ProfileEventSpaceDisabledPermission(Own),
			ProfileEventSpaceDeletedPermission(Own),
			ProfileEventPostprocessingStepFinishedPermission(Own),
			LanguageManagementPermission(Own),
			ListFavoritesPermission(Own),
			SelfManagementPermission(Own),
			WriteFavoritesPermission(Own),
		},
	}
}

func generateBundleUserLightRole() *settingsmsg.Bundle {
	return &settingsmsg.Bundle{
		Id:          BundleUUIDRoleUserLight,
		Name:        "user-light",
		Type:        settingsmsg.Bundle_TYPE_ROLE,
		Extension:   "opencloud-roles",
		DisplayName: "User Light",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Settings: []*settingsmsg.Setting{
			AutoAcceptSharesPermission(Own),
			DisableEmailNotificationsPermission(Own),
			ProfileEmailSendingIntervalPermission(Own),
			LanguageManagementPermission(Own),
		},
	}
}

func generateBundleProfileRequest() *settingsmsg.Bundle {
	return &settingsmsg.Bundle{
		Id:        BundleUUIDProfile,
		Name:      "profile",
		Extension: "opencloud-accounts",
		Type:      settingsmsg.Bundle_TYPE_DEFAULT,
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		DisplayName: "Profile",
		Settings: []*settingsmsg.Setting{
			{
				Id:          SettingUUIDProfileLanguage,
				Name:        "language",
				DisplayName: "Language",
				Description: "User language",
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &languageSetting,
			},
			{
				Id:          SettingUUIDProfileDisableNotifications,
				Name:        "disable-email-notifications",
				DisplayName: "Disable Email Notifications",
				Description: "Disable email notifications",
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_BoolValue{BoolValue: &settingsmsg.Bool{Default: false, Label: "disable notifications"}},
			},
			{
				Id:          SettingUUIDProfileAutoAcceptShares,
				Name:        "auto-accept-shares",
				DisplayName: "Auto accept shares",
				Description: "Automatically accept shares",
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_BoolValue{BoolValue: &settingsmsg.Bool{Default: true, Label: "auto accept shares"}},
			},
			{
				Id:          SettingUUIDProfileEmailSendingInterval,
				Name:        "email-sending-interval-options",
				DisplayName: TemplateEmailSendingInterval,
				Description: TemplateEmailSendingIntervalDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &sendEmailOptions,
			},
			{
				Id:          SettingUUIDProfileEventShareCreated,
				Name:        "event-share-created-options",
				DisplayName: TemplateShareCreated,
				Description: TemplateShareCreatedDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_MultiChoiceCollectionValue{
					MultiChoiceCollectionValue: &settingsmsg.MultiChoiceCollection{
						Options: []*settingsmsg.MultiChoiceCollectionOption{
							&optionInAppTrue,
							&optionMailTrue,
						},
					},
				},
			},
			{
				Id:          SettingUUIDProfileEventShareRemoved,
				Name:        "event-share-removed-options",
				DisplayName: TemplateShareRemoved,
				Description: TemplateShareRemovedDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_MultiChoiceCollectionValue{
					MultiChoiceCollectionValue: &settingsmsg.MultiChoiceCollection{
						Options: []*settingsmsg.MultiChoiceCollectionOption{
							&optionInAppTrue,
							&optionMailFalseDisabled,
						},
					},
				},
			},
			{
				Id:          SettingUUIDProfileEventShareExpired,
				Name:        "event-share-expired-options",
				DisplayName: TemplateShareExpired,
				Description: TemplateShareExpiredDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_MultiChoiceCollectionValue{
					MultiChoiceCollectionValue: &settingsmsg.MultiChoiceCollection{
						Options: []*settingsmsg.MultiChoiceCollectionOption{
							&optionInAppTrue,
							&optionMailTrue,
						},
					},
				},
			},
			{
				Id:          SettingUUIDProfileEventSpaceShared,
				Name:        "event-space-shared-options",
				DisplayName: TemplateSpaceShared,
				Description: TemplateSpaceSharedDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_MultiChoiceCollectionValue{
					MultiChoiceCollectionValue: &settingsmsg.MultiChoiceCollection{
						Options: []*settingsmsg.MultiChoiceCollectionOption{
							&optionInAppTrue,
							&optionMailTrue,
						},
					},
				},
			},
			{
				Id:          SettingUUIDProfileEventSpaceUnshared,
				Name:        "event-space-unshared-options",
				DisplayName: TemplateSpaceUnshared,
				Description: TemplateSpaceUnsharedDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_MultiChoiceCollectionValue{
					MultiChoiceCollectionValue: &settingsmsg.MultiChoiceCollection{
						Options: []*settingsmsg.MultiChoiceCollectionOption{
							&optionInAppTrue,
							&optionMailTrue,
						},
					},
				},
			},
			{
				Id:          SettingUUIDProfileEventSpaceMembershipExpired,
				Name:        "event-space-membership-expired-options",
				DisplayName: TemplateSpaceMembershipExpired,
				Description: TemplateSpaceMembershipExpiredDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_MultiChoiceCollectionValue{
					MultiChoiceCollectionValue: &settingsmsg.MultiChoiceCollection{
						Options: []*settingsmsg.MultiChoiceCollectionOption{
							&optionInAppTrue,
							&optionMailTrue,
						},
					},
				},
			},
			{
				Id:          SettingUUIDProfileEventSpaceDisabled,
				Name:        "event-space-disabled-options",
				DisplayName: TemplateSpaceDisabled,
				Description: TemplateSpaceDisabledDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_MultiChoiceCollectionValue{
					MultiChoiceCollectionValue: &settingsmsg.MultiChoiceCollection{
						Options: []*settingsmsg.MultiChoiceCollectionOption{
							&optionInAppTrue,
							&optionMailFalseDisabled,
						},
					},
				},
			},
			{
				Id:          SettingUUIDProfileEventSpaceDeleted,
				Name:        "event-space-deleted-options",
				DisplayName: TemplateSpaceDeleted,
				Description: TemplateSpaceDeletedDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_MultiChoiceCollectionValue{
					MultiChoiceCollectionValue: &settingsmsg.MultiChoiceCollection{
						Options: []*settingsmsg.MultiChoiceCollectionOption{
							&optionInAppTrue,
							&optionMailFalseDisabled,
						},
					},
				},
			},
			{
				Id:          SettingUUIDProfileEventPostprocessingStepFinished,
				Name:        "event-postprocessing-step-finished-options",
				DisplayName: TemplateFileRejected,
				Description: TemplateFileRejectedDescription,
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Setting_MultiChoiceCollectionValue{
					MultiChoiceCollectionValue: &settingsmsg.MultiChoiceCollection{
						Options: []*settingsmsg.MultiChoiceCollectionOption{
							&optionInAppTrue,
							&optionMailFalseDisabled,
						},
					},
				},
			},
		},
	}
}

var sendEmailOptions = settingsmsg.Setting_SingleChoiceValue{
	SingleChoiceValue: &settingsmsg.SingleChoiceList{
		Options: []*settingsmsg.ListOption{
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "instant",
					},
				},
				DisplayValue: TemplateIntervalInstant,
				Default:      true,
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "daily",
					},
				},
				DisplayValue: TemplateIntervalDaily,
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "weekly",
					},
				},
				DisplayValue: TemplateIntervalWeekly,
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "never",
					},
				},
				DisplayValue: TemplateIntervalNever,
			},
		},
	},
}

var optionInAppTrue = settingsmsg.MultiChoiceCollectionOption{
	Key:          "in-app",
	DisplayValue: "In-App",
	Value: &settingsmsg.MultiChoiceCollectionOptionValue{
		Option: &settingsmsg.MultiChoiceCollectionOptionValue_BoolValue{
			BoolValue: &settingsmsg.Bool{
				Default: true,
			},
		},
	},
}

var optionMailTrue = settingsmsg.MultiChoiceCollectionOption{
	Key:          "mail",
	DisplayValue: "Email",
	Value: &settingsmsg.MultiChoiceCollectionOptionValue{
		Option: &settingsmsg.MultiChoiceCollectionOptionValue_BoolValue{
			BoolValue: &settingsmsg.Bool{
				Default: true,
			},
		},
	},
}

var optionMailFalseDisabled = settingsmsg.MultiChoiceCollectionOption{
	Key:          "mail",
	Attribute:    "disabled",
	DisplayValue: "Email",
	Value: &settingsmsg.MultiChoiceCollectionOptionValue{
		Option: &settingsmsg.MultiChoiceCollectionOptionValue_BoolValue{
			BoolValue: &settingsmsg.Bool{
				Default: false,
			},
		},
	},
}

// TODO: languageSetting needed?
var languageSetting = settingsmsg.Setting_SingleChoiceValue{
	SingleChoiceValue: &settingsmsg.SingleChoiceList{
		Options: []*settingsmsg.ListOption{
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "bg",
					},
				},
				DisplayValue: "български",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "cs",
					},
				},
				DisplayValue: "Czech",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "de",
					},
				},
				DisplayValue: "Deutsch",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "en",
					},
				},
				DisplayValue: "English",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "es",
					},
				},
				DisplayValue: "Español",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "fr",
					},
				},
				DisplayValue: "Français",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "gl",
					},
				},
				DisplayValue: "Galego",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "it",
					},
				},
				DisplayValue: "Italiano",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "nl",
					},
				},
				DisplayValue: "Nederlands",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "ko",
					},
				},
				DisplayValue: "한국어",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "sq",
					},
				},
				DisplayValue: "Shqipja",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "sv",
					},
				},
				DisplayValue: "Svenska",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "tr",
					},
				},
				DisplayValue: "Türkçe",
			},
			{
				Value: &settingsmsg.ListOptionValue{
					Option: &settingsmsg.ListOptionValue_StringValue{
						StringValue: "zh",
					},
				},
				DisplayValue: "汉语",
			},
		},
	},
}

// DefaultRoleAssignments returns (as one might guess) the default role assignments
func DefaultRoleAssignments(cfg *config.Config) []*settingsmsg.UserRoleAssignment {
	assignments := []*settingsmsg.UserRoleAssignment{}

	if cfg.SetupDefaultAssignments {
		assignments = []*settingsmsg.UserRoleAssignment{
			// default users with role "user"
			{
				AccountUuid: "b1f74ec4-dd7e-11ef-a543-03775734d0f7",
				RoleId:      BundleUUIDRoleUser,
			},
			{
				AccountUuid: "056fc874-dd7f-11ef-ba84-af6fca4b7289",
				RoleId:      BundleUUIDRoleUser,
			},
			{
				AccountUuid: "60708dda-e897-11ef-919f-bbb7437d6ec2",
				RoleId:      BundleUUIDRoleUser,
			},
			{
				// additional admin user
				AccountUuid: "cd88bf9a-dd7f-11ef-a609-7f78deb2345f", // demo user "dennis"
				RoleId:      BundleUUIDRoleAdmin,
			},
			{
				// default users with role "spaceadmin"
				AccountUuid: "801abee4-dd7f-11ef-a324-83f55a754b62",
				RoleId:      BundleUUIDRoleSpaceAdmin,
			},
			{
				// service user
				AccountUuid: "service-user-id",
				RoleId:      BundleUUIDRoleAdmin,
			},
		}
	}

	if cfg.AdminUserID != "" {
		// default admin user
		assignments = append(assignments, &settingsmsg.UserRoleAssignment{
			AccountUuid: cfg.AdminUserID,
			RoleId:      BundleUUIDRoleAdmin,
		})
	}

	return assignments
}
