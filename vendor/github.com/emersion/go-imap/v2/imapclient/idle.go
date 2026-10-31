package imapclient

import (
	"fmt"
	"sync/atomic"
	"time"
)

const idleRestartInterval = 28 * time.Minute

// Idle sends an IDLE command.
//
// Unlike other commands, this method blocks until the server acknowledges it.
// On success, the IDLE command is running and other commands cannot be sent.
// The caller must invoke IdleCommand.Close to stop IDLE and unblock the
// client.
//
// This command requires support for IMAP4rev2 or the IDLE extension. The IDLE
// command is restarted automatically to avoid getting disconnected due to
// inactivity timeouts.
func (c *Client) Idle() (*IdleCommand, error) {
	child, err := c.idle()
	if err != nil {
		return nil, err
	}

	cmd := &IdleCommand{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go cmd.run(c, child)
	return cmd, nil
}

// IdleCommand is an IDLE command.
//
// Initially, the IDLE command is running. The server may send unilateral
// data. The client cannot send any command while IDLE is running.
//
// Close must be called to stop the IDLE command.
type IdleCommand struct {
	stopped atomic.Bool
	stop    chan struct{}
	done    chan struct{}

	err       error
	lastChild *idleCommand
}

func (cmd *IdleCommand) run(c *Client, child *idleCommand) {
	defer close(cmd.done)

	timer := time.NewTimer(idleRestartInterval)
	defer timer.Stop()

	defer func() {
		if child != nil {
			if err := child.Close(); err != nil && cmd.err == nil {
				cmd.err = err
			}
		}
	}()

	for {
		select {
		case <-timer.C:
			timer.Reset(idleRestartInterval)

			if cmd.err = child.Close(); cmd.err != nil {
				return
			}
			if child, cmd.err = c.idle(); cmd.err != nil {
				return
			}
		case <-c.decCh:
			cmd.lastChild = child
			return
		case <-cmd.stop:
			cmd.lastChild = child
			return
		}
	}
}

// Close stops the IDLE command.
//
// This method blocks until the command to stop IDLE is written, but doesn't
// wait for the server to respond. Callers can use Wait for this purpose.
func (cmd *IdleCommand) Close() error {
	if cmd.stopped.Swap(true) {
		return fmt.Errorf("imapclient: IDLE already closed")
	}
	close(cmd.stop)
	<-cmd.done
	return cmd.err
}

// Wait blocks until the IDLE command has completed.
func (cmd *IdleCommand) Wait() error {
	<-cmd.done
	if cmd.err != nil {
		return cmd.err
	}
	return cmd.lastChild.Wait()
}

func (c *Client) idle() (*idleCommand, error) {
	cmd := &idleCommand{}
	contReq := c.registerContReq(cmd)
	cmd.enc = c.beginCommand("IDLE", cmd)
	cmd.enc.flush()

	_, err := contReq.Wait()
	if err != nil {
		cmd.enc.end()
		return nil, err
	}

	return cmd, nil
}

// idleCommand represents a singular IDLE command, without the restart logic.
type idleCommand struct {
	commandBase
	enc *commandEncoder
}

// Close stops the IDLE command.
//
// This method blocks until the command to stop IDLE is written, but doesn't
// wait for the server to respond. Callers can use Wait for this purpose.
func (cmd *idleCommand) Close() error {
	if cmd.err != nil {
		return cmd.err
	}
	if cmd.enc == nil {
		return fmt.Errorf("imapclient: IDLE command closed twice")
	}
	cmd.enc.client.setWriteTimeout(cmdWriteTimeout)
	_, err := cmd.enc.client.bw.WriteString("DONE\r\n")
	if err == nil {
		err = cmd.enc.client.bw.Flush()
	}
	cmd.enc.end()
	cmd.enc = nil
	return err
}

// Wait blocks until the IDLE command has completed.
//
// Wait can only be called after Close.
func (cmd *idleCommand) Wait() error {
	if cmd.enc != nil {
		panic("imapclient: idleCommand.Close must be called before Wait")
	}
	return cmd.wait()
}
