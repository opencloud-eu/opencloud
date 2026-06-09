package defaults

import settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"

var (
	// All is a convenience variable to set constraint to all
	All = settingsmsg.Permission_CONSTRAINT_ALL
	// Own is a convenience variable to set constraint to own
	Own = settingsmsg.Permission_CONSTRAINT_OWN
)

// AccountManagementPermission is the permission to manage accounts
func AccountManagementPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "8e587774-d929-4215-910b-a317b1e80f73",
		Name:        "Accounts.ReadWrite",
		DisplayName: "Account Management",
		Description: "This permission gives full access to everything that is related to account management.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_USER,
			Id:   "all",
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// AutoAcceptSharesPermission is the permission to enable share auto-accept
func AutoAcceptSharesPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "4e41363c-a058-40a5-aec8-958897511209",
		Name:        "AutoAcceptShares.ReadWriteDisabled",
		DisplayName: "enable/disable auto accept shares",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileAutoAcceptShares,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ChangeLogoPermission is the permission to change the logo
func ChangeLogoPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "ed83fc10-1f54-4a9e-b5a7-fb517f5f3e01",
		Name:        "Logo.Write",
		DisplayName: "Change logo",
		Description: "This permission permits to change the system logo.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// CollaborationManageFontsPermission is the permission to manage fonts
func CollaborationManageFontsPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "ed83fc10-1f54-4a9e-b5a7-fb517f5f3e01",
		Name:        "Collaboration.Fonts.Manage",
		DisplayName: "Manage fonts",
		Description: "This permission permits to manage the collaboration fonts.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// CollaborationPublishNotificationPermission is the permission to manage fonts
func CollaborationPublishNotificationPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "43e5948e-8238-41d6-9ef1-f259f00591db",
		Name:        "Collaboration.Notification.Publish",
		DisplayName: "Publish collaboration notifications",
		Description: "This permission permits to publish collaboration related notifications.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// CreatePublicLinkPermission is the permission to create public links
func CreatePublicLinkPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "11516bbd-7157-49e1-b6ac-d00c820f980b",
		Name:        "PublicLink.Write",
		DisplayName: "Write publiclink",
		Description: "This permission allows creating public links.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SHARE,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_WRITE,
				Constraint: c,
			},
		},
	}
}

// CreateSharePermission is the permission to create shares
func CreateSharePermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "069c08b1-e31f-4799-9ed6-194b310e7244",
		Name:        "Shares.Write",
		DisplayName: "Write share",
		Description: "This permission allows creating shares.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SHARE,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_WRITE,
				Constraint: c,
			},
		},
	}
}

// CreateSpacesPermission is the permission to create spaces
func CreateSpacesPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "79e13b30-3e22-11eb-bc51-0b9f0bad9a58",
		Name:        "Drives.Create",
		DisplayName: "Create Space",
		Description: "This permission allows creating new spaces.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// DeletePersonalSpacesPermission is the permission to delete personal spaces
func DeletePersonalSpacesPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "5de9fe0a-4bc5-4a47-b758-28f370caf169",
		Name:        "Drives.DeletePersonal",
		DisplayName: "Delete All Home Spaces",
		Description: "This permission allows deleting home spaces.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_DELETE,
				Constraint: c,
			},
		},
	}
}

// DeleteProjectSpacesPermission is the permission to delete project spaces
func DeleteProjectSpacesPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "fb60b004-c1fa-4f09-bf87-55ce7d46ac61",
		Name:        "Drives.DeleteProject",
		DisplayName: "Delete AllSpaces",
		Description: "This permission allows deleting all spaces.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_DELETE,
				Constraint: c,
			},
		},
	}
}

// DeleteReadOnlyPublicLinkPasswordPermission is the permission to delete read-only public link passwords
func DeleteReadOnlyPublicLinkPasswordPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "e9a697c5-c67b-40fc-982b-bcf628e9916d",
		Name:        "ReadOnlyPublicLinkPassword.Delete",
		DisplayName: "Delete Read-Only Public link password",
		Description: "This permission permits to opt out of a public link password enforcement.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SHARE,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_WRITE,
				Constraint: c,
			},
		},
	}
}

