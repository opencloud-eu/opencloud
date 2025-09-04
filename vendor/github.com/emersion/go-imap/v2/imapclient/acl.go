package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// MyRights sends a MYRIGHTS command.
//
// This command requires support for the ACL extension.
func (c *Client) MyRights(mailbox string) *MyRightsCommand {
	cmd := &MyRightsCommand{}
	enc := c.beginCommand("MYRIGHTS", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// SetACL sends a SETACL command.
//
// This command requires support for the ACL extension.
func (c *Client) SetACL(mailbox string, ri imap.RightsIdentifier, rm imap.RightModification, rs imap.RightSet) *SetACLCommand {
	cmd := &SetACLCommand{}
	enc := c.beginCommand("SETACL", cmd)
	enc.SP().Mailbox(mailbox).SP().String(string(ri)).SP()
	enc.String(internal.FormatRights(rm, rs))
	enc.end()
	return cmd
}

// SetACLCommand is a SETACL command.
type SetACLCommand struct {
	commandBase
}

func (cmd *SetACLCommand) Wait() error {
	return cmd.wait()
}

// GetACL sends a GETACL command.
//
// This command requires support for the ACL extension.
func (c *Client) GetACL(mailbox string) *GetACLCommand {
	cmd := &GetACLCommand{}
	enc := c.beginCommand("GETACL", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// GetACLCommand is a GETACL command.
type GetACLCommand struct {
	commandBase
	data GetACLData
}

func (cmd *GetACLCommand) Wait() (*GetACLData, error) {
	return &cmd.data, cmd.wait()
}

func (c *Client) handleMyRights() error {
	data, err := readMyRights(c.dec)
	if err != nil {
		return fmt.Errorf("in myrights-response: %v", err)
	}
	if cmd := findPendingCmdByType[*MyRightsCommand](c); cmd != nil {
		cmd.data = *data
	}
	return nil
}

func (c *Client) handleGetACL() error {
	data, err := readGetACL(c.dec)
	if err != nil {
		return fmt.Errorf("in getacl-response: %v", err)
	}
	if cmd := findPendingCmdByType[*GetACLCommand](c); cmd != nil {
		cmd.data = *data
	}
	return nil
}

// MyRightsCommand is a MYRIGHTS command.
type MyRightsCommand struct {
	commandBase
	data MyRightsData
}

func (cmd *MyRightsCommand) Wait() (*MyRightsData, error) {
	return &cmd.data, cmd.wait()
}

// MyRightsData is the data returned by the MYRIGHTS command.
type MyRightsData struct {
	Mailbox string
	Rights  imap.RightSet
}

func readMyRights(dec *imapwire.Decoder) (*MyRightsData, error) {
	var (
		rights string
		data   MyRightsData
	)
	if !dec.ExpectMailbox(&data.Mailbox) || !dec.ExpectSP() || !dec.ExpectAString(&rights) {
		return nil, dec.Err()
	}

	data.Rights = imap.RightSet(rights)
	return &data, nil
}

// GetACLData is the data returned by the GETACL command.
type GetACLData struct {
	Mailbox string
	Rights  map[imap.RightsIdentifier]imap.RightSet
}

func readGetACL(dec *imapwire.Decoder) (*GetACLData, error) {
	data := &GetACLData{Rights: make(map[imap.RightsIdentifier]imap.RightSet)}

	if !dec.ExpectMailbox(&data.Mailbox) {
		return nil, dec.Err()
	}

	for dec.SP() {
		var rsStr, riStr string
		if !dec.ExpectAString(&riStr) || !dec.ExpectSP() || !dec.ExpectAString(&rsStr) {
			return nil, dec.Err()
		}

		data.Rights[imap.RightsIdentifier(riStr)] = imap.RightSet(rsStr)
	}

	return data, nil
}
