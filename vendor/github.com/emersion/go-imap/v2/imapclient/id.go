package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// ID sends an ID command.
//
// The ID command is introduced in RFC 2971. It requires support for the ID
// extension.
//
// An example ID command:
//
//	ID ("name" "go-imap" "version" "1.0" "os" "Linux" "os-version" "7.9.4" "vendor" "Yahoo")
func (c *Client) ID(idData *imap.IDData) *IDCommand {
	cmd := &IDCommand{}
	enc := c.beginCommand("ID", cmd)

	if idData == nil {
		enc.SP().NIL()
		enc.end()
		return cmd
	}

	enc.SP().Special('(')
	isFirstKey := true
	if idData.Name != "" {
		addIDKeyValue(enc, &isFirstKey, "name", idData.Name)
	}
	if idData.Version != "" {
		addIDKeyValue(enc, &isFirstKey, "version", idData.Version)
	}
	if idData.OS != "" {
		addIDKeyValue(enc, &isFirstKey, "os", idData.OS)
	}
	if idData.OSVersion != "" {
		addIDKeyValue(enc, &isFirstKey, "os-version", idData.OSVersion)
	}
	if idData.Vendor != "" {
		addIDKeyValue(enc, &isFirstKey, "vendor", idData.Vendor)
	}
	if idData.SupportURL != "" {
		addIDKeyValue(enc, &isFirstKey, "support-url", idData.SupportURL)
	}
	if idData.Address != "" {
		addIDKeyValue(enc, &isFirstKey, "address", idData.Address)
	}
	if idData.Date != "" {
		addIDKeyValue(enc, &isFirstKey, "date", idData.Date)
	}
	if idData.Command != "" {
		addIDKeyValue(enc, &isFirstKey, "command", idData.Command)
	}
	if idData.Arguments != "" {
		addIDKeyValue(enc, &isFirstKey, "arguments", idData.Arguments)
	}
	if idData.Environment != "" {
		addIDKeyValue(enc, &isFirstKey, "environment", idData.Environment)
	}

	enc.Special(')')
	enc.end()
	return cmd
}

func addIDKeyValue(enc *commandEncoder, isFirstKey *bool, key, value string) {
	if isFirstKey == nil {
		panic("isFirstKey cannot be nil")
	} else if !*isFirstKey {
		enc.SP().Quoted(key).SP().Quoted(value)
	} else {
		enc.Quoted(key).SP().Quoted(value)
	}
	*isFirstKey = false
}

func (c *Client) handleID() error {
	data, err := c.readID(c.dec)
	if err != nil {
		return fmt.Errorf("in id: %v", err)
	}

	if cmd := findPendingCmdByType[*IDCommand](c); cmd != nil {
		cmd.data = *data
	}

	return nil
}

func (c *Client) readID(dec *imapwire.Decoder) (*imap.IDData, error) {
	var data = imap.IDData{}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	if dec.ExpectNIL() {
		return &data, nil
	}

	currKey := ""
	err := dec.ExpectList(func() error {
		var keyOrValue string
		if !dec.String(&keyOrValue) {
			return fmt.Errorf("in id key-val list: %v", dec.Err())
		}

		if currKey == "" {
			currKey = keyOrValue
			return nil
		}

		switch currKey {
		case "name":
			data.Name = keyOrValue
		case "version":
			data.Version = keyOrValue
		case "os":
			data.OS = keyOrValue
		case "os-version":
			data.OSVersion = keyOrValue
		case "vendor":
			data.Vendor = keyOrValue
		case "support-url":
			data.SupportURL = keyOrValue
		case "address":
			data.Address = keyOrValue
		case "date":
			data.Date = keyOrValue
		case "command":
			data.Command = keyOrValue
		case "arguments":
			data.Arguments = keyOrValue
		case "environment":
			data.Environment = keyOrValue
		default:
			// Ignore unknown key
			// Yahoo server sends "host" and "remote-host" keys
			// which are not defined in RFC 2971
		}
		currKey = ""

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &data, nil
}

type IDCommand struct {
	commandBase
	data imap.IDData
}

func (r *IDCommand) Wait() (*imap.IDData, error) {
	return &r.data, r.wait()
}
