package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// GetQuota sends a GETQUOTA command.
//
// This command requires support for the QUOTA extension.
func (c *Client) GetQuota(root string) *GetQuotaCommand {
	cmd := &GetQuotaCommand{root: root}
	enc := c.beginCommand("GETQUOTA", cmd)
	enc.SP().String(root)
	enc.end()
	return cmd
}

// GetQuotaRoot sends a GETQUOTAROOT command.
//
// This command requires support for the QUOTA extension.
func (c *Client) GetQuotaRoot(mailbox string) *GetQuotaRootCommand {
	cmd := &GetQuotaRootCommand{mailbox: mailbox}
	enc := c.beginCommand("GETQUOTAROOT", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// SetQuota sends a SETQUOTA command.
//
// This command requires support for the SETQUOTA extension.
func (c *Client) SetQuota(root string, limits map[imap.QuotaResourceType]int64) *Command {
	// TODO: consider returning the QUOTA response data?
	cmd := &Command{}
	enc := c.beginCommand("SETQUOTA", cmd)
	enc.SP().String(root).SP().Special('(')
	i := 0
	for typ, limit := range limits {
		if i > 0 {
			enc.SP()
		}
		enc.Atom(string(typ)).SP().Number64(limit)
		i++
	}
	enc.Special(')')
	enc.end()
	return cmd
}

func (c *Client) handleQuota() error {
	data, err := readQuotaResponse(c.dec)
	if err != nil {
		return fmt.Errorf("in quota-response: %v", err)
	}

	cmd := c.findPendingCmdFunc(func(cmd command) bool {
		switch cmd := cmd.(type) {
		case *GetQuotaCommand:
			return cmd.root == data.Root
		case *GetQuotaRootCommand:
			for _, root := range cmd.roots {
				if root == data.Root {
					return true
				}
			}
			return false
		default:
			return false
		}
	})
	switch cmd := cmd.(type) {
	case *GetQuotaCommand:
		cmd.data = data
	case *GetQuotaRootCommand:
		cmd.data = append(cmd.data, *data)
	}
	return nil
}

func (c *Client) handleQuotaRoot() error {
	mailbox, roots, err := readQuotaRoot(c.dec)
	if err != nil {
		return fmt.Errorf("in quotaroot-response: %v", err)
	}

	cmd := c.findPendingCmdFunc(func(anyCmd command) bool {
		cmd, ok := anyCmd.(*GetQuotaRootCommand)
		if !ok {
			return false
		}
		return cmd.mailbox == mailbox
	})
	if cmd != nil {
		cmd := cmd.(*GetQuotaRootCommand)
		cmd.roots = roots
	}
	return nil
}

// GetQuotaCommand is a GETQUOTA command.
type GetQuotaCommand struct {
	commandBase
	root string
	data *QuotaData
}

func (cmd *GetQuotaCommand) Wait() (*QuotaData, error) {
	if err := cmd.wait(); err != nil {
		return nil, err
	}
	return cmd.data, nil
}

// GetQuotaRootCommand is a GETQUOTAROOT command.
type GetQuotaRootCommand struct {
	commandBase
	mailbox string
	roots   []string
	data    []QuotaData
}

func (cmd *GetQuotaRootCommand) Wait() ([]QuotaData, error) {
	if err := cmd.wait(); err != nil {
		return nil, err
	}
	return cmd.data, nil
}

// QuotaData is the data returned by a QUOTA response.
type QuotaData struct {
	Root      string
	Resources map[imap.QuotaResourceType]QuotaResourceData
}

// QuotaResourceData contains the usage and limit for a quota resource.
type QuotaResourceData struct {
	Usage int64
	Limit int64
}

func readQuotaResponse(dec *imapwire.Decoder) (*QuotaData, error) {
	var data QuotaData
	if !dec.ExpectAString(&data.Root) || !dec.ExpectSP() {
		return nil, dec.Err()
	}
	data.Resources = make(map[imap.QuotaResourceType]QuotaResourceData)
	err := dec.ExpectList(func() error {
		var (
			name    string
			resData QuotaResourceData
		)
		if !dec.ExpectAtom(&name) || !dec.ExpectSP() || !dec.ExpectNumber64(&resData.Usage) || !dec.ExpectSP() || !dec.ExpectNumber64(&resData.Limit) {
			return fmt.Errorf("in quota-resource: %v", dec.Err())
		}
		data.Resources[imap.QuotaResourceType(name)] = resData
		return nil
	})
	return &data, err
}

func readQuotaRoot(dec *imapwire.Decoder) (mailbox string, roots []string, err error) {
	if !dec.ExpectMailbox(&mailbox) {
		return "", nil, dec.Err()
	}
	for dec.SP() {
		var root string
		if !dec.ExpectAString(&root) {
			return "", nil, dec.Err()
		}
		roots = append(roots, root)
	}
	return mailbox, roots, nil
}
