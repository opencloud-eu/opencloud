package service

import "github.com/opencloud-eu/opencloud/pkg/l10n"

// the available templates
var (
	VirusFound = NotificationTemplate{
		Subject: l10n.Template("Virus found"),
		Message: l10n.Template("Virus found in {resource}. Upload not possible. Virus: {virus}"),
	}

	PoliciesEnforced = NotificationTemplate{
		Subject: l10n.Template("Policies enforced"),
		Message: l10n.Template("The file {resource} was deleted because it violates the restrictions of this cloud. This could be due to an unsupported file type, potentially harmful or prohibited content, or file size limits."),
	}

	SpaceShared = NotificationTemplate{
		Subject: l10n.Template("Space shared"),
		Message: l10n.Template("{user} added you to Space {space}"),
	}

	SpaceUnshared = NotificationTemplate{
		Subject: l10n.Template("Removed from Space"),
		Message: l10n.Template("{user} removed you from Space {space}"),
	}

	SpaceDisabled = NotificationTemplate{
		Subject: l10n.Template("Space disabled"),
		Message: l10n.Template("{user} disabled Space {space}"),
	}

	SpaceDeleted = NotificationTemplate{
		Subject: l10n.Template("Space deleted"),
		Message: l10n.Template("{user} deleted Space {space}"),
	}

	SpaceMembershipExpired = NotificationTemplate{
		Subject: l10n.Template("Membership expired"),
		Message: l10n.Template("Access to Space {space} lost"),
	}

	ShareCreated = NotificationTemplate{
		Subject: l10n.Template("Resource shared"),
		Message: l10n.Template("{user} shared {resource} with you"),
	}

	ShareRemoved = NotificationTemplate{
		Subject: l10n.Template("Resource unshared"),
		Message: l10n.Template("{user} unshared {resource} with you"),
	}

	ShareExpired = NotificationTemplate{
		Subject: l10n.Template("Share expired"),
		Message: l10n.Template("Access to {resource} expired"),
	}

	PlatformDeprovision = NotificationTemplate{
		Subject: l10n.Template("Instance will be shut down and deprovisioned"),
		Message: l10n.Template("Attention! The instance will be shut down and deprovisioned on {date}. Download all your data before that date as no access past that date is possible."),
	}
)

// holds the information to turn the raw template into a parseable go template
var _placeholders = map[string]string{
	"{user}":     "{{ .username }}",
	"{space}":    "{{ .spacename }}",
	"{resource}": "{{ .resourcename }}",
	"{virus}":    "{{ .virusdescription }}",
	"{date}":     "{{ .date }}",
}

// NotificationTemplate is the data structure for the notifications
type NotificationTemplate struct {
	Subject string
	Message string
}
