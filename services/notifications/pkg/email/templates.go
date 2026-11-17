package email

import "github.com/opencloud-eu/opencloud/pkg/l10n"

const (
	_textTemplate = "templates/text/email.text.tmpl"
	// _htmlLayout is the parent of every html email, the children fill its "content" block
	_htmlLayout   = "templates/html/layout.html.tmpl"
	_htmlTemplate = "templates/html/message.html.tmpl"
	_htmlDigest   = "templates/html/digest.html.tmpl"
	// _htmlLegacyTemplate is the single html template used before the layout was split,
	// custom template folders that only contain this file keep working with it
	_htmlLegacyTemplate = "templates/html/email.html.tmpl"
)

// the available templates
var (
	// Shares
	ShareCreated = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// ShareCreated email template, Subject field (resolves directly)
		Subject: l10n.Template(`{ShareSharer} shared '{ShareFolder}' with you`),
		// ShareCreated email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hello {ShareGrantee}`),
		// ShareCreated email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`{ShareSharer} has shared "{ShareFolder}" with you.`),
		// ShareCreated email template, resolves via {{ .CallToAction }}
		CallToAction: l10n.Template(`Click here to view it: {ShareLink}`),
		// ShareCreated email template, label of the button in the html email
		ActionLabel:   l10n.Template(`View shares`),
		actionLink:    `{ShareLink}`,
		digestLinkVar: "ShareFolder",
	}

	ShareRemoved = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// ShareRemoved email template, Subject field (resolves directly)
		Subject: l10n.Template(`{ShareSharer} unshared '{ShareFolder}' with you`),
		// ShareRemoved email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hello {ShareGrantee},`),
		// ShareRemoved email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`{ShareSharer} has unshared '{ShareFolder}' with you.

Even though this share has been revoked you still might have access through other shares and/or space memberships.`),
		bodyLayout: []paragraphRole{paragraphText, paragraphHint},
	}

	ShareExpired = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// ShareExpired email template, Subject field (resolves directly)
		Subject: l10n.Template(`Share to '{ShareFolder}' expired at {ExpiredAt}`),
		// ShareExpired email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hello {ShareGrantee},`),
		// ShareExpired email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`Your share to {ShareFolder} has expired at {ExpiredAt}

Even though this share has been revoked you still might have access through other shares and/or space memberships.`),
		bodyLayout: []paragraphRole{paragraphText, paragraphHint},
	}

	// Guest link templates
	GuestLinkShareCreated = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// GuestLinkShareCreated email template, Subject field (resolves directly)
		Subject: l10n.Template(`{ShareSharer} shared '{ShareFolder}' with you`),
		// GuestLinkShareCreated email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hello,`),
		// GuestLinkShareCreated email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`{ShareSharer} has shared "{ShareFolder}" with you.

The link below is personal and can only be used once. Please do not forward it.

The link is only valid for 30 minutes.`),
		// GuestLinkShareCreated email template, resolves via {{ .CallToAction }}
		CallToAction: l10n.Template(`Click here to view it: {ShareLink}`),
		// GuestLinkShareCreated email template, label of the button in the html email
		ActionLabel: l10n.Template(`Open link`),
		actionLink:  `{ShareLink}`,
		bodyLayout:  []paragraphRole{paragraphText, paragraphText, paragraphNote},
	}

	GuestLinkShareRenewed = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// GuestLinkShareRenewed email template, Subject field (resolves directly)
		Subject: l10n.Template(`Your new link to '{ShareFolder}'`),
		// GuestLinkShareRenewed email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hello,`),
		// GuestLinkShareRenewed email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`A new link to "{ShareFolder}", shared with you by {ShareSharer}, has been requested.

Your PIN: {SharePin}

Open the link below or enter the PIN on the page where you requested the new link. The link and the PIN are personal and can only be used once. Please do not forward them.

The link and the PIN are only valid for 30 minutes.`),
		// GuestLinkShareRenewed email template, resolves via {{ .CallToAction }}
		CallToAction: l10n.Template(`Click here to view it: {ShareLink}`),
		// GuestLinkShareRenewed email template, label of the button in the html email
		ActionLabel: l10n.Template(`Open link`),
		actionLink:  `{ShareLink}`,
		bodyLayout:  []paragraphRole{paragraphText, paragraphCode, paragraphText, paragraphNote},
	}

	GuestLinkShareRenewedWithoutPin = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// GuestLinkShareRenewedWithoutPin email template, Subject field (resolves directly)
		Subject: l10n.Template(`Your new link to '{ShareFolder}'`),
		// GuestLinkShareRenewedWithoutPin email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hello,`),
		// GuestLinkShareRenewedWithoutPin email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`A new link to "{ShareFolder}", shared with you by {ShareSharer}, has been requested.

