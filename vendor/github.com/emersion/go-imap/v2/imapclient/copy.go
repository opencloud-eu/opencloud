package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Copy sends a COPY command.
func (c *Client) Copy(numSet imap.NumSet, mailbox string) *CopyCommand {
	cmd := &CopyCommand{}
	enc := c.beginCommand(uidCmdName("COPY", imapwire.NumSetKind(numSet)), cmd)
	enc.SP().NumSet(numSet).SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// CopyCommand is a COPY command.
type CopyCommand struct {
	commandBase
	data imap.CopyData
}

func (cmd *CopyCommand) Wait() (*imap.CopyData, error) {
	return &cmd.data, cmd.wait()
}

func readRespCodeCopyUID(dec *imapwire.Decoder) (uidValidity uint32, srcUIDs, dstUIDs imap.UIDSet, err error) {
	if !dec.ExpectNumber(&uidValidity) || !dec.ExpectSP() || !dec.ExpectUIDSet(&srcUIDs) || !dec.ExpectSP() || !dec.ExpectUIDSet(&dstUIDs) {
		return 0, nil, nil, dec.Err()
	}
	if srcUIDs.Dynamic() || dstUIDs.Dynamic() {
		return 0, nil, nil, fmt.Errorf("imapclient: server returned dynamic number set in COPYUID response")
	}
	return uidValidity, srcUIDs, dstUIDs, nil
}
