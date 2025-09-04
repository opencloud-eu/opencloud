package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

// Enable sends an ENABLE command.
//
// This command requires support for IMAP4rev2 or the ENABLE extension.
func (c *Client) Enable(caps ...imap.Cap) *EnableCommand {
	// Enabling an extension may change the IMAP syntax, so only allow the
	// extensions we support here
	for _, name := range caps {
		switch name {
		case imap.CapIMAP4rev2, imap.CapUTF8Accept, imap.CapMetadata, imap.CapMetadataServer:
			// ok
		default:
			done := make(chan error)
			close(done)
			err := fmt.Errorf("imapclient: cannot enable %q: not supported", name)
			return &EnableCommand{commandBase: commandBase{done: done, err: err}}
		}
	}

	cmd := &EnableCommand{}
	enc := c.beginCommand("ENABLE", cmd)
	for _, c := range caps {
		enc.SP().Atom(string(c))
	}
	enc.end()
	return cmd
}

func (c *Client) handleEnabled() error {
	caps, err := readCapabilities(c.dec)
	if err != nil {
		return err
	}

	c.mutex.Lock()
	for name := range caps {
		c.enabled[name] = struct{}{}
	}
	c.mutex.Unlock()

	if cmd := findPendingCmdByType[*EnableCommand](c); cmd != nil {
		cmd.data.Caps = caps
	}

	return nil
}

// EnableCommand is an ENABLE command.
type EnableCommand struct {
	commandBase
	data EnableData
}

func (cmd *EnableCommand) Wait() (*EnableData, error) {
	return &cmd.data, cmd.wait()
}

// EnableData is the data returned by the ENABLE command.
type EnableData struct {
	// Capabilities that were successfully enabled
	Caps imap.CapSet
}
