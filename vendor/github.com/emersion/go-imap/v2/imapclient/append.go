package imapclient

import (
	"io"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
)

// Append sends an APPEND command.
//
// The caller must call AppendCommand.Close.
//
// The options are optional.
func (c *Client) Append(mailbox string, size int64, options *imap.AppendOptions) *AppendCommand {
	cmd := &AppendCommand{}
	cmd.enc = c.beginCommand("APPEND", cmd)
	cmd.enc.SP().Mailbox(mailbox).SP()
	if options != nil && len(options.Flags) > 0 {
		cmd.enc.List(len(options.Flags), func(i int) {
			cmd.enc.Flag(options.Flags[i])
		}).SP()
	}
	if options != nil && !options.Time.IsZero() {
		cmd.enc.String(options.Time.Format(internal.DateTimeLayout)).SP()
	}
	// TODO: literal8 for BINARY
	// TODO: UTF8 data ext for UTF8=ACCEPT, with literal8
	cmd.wc = cmd.enc.Literal(size)
	return cmd
}

// AppendCommand is an APPEND command.
//
// Callers must write the message contents, then call Close.
type AppendCommand struct {
	commandBase
	enc  *commandEncoder
	wc   io.WriteCloser
	data imap.AppendData
}

func (cmd *AppendCommand) Write(b []byte) (int, error) {
	return cmd.wc.Write(b)
}

func (cmd *AppendCommand) Close() error {
	err := cmd.wc.Close()
	if cmd.enc != nil {
		cmd.enc.end()
		cmd.enc = nil
	}
	return err
}

func (cmd *AppendCommand) Wait() (*imap.AppendData, error) {
	return &cmd.data, cmd.wait()
}
