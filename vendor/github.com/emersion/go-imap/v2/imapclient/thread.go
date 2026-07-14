package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// ThreadOptions contains options for the THREAD command.
type ThreadOptions struct {
	Algorithm      imap.ThreadAlgorithm
	SearchCriteria *imap.SearchCriteria
}

func (c *Client) thread(numKind imapwire.NumKind, options *ThreadOptions) *ThreadCommand {
	cmd := &ThreadCommand{}
	enc := c.beginCommand(uidCmdName("THREAD", numKind), cmd)
	enc.SP().Atom(string(options.Algorithm)).SP().Atom("UTF-8").SP()
	writeSearchKey(enc.Encoder, options.SearchCriteria)
	enc.end()
	return cmd
}

// Thread sends a THREAD command.
//
// This command requires support for the THREAD extension.
func (c *Client) Thread(options *ThreadOptions) *ThreadCommand {
	return c.thread(imapwire.NumKindSeq, options)
}

// UIDThread sends a UID THREAD command.
//
// See Thread.
func (c *Client) UIDThread(options *ThreadOptions) *ThreadCommand {
	return c.thread(imapwire.NumKindUID, options)
}

func (c *Client) handleThread() error {
	cmd := findPendingCmdByType[*ThreadCommand](c)
	for c.dec.SP() {
		data, err := readThreadList(c.dec)
		if err != nil {
			return fmt.Errorf("in thread-list: %v", err)
		}
		if cmd != nil {
			cmd.data = append(cmd.data, *data)
		}
	}
	return nil
}

// ThreadCommand is a THREAD command.
type ThreadCommand struct {
	commandBase
	data []ThreadData
}

func (cmd *ThreadCommand) Wait() ([]ThreadData, error) {
	err := cmd.wait()
	return cmd.data, err
}

type ThreadData struct {
	Chain      []uint32
	SubThreads []ThreadData
}

func readThreadList(dec *imapwire.Decoder) (*ThreadData, error) {
	var data ThreadData
	err := dec.ExpectList(func() error {
		var num uint32
		if len(data.SubThreads) == 0 && dec.Number(&num) {
			data.Chain = append(data.Chain, num)
		} else {
			sub, err := readThreadList(dec)
			if err != nil {
				return err
			}
			data.SubThreads = append(data.SubThreads, *sub)
		}
		return nil
	})
	return &data, err
}
