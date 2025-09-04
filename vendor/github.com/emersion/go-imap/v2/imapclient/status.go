package imapclient

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func statusItems(options *imap.StatusOptions) []string {
	m := map[string]bool{
		"MESSAGES":        options.NumMessages,
		"UIDNEXT":         options.UIDNext,
		"UIDVALIDITY":     options.UIDValidity,
		"UNSEEN":          options.NumUnseen,
		"DELETED":         options.NumDeleted,
		"SIZE":            options.Size,
		"APPENDLIMIT":     options.AppendLimit,
		"DELETED-STORAGE": options.DeletedStorage,
		"HIGHESTMODSEQ":   options.HighestModSeq,
	}

	var l []string
	for k, req := range m {
		if req {
			l = append(l, k)
		}
	}
	return l
}

// Status sends a STATUS command.
//
// A nil options pointer is equivalent to a zero options value.
func (c *Client) Status(mailbox string, options *imap.StatusOptions) *StatusCommand {
	if options == nil {
		options = new(imap.StatusOptions)
	}

	cmd := &StatusCommand{mailbox: mailbox}
	enc := c.beginCommand("STATUS", cmd)
	enc.SP().Mailbox(mailbox).SP()
	items := statusItems(options)
	enc.List(len(items), func(i int) {
		enc.Atom(items[i])
	})
	enc.end()
	return cmd
}

func (c *Client) handleStatus() error {
	data, err := readStatus(c.dec)
	if err != nil {
		return fmt.Errorf("in status: %v", err)
	}

	cmd := c.findPendingCmdFunc(func(cmd command) bool {
		switch cmd := cmd.(type) {
		case *StatusCommand:
			return cmd.mailbox == data.Mailbox
		case *ListCommand:
			return cmd.returnStatus && cmd.pendingData != nil && cmd.pendingData.Mailbox == data.Mailbox
		default:
			return false
		}
	})
	switch cmd := cmd.(type) {
	case *StatusCommand:
		cmd.data = *data
	case *ListCommand:
		cmd.pendingData.Status = data
		cmd.mailboxes <- cmd.pendingData
		cmd.pendingData = nil
	}

	return nil
}

// StatusCommand is a STATUS command.
type StatusCommand struct {
	commandBase
	mailbox string
	data    imap.StatusData
}

func (cmd *StatusCommand) Wait() (*imap.StatusData, error) {
	return &cmd.data, cmd.wait()
}

func readStatus(dec *imapwire.Decoder) (*imap.StatusData, error) {
	var data imap.StatusData

	if !dec.ExpectMailbox(&data.Mailbox) || !dec.ExpectSP() {
		return nil, dec.Err()
	}

	err := dec.ExpectList(func() error {
		if err := readStatusAttVal(dec, &data); err != nil {
			return fmt.Errorf("in status-att-val: %v", dec.Err())
		}
		return nil
	})
	return &data, err
}

func readStatusAttVal(dec *imapwire.Decoder, data *imap.StatusData) error {
	var name string
	if !dec.ExpectAtom(&name) || !dec.ExpectSP() {
		return dec.Err()
	}

	var ok bool
	switch strings.ToUpper(name) {
	case "MESSAGES":
		var num uint32
		ok = dec.ExpectNumber(&num)
		data.NumMessages = &num
	case "UIDNEXT":
		var uidNext imap.UID
		ok = dec.ExpectUID(&uidNext)
		data.UIDNext = uidNext
	case "UIDVALIDITY":
		ok = dec.ExpectNumber(&data.UIDValidity)
	case "UNSEEN":
		var num uint32
		ok = dec.ExpectNumber(&num)
		data.NumUnseen = &num
	case "DELETED":
		var num uint32
		ok = dec.ExpectNumber(&num)
		data.NumDeleted = &num
	case "SIZE":
		var size int64
		ok = dec.ExpectNumber64(&size)
		data.Size = &size
	case "APPENDLIMIT":
		var num uint32
		if dec.Number(&num) {
			ok = true
		} else {
			ok = dec.ExpectNIL()
			num = ^uint32(0)
		}
		data.AppendLimit = &num
	case "DELETED-STORAGE":
		var storage int64
		ok = dec.ExpectNumber64(&storage)
		data.DeletedStorage = &storage
	case "HIGHESTMODSEQ":
		ok = dec.ExpectModSeq(&data.HighestModSeq)
	default:
		if !dec.DiscardValue() {
			return dec.Err()
		}
	}
	if !ok {
		return dec.Err()
	}
	return nil
}
