package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Capability sends a CAPABILITY command.
func (c *Client) Capability() *CapabilityCommand {
	cmd := &CapabilityCommand{}
	c.beginCommand("CAPABILITY", cmd).end()
	return cmd
}

func (c *Client) handleCapability() error {
	caps, err := readCapabilities(c.dec)
	if err != nil {
		return err
	}
	c.setCaps(caps)
	if cmd := findPendingCmdByType[*CapabilityCommand](c); cmd != nil {
		cmd.caps = caps
	}
	return nil
}

// CapabilityCommand is a CAPABILITY command.
type CapabilityCommand struct {
	commandBase
	caps imap.CapSet
}

func (cmd *CapabilityCommand) Wait() (imap.CapSet, error) {
	err := cmd.wait()
	return cmd.caps, err
}

func readCapabilities(dec *imapwire.Decoder) (imap.CapSet, error) {
	caps := make(imap.CapSet)
	for dec.SP() {
		// Some IMAP servers send multiple SP between caps:
		// https://github.com/emersion/go-imap/pull/652
		for dec.SP() {
		}

		var name string
		if !dec.ExpectAtom(&name) {
			return caps, fmt.Errorf("in capability-data: %v", dec.Err())
		}
		caps[imap.Cap(name)] = struct{}{}
	}
	return caps, nil
}
