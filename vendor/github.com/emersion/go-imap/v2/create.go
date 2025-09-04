package imap

// CreateOptions contains options for the CREATE command.
type CreateOptions struct {
	SpecialUse []MailboxAttr // requires CREATE-SPECIAL-USE
}
