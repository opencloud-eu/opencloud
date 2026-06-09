package defaults

import "github.com/opencloud-eu/opencloud/pkg/l10n"

// Translatable configuration options
var (
	// name of the notification option 'Share Received'
	TemplateShareCreated = l10n.Template("Share Received")
	// description of the notification option 'Share Received'
	TemplateShareCreatedDescription = l10n.Template("Notify when I have received a share")
	// name of the notification option 'Share Removed'
	TemplateShareRemoved = l10n.Template("Share Removed")
	// description of the notification option 'Share Removed'
	TemplateShareRemovedDescription = l10n.Template("Notify when a received share has been removed")
	// name of the notification option 'Resource Mention'
	TemplateResourceMention = l10n.Template("Resource Mention")
	// description of the notification option 'Resource Mention'
	TemplateResourceMentionDescription = l10n.Template("Notify on resource mentions")
	// name of the notification option 'Share Expired'
	TemplateShareExpired = l10n.Template("Share Expired")
	// description of the notification option 'Share Expired'
	TemplateShareExpiredDescription = l10n.Template("Notify when a received share has expired")
	// name of the notification option 'Space Shared'
	TemplateSpaceShared = l10n.Template("Added as space member")
	// description of the notification option 'Space Shared'
	TemplateSpaceSharedDescription = l10n.Template("Notify when I have been added as a member to a space")
	// name of the notification option 'Space Unshared'
	TemplateSpaceUnshared = l10n.Template("Removed as space member")
	// description of the notification option 'Space Unshared'
	TemplateSpaceUnsharedDescription = l10n.Template("Notify when I have been removed as member from a space")
	// name of the notification option 'Space Membership Expired'
	TemplateSpaceMembershipExpired = l10n.Template("Space membership expired")
	// description of the notification option 'Space Membership Expired'
	TemplateSpaceMembershipExpiredDescription = l10n.Template("Notify when a space membership has expired")
	// name of the notification option 'Space Disabled'
	TemplateSpaceDisabled = l10n.Template("Space disabled")
	// description of the notification option 'Space Disabled'
	TemplateSpaceDisabledDescription = l10n.Template("Notify when a space I am member of has been disabled")
	// name of the notification option 'Space Deleted'
	TemplateSpaceDeleted = l10n.Template("Space deleted")
	// description of the notification option 'Space Deleted'
	TemplateSpaceDeletedDescription = l10n.Template("Notify when a space I am member of has been deleted")
	// name of the notification option 'File Rejected'
	TemplateFileRejected = l10n.Template("File rejected")
	// description of the notification option 'File Rejected'
	TemplateFileRejectedDescription = l10n.Template("Notify when a file I uploaded was rejected because of a virus infection or policy violation")
	// name of the notification option 'Email Interval'
	TemplateEmailSendingInterval = l10n.Template("Email sending interval")
	// description of the notification option 'Email Interval'
	TemplateEmailSendingIntervalDescription = l10n.Template("Selected value:")
	// translation for the 'instant' email interval option
	TemplateIntervalInstant = l10n.Template("Instant")
	// translation for the 'daily' email interval option
	TemplateIntervalDaily = l10n.Template("Daily")
	// translation for the 'weekly' email interval option
	TemplateIntervalWeekly = l10n.Template("Weekly")
	// translation for the 'never' email interval option
	TemplateIntervalNever = l10n.Template("Never")
)
