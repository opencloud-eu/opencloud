package sse

import (
	"io"
	"strconv"
	"strings"

	"github.com/tmaxmax/go-sse/internal/parser"
)

// The Event struct represents an event sent to the client by a server.
type Event struct {
	// The last non-empty ID of all the events received. This may not be
	// the ID of the latest event!
	LastEventID string
	// The event's type. It is empty if the event is unnamed.
	Type string
	// The event's payload.
	Data string
}

// ReadConfig is used to configure how Read behaves.
type ReadConfig struct {
	// MaxEventSize is the maximum expected length of the byte sequence
	// representing a single event. Parsing events longer than that
	// will result in an error.
	//
	// By default this limit is 64KB. You don't need to set this if it
	// is enough for your needs (e.g. the events you receive don't contain
	// larger amounts of data).
	MaxEventSize int
}

// Read parses an SSE stream and yields all incoming events,
// On any encountered errors iteration stops and no further events are parsed –
// the loop can safely be ended on error. If EOF is reached, the Read operation
// is considered successful and no error is returned. An Event will never
// be yielded together with an error.
//
// Read is especially useful for parsing responses from services which
// communicate using SSE but not over long-lived connections – for example,
// LLM APIs.
//
// Read handles the Event.LastEventID value just as the browser SSE client
// (EventSource) would – for every event, the last encountered event ID will be given,
// even if the ID is not the current event's ID. Read, unlike EventSource, does
// not set Event.Type to "message" if no "event" field is received, leaving
// it blank.
//
// Read provides no way to handle the "retry" field and doesn't handle retrying.
// Use a Client and a Connection if you need to retry requests.
func Read(r io.Reader, cfg *ReadConfig) func(func(Event, error) bool) {
	pf := func() *parser.Parser {
		p := parser.New(r)
		if cfg != nil && cfg.MaxEventSize > 0 {
			// NOTE(tmaxmax): we don't allow setting the buffer at the moment.
			// ReadConfig objects might be shared between Read calls executed in
			// different goroutines and having an actual []byte in it seems dangerous.
			// If there is demand it can be added.
			p.Buffer(nil, cfg.MaxEventSize)
		}
		return p
	}

	// We take a factory function for the parser so that Read can be inlined by the compiler.
	return read(pf, "", nil, true)
}

func read(pf func() *parser.Parser, lastEventID string, onRetry func(int64), ignoreEOF bool) func(func(Event, error) bool) {
	return func(yield func(Event, error) bool) {
		p := pf()

		typ, sb, dirty := "", strings.Builder{}, false
		doYield := func(data string) bool {
			if data != "" {
				data = data[:len(data)-1]
			}
			return yield(Event{LastEventID: lastEventID, Data: data, Type: typ}, nil)
		}

		for f := (parser.Field{}); p.Next(&f); {
			switch f.Name { //nolint:exhaustive // Comment fields are not parsed.
			case parser.FieldNameData:
				sb.WriteString(f.Value)
				sb.WriteByte('\n')
				dirty = true
			case parser.FieldNameEvent:
				typ = f.Value
				dirty = true
			case parser.FieldNameID:
				// empty IDs are valid, only IDs that contain the null byte must be ignored:
				// https://html.spec.whatwg.org/multipage/server-sent-events.html#event-stream-interpretation
				if strings.IndexByte(f.Value, 0) != -1 {
					break
				}

				lastEventID = f.Value
				dirty = true
			case parser.FieldNameRetry:
				n, err := strconv.ParseInt(f.Value, 10, 64)
				if err != nil {
					break
				}
				if n >= 0 && onRetry != nil {
					onRetry(n)
					dirty = true
				}
			default:
				if dirty {
					if !doYield(sb.String()) {
						return
					}
					sb.Reset()
					typ = ""
					dirty = false
				}
			}
		}

		err := p.Err()
		isEOF := err == io.EOF //nolint:errorlint // Our scanner returns io.EOF unwrapped

		if dirty && isEOF {
			if !doYield(sb.String()) {
				return
			}
		}

		if err != nil && !(ignoreEOF && isEOF) {
			yield(Event{}, err)
		}
	}
}
