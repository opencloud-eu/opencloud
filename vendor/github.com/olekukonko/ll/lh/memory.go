package lh

import (
	"fmt"
	"github.com/olekukonko/ll/lx"
	"io"
	"sync"
)

// MemoryHandler is an lx.Handler that stores log entries in memory.
// Useful for testing or buffering logs for later inspection.
type MemoryHandler struct {
	mu      sync.RWMutex
	entries []*lx.Entry
}

// NewMemoryHandler creates a new MemoryHandler.
func NewMemoryHandler() *MemoryHandler {
	return &MemoryHandler{
		entries: make([]*lx.Entry, 0),
	}
}

// Handle stores the log entry in memory.
func (h *MemoryHandler) Handle(entry *lx.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, entry)
	return nil
}

// Entries returns a copy of the stored log entries.
func (h *MemoryHandler) Entries() []*lx.Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	entries := make([]*lx.Entry, len(h.entries))
	copy(entries, h.entries)
	return entries
}

// Reset clears all stored entries.
func (h *MemoryHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = h.entries[:0]
}

// Dump writes all stored log entries to the provided io.Writer in text format.
// Entries are formatted as they would be by a TextHandler, including namespace, level,
// message, and fields. Thread-safe with read lock.
// Example:
//
//	logger := New("test", WithHandler(NewMemoryHandler())).Enable()
//	logger.Info("Test message")
//	handler := logger.handler.(*MemoryHandler)
//	handler.Dump(os.Stdout) // Output: [test] INFO: Test message
func (h *MemoryHandler) Dump(w io.Writer) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Create a temporary TextHandler to format entries
	tempHandler := NewTextHandler(w)

	for _, entry := range h.entries {
		if err := tempHandler.Handle(entry); err != nil {
			return fmt.Errorf("failed to dump entry: %w", err)
		}
	}
	return nil
}
