package lh

import (
	"encoding/json"
	"fmt"
	"github.com/olekukonko/ll/lx"
	"io"
	"strings"
	"sync"
	"time"
)

type JSONHandler struct {
	writer   io.Writer
	timeFmt  string
	pretty   bool
	fieldMap map[string]string
	mu       sync.Mutex
}

type dumpSegment struct {
	Offset int      `json:"offset"`
	Hex    []string `json:"hex"`
	ASCII  string   `json:"ascii"`
}
type JsonOutput struct {
	Time      string                 `json:"ts"`
	Level     string                 `json:"lvl"`
	Class     string                 `json:"class"`
	Msg       string                 `json:"msg"`
	Namespace string                 `json:"ns"`
	Stack     []byte                 `json:"stack"`
	Dump      []dumpSegment          `json:"dump"`
	Fields    map[string]interface{} `json:"fields"`
}

func NewJSONHandler(w io.Writer, opts ...func(*JSONHandler)) *JSONHandler {
	h := &JSONHandler{
		writer:  w,
		timeFmt: time.RFC3339Nano,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *JSONHandler) Handle(e *lx.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if e.Class == lx.ClassDump {
		return h.handleDump(e)
	}
	return h.handleRegular(e)
}

func (h *JSONHandler) handleRegular(e *lx.Entry) error {
	entry := JsonOutput{
		Time:      e.Timestamp.Format(h.timeFmt),
		Level:     e.Level.String(),
		Class:     e.Class.String(),
		Msg:       e.Message,
		Namespace: e.Namespace,
		Dump:      nil,
		Fields:    e.Fields,
		Stack:     e.Stack,
	}

	enc := json.NewEncoder(h.writer)
	if h.pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(entry)
}

func (h *JSONHandler) handleDump(e *lx.Entry) error {

	var segments []dumpSegment
	lines := strings.Split(e.Message, "\n")

	for _, line := range lines {
		if !strings.HasPrefix(line, "pos") {
			continue
		}
		parts := strings.SplitN(line, "hex:", 2)
		if len(parts) != 2 {
			continue
		}
		// Parse position
		var offset int
		fmt.Sscanf(parts[0], "pos %d", &offset)

		// Parse hex and ASCII
		hexAscii := strings.SplitN(parts[1], "'", 2)
		hexStr := strings.Fields(strings.TrimSpace(hexAscii[0]))

		segments = append(segments, dumpSegment{
			Offset: offset,
			Hex:    hexStr,
			ASCII:  strings.Trim(hexAscii[1], "'"),
		})
	}

	return json.NewEncoder(h.writer).Encode(JsonOutput{
		Time:      e.Timestamp.Format(h.timeFmt),
		Level:     e.Level.String(),
		Class:     e.Class.String(),
		Msg:       "dumping segments",
		Namespace: e.Namespace,
		Dump:      segments,
		Fields:    e.Fields,
		Stack:     e.Stack,
	})
}
