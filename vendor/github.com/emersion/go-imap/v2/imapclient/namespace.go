package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Namespace sends a NAMESPACE command.
//
// This command requires support for IMAP4rev2 or the NAMESPACE extension.
func (c *Client) Namespace() *NamespaceCommand {
	cmd := &NamespaceCommand{}
	c.beginCommand("NAMESPACE", cmd).end()
	return cmd
}

func (c *Client) handleNamespace() error {
	data, err := readNamespaceResponse(c.dec)
	if err != nil {
		return fmt.Errorf("in namespace-response: %v", err)
	}
	if cmd := findPendingCmdByType[*NamespaceCommand](c); cmd != nil {
		cmd.data = *data
	}
	return nil
}

// NamespaceCommand is a NAMESPACE command.
type NamespaceCommand struct {
	commandBase
	data imap.NamespaceData
}

func (cmd *NamespaceCommand) Wait() (*imap.NamespaceData, error) {
	return &cmd.data, cmd.wait()
}

func readNamespaceResponse(dec *imapwire.Decoder) (*imap.NamespaceData, error) {
	var (
		data imap.NamespaceData
		err  error
	)

	data.Personal, err = readNamespace(dec)
	if err != nil {
		return nil, err
	}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	data.Other, err = readNamespace(dec)
	if err != nil {
		return nil, err
	}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	data.Shared, err = readNamespace(dec)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func readNamespace(dec *imapwire.Decoder) ([]imap.NamespaceDescriptor, error) {
	var l []imap.NamespaceDescriptor
	err := dec.ExpectNList(func() error {
		descr, err := readNamespaceDescr(dec)
		if err != nil {
			return fmt.Errorf("in namespace-descr: %v", err)
		}
		l = append(l, *descr)
		return nil
	})
	return l, err
}

func readNamespaceDescr(dec *imapwire.Decoder) (*imap.NamespaceDescriptor, error) {
	var descr imap.NamespaceDescriptor

	if !dec.ExpectSpecial('(') || !dec.ExpectString(&descr.Prefix) || !dec.ExpectSP() {
		return nil, dec.Err()
	}

	var err error
	descr.Delim, err = readDelim(dec)
	if err != nil {
		return nil, err
	}

	// Skip namespace-response-extensions
	for dec.SP() {
		if !dec.DiscardValue() {
			return nil, dec.Err()
		}
	}

	if !dec.ExpectSpecial(')') {
		return nil, dec.Err()
	}

	return &descr, nil
}
