package invitations

import libregraph "github.com/opencloud-eu/libre-graph-api-go"

// Invitation represents an invitation as per https://learn.microsoft.com/en-us/graph/api/resources/invitation?view=graph-rest-1.0
type Invitation struct {
	// The display name of the user being invited.
	InvitedUserDisplayName string `json:"invitedUserDisplayName,omitempty"`

	// The email address of the user being invited. Required.
	InvitedUserEmailAddress string `json:"invitedUserEmailAddress"`

	// Additional configuration for the message being sent to the
	// invited user, including customizing message text, language
	// and cc recipient list.
	InvitedUserMessageInfo *InvitedUserMessageInfo `json:"invitedUserMessageInfo,omitempty"`
	// The userType of the user being invited. By default, this is
	// `Guest``. You can invite as `Member`` if you are a company
	// administrator.
	InvitedUserType string `json:"invitedUserType,omitempty"`
	// The URL the user should be redirected to once the invitation
	// is redeemed. Required.
	InviteRedirectUrl string `json:"inviteRedirectUrl"`
	// The URL the user can use to redeem their invitation. Read-only.
	InviteRedeemUrl string `json:"inviteRedeemUrl,omitempty"`
	// Reset the user's redemption status and reinvite a user while
	// retaining their user identifier, group memberships, and app
	// assignments. This property allows you to enable a user to
	// sign-in using a different email address from the one in the
	// previous invitation.
	ResetRedemption string `json:"resetRedemption,omitempty"`
	// Indicates whether an email should be sent to the user being
	// invited. The default is false.
	SendInvitationMessage bool `json:"sendInvitationMessage,omitempty"`
	// The status of the invitation. Possible values are:
	// `PendingAcceptance`, `Completed`, `InProgress`, and `Error`.
	Status string `json:"status,omitempty"`

	// Relations

	// The user created as part of the invitation creation. Read-Only
	InvitedUser *libregraph.User `json:"invitedUser,omitempty"`
}

type InvitedUserMessageInfo struct {
	// Additional recipients the invitation message should be sent
	// to. Currently only 1 additional recipient is supported.
	CcRecipients []Recipient `json:"ccRecipients"`

	// Customized message body you want to send if you don't want
	// the default message.
	CustomizedMessageBody string `json:"customizedMessageBody"`

	// The language you want to send the default message in. If the
	// customizedMessageBody is specified, this property is ignored,
	// and the message is sent using the customizedMessageBody. The
	// language format should be in ISO 639. The default is en-US.
	MessageLanguage string `json:"messageLanguage"`
}
type Recipient struct {
	// The recipient's email address.
	EmailAddress EmailAddress `json:"emailAddress"`
}
type EmailAddress struct {
	// The email address of the person or entity.
	Aaddress string `json:"address"`

	// The display name of the person or entity.
	Name string `json:"name"`
}
