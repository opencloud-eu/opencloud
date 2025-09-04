package imapclient

import (
	"github.com/emersion/go-imap/v2"
)

// Create sends a CREATE command.
//
// A nil options pointer is equivalent to a zero options value.
func (c *Client) Create(mailbox string, options *imap.CreateOptions) *Command {
	cmd := &Command{}
	enc := c.beginCommand("CREATE", cmd)
	enc.SP().Mailbox(mailbox)
	if options != nil && len(options.SpecialUse) > 0 {
		enc.SP().Special('(').Atom("USE").SP().List(len(options.SpecialUse), func(i int) {
			enc.MailboxAttr(options.SpecialUse[i])
		}).Special(')')
	}
	enc.end()
	return cmd
}
