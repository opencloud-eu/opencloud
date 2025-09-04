package enmime

import "net/smtp"

// Sender provides a method for enmime to send an email.
type Sender interface {
	// Sends the provided msg to the specified recipients, providing the specified reverse-path to
	// the mail server to use for delivery error reporting.
	//
	// The message headers should usually include fields such as "From", "To", "Subject", and "Cc".
	// Sending "Bcc" messages is accomplished by including an email address in the recipients
	// parameter but not including it in the message headers.
	Send(reversePath string, recipients []string, msg []byte) error
}

// SMTPSender is a Sender backed by Go's built-in net/smtp.SendMail function.
type SMTPSender struct {
	addr string
	auth smtp.Auth
}

var _ Sender = &SMTPSender{}

// NewSMTP creates a new SMTPSender, which uses net/smtp.SendMail, and accepts the same
// authentication parameters.  If no authentication is required, `auth` may be nil.
func NewSMTP(addr string, auth smtp.Auth) *SMTPSender {
	return &SMTPSender{addr, auth}
}

// Send a message using net/smtp.SendMail.
func (s *SMTPSender) Send(reversePath string, recipients []string, msg []byte) error {
	return smtp.SendMail(s.addr, s.auth, reversePath, recipients, msg)
}