The link below is personal and can only be used once. Please do not forward it.

The link is only valid for 30 minutes.`),
		// GuestLinkShareRenewedWithoutPin email template, resolves via {{ .CallToAction }}
		CallToAction: l10n.Template(`Click here to view it: {ShareLink}`),
		// GuestLinkShareRenewedWithoutPin email template, label of the button in the html email
		ActionLabel: l10n.Template(`Open link`),
		actionLink:  `{ShareLink}`,
		bodyLayout:  []paragraphRole{paragraphText, paragraphText, paragraphNote},
	}

	// Spaces templates
	SharedSpace = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// SharedSpace email template, Subject field (resolves directly)
		Subject: l10n.Template("{SpaceSharer} invited you to join {SpaceName}"),
		// SharedSpace email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hello {SpaceGrantee},`),
		// SharedSpace email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`{SpaceSharer} has invited you to join "{SpaceName}".`),
		// SharedSpace email template, resolves via {{ .CallToAction }}
		CallToAction: l10n.Template(`Click here to view it: {ShareLink}`),
		// SharedSpace email template, label of the button in the html email
		ActionLabel:   l10n.Template(`Open space`),
		actionLink:    `{ShareLink}`,
		digestLinkVar: "SpaceName",
	}

	UnsharedSpace = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// UnsharedSpace email template, Subject field (resolves directly)
		Subject: l10n.Template(`{SpaceSharer} removed you from {SpaceName}`),
		// UnsharedSpace email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hello {SpaceGrantee},`),
		// UnsharedSpace email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`{SpaceSharer} has removed you from "{SpaceName}".

You might still have access through your other groups or direct membership.`),
		// UnsharedSpace email template, resolves via {{ .CallToAction }}
		CallToAction: l10n.Template(`Click here to check it: {ShareLink}`),
		// UnsharedSpace email template, label of the button in the html email
		ActionLabel:   l10n.Template(`Check access`),
		actionLink:    `{ShareLink}`,
		bodyLayout:    []paragraphRole{paragraphText, paragraphHint},
		digestLinkVar: "SpaceName",
	}

	MembershipExpired = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// MembershipExpired email template, Subject field (resolves directly)
		Subject: l10n.Template(`Membership of '{SpaceName}' expired at {ExpiredAt}`),
		// MembershipExpired email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hello {SpaceGrantee},`),
		// MembershipExpired email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`Your membership of space {SpaceName} has expired at {ExpiredAt}

Even though this membership has expired you still might have access through other shares and/or space memberships`),
		bodyLayout: []paragraphRole{paragraphText, paragraphHint},
	}

	ScienceMeshInviteTokenGenerated = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// ScienceMeshInviteTokenGenerated email template, Subject field (resolves directly)
		Subject: l10n.Template(`ScienceMesh: {InitiatorName} wants to collaborate with you`),
		// ScienceMeshInviteTokenGenerated email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hi,`),
		// ScienceMeshInviteTokenGenerated email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`{ShareSharer} ({ShareSharerMail}) wants to start sharing collaboration resources with you.
To accept the invite, please visit the following URL:
{ShareLink}

Alternatively, you can visit your federation settings and use the following details:
  Token: {Token}
  ProviderDomain: {ProviderDomain}`),
	}

	ScienceMeshInviteTokenGeneratedWithoutShareLink = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		// ScienceMeshInviteTokenGeneratedWithoutShareLink email template, Subject field (resolves directly)
		Subject: l10n.Template(`ScienceMesh: {InitiatorName} wants to collaborate with you`),
		// ScienceMeshInviteTokenGeneratedWithoutShareLink email template, resolves via {{ .Greeting }}
		Greeting: l10n.Template(`Hi,`),
		// ScienceMeshInviteTokenGeneratedWithoutShareLink email template, resolves via {{ .MessageBody }}
		MessageBody: l10n.Template(`{ShareSharer} ({ShareSharerMail}) wants to start sharing collaboration resources with you.
