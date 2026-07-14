package imapclient

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Move sends a MOVE command.
//
// If the server doesn't support IMAP4rev2 nor the MOVE extension, a fallback
// with COPY + STORE + EXPUNGE commands is used.
func (c *Client) Move(numSet imap.NumSet, mailbox string) *MoveCommand {
	// If the server doesn't support MOVE, fallback to [UID] COPY,
	// [UID] STORE +FLAGS.SILENT \Deleted and [UID] EXPUNGE
	cmdName := "MOVE"
	if !c.Caps().Has(imap.CapMove) {
		cmdName = "COPY"
	}

	cmd := &MoveCommand{}
	enc := c.beginCommand(uidCmdName(cmdName, imapwire.NumSetKind(numSet)), cmd)
	enc.SP().NumSet(numSet).SP().Mailbox(mailbox)
	enc.end()

	if cmdName == "COPY" {
		cmd.store = c.Store(numSet, &imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Silent: true,
			Flags:  []imap.Flag{imap.FlagDeleted},
		}, nil)
		if uidSet, ok := numSet.(imap.UIDSet); ok && c.Caps().Has(imap.CapUIDPlus) {
			cmd.expunge = c.UIDExpunge(uidSet)
		} else {
			cmd.expunge = c.Expunge()
		}
	}

	return cmd
}

// MoveCommand is a MOVE command.
type MoveCommand struct {
	commandBase
	data MoveData

	// Fallback
	store   *FetchCommand
	expunge *ExpungeCommand
}

func (cmd *MoveCommand) Wait() (*MoveData, error) {
	if err := cmd.wait(); err != nil {
		return nil, err
	}
	if cmd.store != nil {
		if err := cmd.store.Close(); err != nil {
			return nil, err
		}
	}
	if cmd.expunge != nil {
		if err := cmd.expunge.Close(); err != nil {
			return nil, err
		}
	}
	return &cmd.data, nil
}

// MoveData contains the data returned by a MOVE command.
type MoveData struct {
	// requires UIDPLUS or IMAP4rev2
	UIDValidity uint32
	SourceUIDs  imap.NumSet
	DestUIDs    imap.NumSet
}