// DisableEmailNotificationsPermission is the permission to disable email notifications
func DisableEmailNotificationsPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "ad5bb5e5-dc13-4cd3-9304-09a424564ea8",
		Name:        "EmailNotifications.ReadWriteDisabled",
		DisplayName: "Disable Email Notifications",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileDisableNotifications,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEmailSendingIntervalPermission is the permission to set the email sending interval
func ProfileEmailSendingIntervalPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "7dc204ee-799a-43b6-b85d-425fb3b1fa5a",
		Name:        "EmailSendingInterval.ReadWrite",
		DisplayName: "Email Sending Interval",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEmailSendingInterval,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEventShareCreatedPermission is
func ProfileEventShareCreatedPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "8a50540c-1cdd-481f-b85f-44654393c8f0",
		Name:        "Event.ShareCreated.ReadWrite",
		DisplayName: "Event Share Created",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEventShareCreated,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEventShareRemovedPermission is the permission to set the email sending interval
func ProfileEventShareRemovedPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "5ef55465-8e39-4a6c-ba97-1d19f5b07116",
		Name:        "Event.ShareRemoved.ReadWrite",
		DisplayName: "Event Share Removed",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEventShareRemoved,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEventShareExpiredPermission is the permission to set the email sending interval
func ProfileEventShareExpiredPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "7d4f961b-d471-451b-b1fd-ac6a9d59ce88",
		Name:        "Event.ShareExpired.ReadWrite",
		DisplayName: "Event Share Expired",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEventShareExpired,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEventSpaceSharedPermission is the permission to set the email sending interval
func ProfileEventSpaceSharedPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "feb16d2c-614c-4f79-ac37-755a028f5616",
		Name:        "Event.SpaceShared.ReadWrite",
		DisplayName: "Event Space Shared",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEventSpaceShared,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEventSpaceUnsharedPermission is the permission to set the email sending interval
func ProfileEventSpaceUnsharedPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "4f979732-631b-4f27-9be7-a89fb223a6d2",
		Name:        "Event.SpaceUnshared.ReadWrite",
		DisplayName: "Event Space Unshared",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEventSpaceUnshared,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEventSpaceMembershipExpiredPermission is the permission to set the email sending interval
func ProfileEventSpaceMembershipExpiredPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "a3cc45bf-9720-4e08-b403-b9133fe33f0b",
		Name:        "Event.SpaceMembershipExpired.ReadWrite",
		DisplayName: "Event Space Membership Expired",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEventSpaceMembershipExpired,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEventSpaceDisabledPermission is the permission to set the email sending interval
func ProfileEventSpaceDisabledPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "896194c2-5055-4ea3-94a3-0a1419187a00",
		Name:        "Event.SpaceDisabled.ReadWrite",
		DisplayName: "Event Space Disabled",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEventSpaceDisabled,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEventSpaceDeletedPermission is the permission to set the email sending interval
func ProfileEventSpaceDeletedPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "2083c280-b140-4b73-a931-9a4af2931531",
		Name:        "Event.SpaceDeleted.ReadWrite",
		DisplayName: "Event Space Deleted",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEventSpaceDeleted,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ProfileEventPostprocessingStepFinishedPermission is the permission to set the email sending interval
func ProfileEventPostprocessingStepFinishedPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "27ba8e97-0bdf-4b18-97d4-df44c9568cda",
		Name:        "Event.PostprocessingStepFinished.ReadWrite",
		DisplayName: "Event Postprocessing Step Finished",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileEventPostprocessingStepFinished,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// GroupManagementPermission is the permission to manage groups
func GroupManagementPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "522adfbe-5908-45b4-b135-41979de73245",
		Name:        "Groups.ReadWrite",
		DisplayName: "Group Management",
		Description: "This permission gives full access to everything that is related to group management.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_GROUP,
			Id:   "all",
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// LanguageManagementPermission is the permission to manage the language
func LanguageManagementPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "7d81f103-0488-4853-bce5-98dcce36d649",
		Name:        "Language.ReadWrite",
		DisplayName: "Permission to read and set the language",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SETTING,
			Id:   SettingUUIDProfileLanguage,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// ListFavoritesPermission is the permission to list favorites
func ListFavoritesPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "4ebaa725-bfaa-43c5-9817-78bc9994bde4",
		Name:        "Favorites.List",
		DisplayName: "List Favorites",
		Description: "This permission allows listing favorites.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READ,
				Constraint: c,
			},
		},
	}
}

// ListSpacesPermission is the permission to list spaces
func ListSpacesPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "016f6ddd-9501-4a0a-8ebe-64a20ee8ec82",
		Name:        "Drives.List",
		DisplayName: "List All Spaces",
		Description: "This permission allows listing all spaces.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READ,
				Constraint: c,
			},
		},
	}
}

// ManageSpacePropertiesPermission is the permission to manage space properties
func ManageSpacePropertiesPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "b44b4054-31a2-42b8-bb71-968b15cfbd4f",
		Name:        "Drives.ReadWrite",
		DisplayName: "Manage space properties",
		Description: "This permission allows managing space properties such as name and description.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// RoleManagementPermission is the permission to manage roles
func RoleManagementPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "a53e601e-571f-4f86-8fec-d4576ef49c62",
		Name:        "Roles.ReadWrite",
		DisplayName: "Role Management",
		Description: "This permission gives full access to everything that is related to role management.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_USER,
			Id:   "all",
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// SelfManagementPermission is the permission to manage itself
func SelfManagementPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "e03070e9-4362-4cc6-a872-1c7cb2eb2b8e",
		Name:        "Self.ReadWrite",
		DisplayName: "Self Management",
		Description: "This permission gives access to self management.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_USER,
			Id:   "me",
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// SetPersonalSpaceQuotaPermission is the permission to set the quota for personal spaces
func SetPersonalSpaceQuotaPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "4e6f9709-f9e7-44f1-95d4-b762d27b7896",
		Name:        "Drives.ReadWritePersonalQuota",
		DisplayName: "Set Personal Space Quota",
		Description: "This permission allows managing personal space quotas.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// SetProjectSpaceQuotaPermission is the permission to set the quota for project spaces
func SetProjectSpaceQuotaPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "977f0ae6-0da2-4856-93f3-22e0a8482489",
		Name:        "Drives.ReadWriteProjectQuota",
		DisplayName: "Set Project Space Quota",
		Description: "This permission allows managing project space quotas.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// SettingsManagementPermission is the permission to manage settings
func SettingsManagementPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "3d58f441-4a05-42f8-9411-ef5874528ae1",
		Name:        "Settings.ReadWrite",
		DisplayName: "Settings Management",
		Description: "This permission gives full access to everything that is related to settings management.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_USER,
			Id:   "all",
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// SpaceAbilityPermission is the permission to enable or disable spaces
func SpaceAbilityPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "cf3faa8c-50d9-4f84-9650-ff9faf21aa9d",
		Name:        "Drives.ReadWriteEnabled",
		DisplayName: "Space ability",
		Description: "This permission allows enabling and disabling spaces.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}

// WriteFavoritesPermission is the permission to mark/unmark files as favorites
func WriteFavoritesPermission(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "a54778fd-1c45-47f0-892d-655caf5236f2",
		Name:        "Favorites.Write",
		DisplayName: "Write Favorites",
		Description: "This permission allows marking files as favorites.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_FILE,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_WRITE,
				Constraint: c,
			},
		},
	}
}

// WebOfficManagementPermssion is the permission to mark/unmark files as favorites
func WebOfficeManagementPermssion(c settingsmsg.Permission_Constraint) *settingsmsg.Setting {
	return &settingsmsg.Setting{
		Id:          "27a29046-a816-424f-bd71-2ffb9029162f",
		Name:        "WebOffice.Manage",
		DisplayName: "Manage WebOffice",
		Description: "This permission gives access to the admin features in the WebOffice suite.",
		Resource: &settingsmsg.Resource{
			Type: settingsmsg.Resource_TYPE_SYSTEM,
		},
		Value: &settingsmsg.Setting_PermissionValue{
			PermissionValue: &settingsmsg.Permission{
				Operation:  settingsmsg.Permission_OPERATION_READWRITE,
				Constraint: c,
			},
		},
	}
}
