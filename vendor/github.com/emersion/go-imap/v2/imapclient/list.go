package imapclient

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func getSelectOpts(options *imap.ListOptions) []string {
	if options == nil {
		return nil
	}

	var l []string
	if options.SelectSubscribed {
		l = append(l, "SUBSCRIBED")
	}
	if options.SelectRemote {
		l = append(l, "REMOTE")
	}
	if options.SelectRecursiveMatch {
		l = append(l, "RECURSIVEMATCH")
	}
	if options.SelectSpecialUse {
		l = append(l, "SPECIAL-USE")
	}
	return l
}

func getReturnOpts(options *imap.ListOptions) []string {
	if options == nil {
		return nil
	}

	var l []string
	if options.ReturnSubscribed {
		l = append(l, "SUBSCRIBED")
	}
	if options.ReturnChildren {
		l = append(l, "CHILDREN")
	}
	if options.ReturnStatus != nil {
		l = append(l, "STATUS")
	}
	if options.ReturnSpecialUse {
		l = append(l, "SPECIAL-USE")
	}
	return l
}

// List sends a LIST command.
//
// The caller must fully consume the ListCommand. A simple way to do so is to
// defer a call to ListCommand.Close.
//
// A nil options pointer is equivalent to a zero options value.
//
// A non-zero options value requires support for IMAP4rev2 or the LIST-EXTENDED
// extension.
func (c *Client) List(ref, pattern string, options *imap.ListOptions) *ListCommand {
	cmd := &ListCommand{
		mailboxes:    make(chan *imap.ListData, 64),
		returnStatus: options != nil && options.ReturnStatus != nil,
	}
	enc := c.beginCommand("LIST", cmd)
	if selectOpts := getSelectOpts(options); len(selectOpts) > 0 {
		enc.SP().List(len(selectOpts), func(i int) {
			enc.Atom(selectOpts[i])
		})
	}
	enc.SP().Mailbox(ref).SP().Mailbox(pattern)
	if returnOpts := getReturnOpts(options); len(returnOpts) > 0 {
		enc.SP().Atom("RETURN").SP().List(len(returnOpts), func(i int) {
			opt := returnOpts[i]
			enc.Atom(opt)
			if opt == "STATUS" {
				returnStatus := statusItems(options.ReturnStatus)
				enc.SP().List(len(returnStatus), func(j int) {
					enc.Atom(returnStatus[j])
				})
			}
		})
	}
	enc.end()
	return cmd
}

func (c *Client) handleList() error {
	data, err := readList(c.dec)
	if err != nil {
		return fmt.Errorf("in LIST: %v", err)
	}

	cmd := c.findPendingCmdFunc(func(cmd command) bool {
		switch cmd := cmd.(type) {
		case *ListCommand:
			return true // TODO: match pattern, check if already handled
		case *SelectCommand:
			return cmd.mailbox == data.Mailbox && cmd.data.List == nil
		default:
			return false
		}
	})
	switch cmd := cmd.(type) {
	case *ListCommand:
		if cmd.returnStatus {
			if cmd.pendingData != nil {
				cmd.mailboxes <- cmd.pendingData
			}
			cmd.pendingData = data
		} else {
			cmd.mailboxes <- data
		}
	case *SelectCommand:
		cmd.data.List = data
	}

	return nil
}

// ListCommand is a LIST command.
type ListCommand struct {
	commandBase
	mailboxes chan *imap.ListData

	returnStatus bool
	pendingData  *imap.ListData
}

// Next advances to the next mailbox.
//
// On success, the mailbox LIST data is returned. On error or if there are no
// more mailboxes, nil is returned.
func (cmd *ListCommand) Next() *imap.ListData {
	return <-cmd.mailboxes
}

// Close releases the command.
//
// Calling Close unblocks the IMAP client decoder and lets it read the next
// responses. Next will always return nil after Close.
func (cmd *ListCommand) Close() error {
	for cmd.Next() != nil {
		// ignore
	}
	return cmd.wait()
}

// Collect accumulates mailboxes into a list.
//
// This is equivalent to calling Next repeatedly and then Close.
func (cmd *ListCommand) Collect() ([]*imap.ListData, error) {
	var l []*imap.ListData
	for {
		data := cmd.Next()
		if data == nil {
			break
		}
		l = append(l, data)
	}
	return l, cmd.Close()
}

func readList(dec *imapwire.Decoder) (*imap.ListData, error) {
	var data imap.ListData

	var err error
	data.Attrs, err = internal.ExpectMailboxAttrList(dec)
	if err != nil {
		return nil, fmt.Errorf("in mbx-list-flags: %w", err)
	}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	data.Delim, err = readDelim(dec)
	if err != nil {
		return nil, err
	}

	if !dec.ExpectSP() || !dec.ExpectMailbox(&data.Mailbox) {
		return nil, dec.Err()
	}

	if dec.SP() {
		err := dec.ExpectList(func() error {
			var tag string
			if !dec.ExpectAString(&tag) || !dec.ExpectSP() {
				return dec.Err()
			}
			var err error
			switch strings.ToUpper(tag) {
			case "CHILDINFO":
				data.ChildInfo, err = readChildInfoExtendedItem(dec)
				if err != nil {
					return fmt.Errorf("in childinfo-extended-item: %v", err)
				}
			case "OLDNAME":
				data.OldName, err = readOldNameExtendedItem(dec)
				if err != nil {
					return fmt.Errorf("in oldname-extended-item: %v", err)
				}
			default:
				if !dec.DiscardValue() {
					return fmt.Errorf("in tagged-ext-val: %v", err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("in mbox-list-extended: %v", err)
		}
	}

	return &data, nil
}

func readChildInfoExtendedItem(dec *imapwire.Decoder) (*imap.ListDataChildInfo, error) {
	var childInfo imap.ListDataChildInfo
	err := dec.ExpectList(func() error {
		var opt string
		if !dec.ExpectAString(&opt) {
			return dec.Err()
		}
		if strings.ToUpper(opt) == "SUBSCRIBED" {
			childInfo.Subscribed = true
		}
		return nil
	})
	return &childInfo, err
}

func readOldNameExtendedItem(dec *imapwire.Decoder) (string, error) {
	var name string
	if !dec.ExpectSpecial('(') || !dec.ExpectMailbox(&name) || !dec.ExpectSpecial(')') {
		return "", dec.Err()
	}
	return name, nil
}

func readDelim(dec *imapwire.Decoder) (rune, error) {
	var delimStr string
	if dec.Quoted(&delimStr) {
		delim, size := utf8.DecodeRuneInString(delimStr)
		if delim == utf8.RuneError || size != len(delimStr) {
			return 0, fmt.Errorf("mailbox delimiter must be a single rune")
		}
		return delim, nil
	} else if !dec.ExpectNIL() {
		return 0, dec.Err()
	} else {
		return 0, nil
	}
}