Please visit your federation settings and use the following details:
  Token: {Token}
  ProviderDomain: {ProviderDomain}`),
	}

	GroupedDaily = GroupedMessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlDigest,
		// GroupedDaily email template, Subject field (resolves directly)
		Subject: l10n.Template(`Your daily notifications`),
		// GroupedDaily email template, resolves via {{ .Greeting }}
		Greeting:    l10n.Template(`Hi {DisplayName},`),
		MessageBody: "", // is generated using the GroupedTemplates
	}

	GroupedWeekly = GroupedMessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlDigest,
		// GroupedWeekly email template, Subject field (resolves directly)
		Subject: l10n.Template(`Your weekly notifications`),
		// GroupedWeekly email template, resolves via {{ .Greeting }}
		Greeting:    l10n.Template(`Hi {DisplayName},`),
		MessageBody: "", // is generated using the GroupedTemplates
	}

	// fallbackLinkLabel introduces the plain link below the button of the html email
	_fallbackLinkLabel = l10n.Template(`Or open this link:`)

	Mention = MessageTemplate{
		textTemplate: _textTemplate,
		htmlTemplate: _htmlTemplate,
		Subject:      l10n.Template(`You were mentioned in '{ResourceName}'`),
		Greeting:     l10n.Template(`Hello {RecipientName},`),
		MessageBody:  l10n.Template(`{AuthorName} mentioned you in "{ResourceName}".`),
		CallToAction: l10n.Template(`You can view the mention here: {ResourceLink}`),
		// Mention email template, label of the button in the html email
		ActionLabel: l10n.Template(`View mention`),
		actionLink:  `{ResourceLink}`,
	}
)

// holds the information to turn the raw template into a parseable go template
var _placeholders = map[string]string{
	"{ShareSharer}":     "{{ .ShareSharer }}",
	"{ShareFolder}":     "{{ .ShareFolder }}",
	"{ShareGrantee}":    "{{ .ShareGrantee }}",
	"{ShareLink}":       "{{ .ShareLink }}",
	"{SharePin}":        "{{ .SharePin }}",
	"{SpaceName}":       "{{ .SpaceName }}",
	"{SpaceGrantee}":    "{{ .SpaceGrantee }}",
	"{SpaceSharer}":     "{{ .SpaceSharer }}",
	"{ExpiredAt}":       "{{ .ExpiredAt }}",
	"{ShareSharerMail}": "{{ .ShareSharerMail }}",
	"{InitiatorName}":   "{{ .InitiatorName }}",
	"{ProviderDomain}":  "{{ .ProviderDomain }}",
	"{Token}":           "{{ .Token }}",
	"{DisplayName}":     "{{ .DisplayName }}",
	"{AuthorName}":      "{{ .AuthorName }}",
	"{RecipientName}":   "{{ .RecipientName }}",
	"{ResourceName}":    "{{ .ResourceName }}",
	"{ResourceLink}":    "{{ .ResourceLink }}",
}

// paragraphRole tells the html template how to show a paragraph of the message body
type paragraphRole int

const (
	// paragraphText is a regular paragraph
	paragraphText paragraphRole = iota
	// paragraphHint is a secondary paragraph shown in a muted color
	paragraphHint
	// paragraphCode holds a "label: value" pair, the value is shown large in a box
	paragraphCode
	// paragraphNote is shown in small print below the button
	paragraphNote
)

// MessageTemplate is the data structure for the email
type MessageTemplate struct {
	// textTemplate represent the path to text plain .tmpl file
	textTemplate string
	// htmlTemplate represent the path to html .tmpl file
	htmlTemplate string
	// The fields below represent the placeholders for the translatable templates
	Subject      string
	Greeting     string
	MessageBody  string
	CallToAction string
	// ActionLabel is the label of the button in the html email
	ActionLabel string
	// actionLink is the placeholder of the link the button opens
	actionLink string
	// digestLinkVar names the variable that is shown as the link to the resource in the
	// timeline of a grouped email
	digestLinkVar string
	// bodyLayout gives the role of each paragraph of the message body in the html email,
	// paragraphs are separated by an empty line. Without a layout, or if a translation has a
	// different number of paragraphs, all paragraphs are regular text.
	bodyLayout []paragraphRole
}

// GroupedMessageTemplate is the data structure for the email
type GroupedMessageTemplate struct {
	// textTemplate represent the path to text plain .tmpl file
	textTemplate string
	// htmlTemplate represent the path to html .tmpl file
	htmlTemplate string
	// The fields below represent the placeholders for the translatable templates
	Subject     string
	Greeting    string
	MessageBody string
}
