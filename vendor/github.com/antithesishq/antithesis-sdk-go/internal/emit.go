//go:build enable_antithesis_sdk

package internal

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
	"runtime"
)

func Json_data(v any) error {
	if data, err := json.Marshal(v); err != nil {
		return err
	} else {
		handler.output(string(data))
		return nil
	}
}

func Get_random() uint64 {
	return handler.random()
}

func Notify(edge uint64) bool {
	return handler.notify(edge)
}

// NotifyV2 reports a coverage hit through the lease ABI (linkage v2); ok is
// false when the loaded library does not support leases.
func NotifyV2(edge uint64, hits uint64) (uint64, bool) {
	return handler.notify_v2(edge, hits)
}

// LeaseGeneration reads the lease revocation word; ok is false when the
// loaded library does not support leases.
func LeaseGeneration() (uint64, bool) {
	return handler.lease_generation()
}

func InitCoverage(num_edges uint64, symbols string) uint64 {
	return handler.init_coverage(num_edges, symbols)
}

type libHandler interface {
	output(message string)
	random() uint64
	notify(edge uint64) bool
	notify_v2(edge uint64, hits uint64) (uint64, bool)
	lease_generation() (uint64, bool)
	init_coverage(num_edges uint64, symbols string) uint64
}

const (
	errorLogLinePrefix = "[* antithesis-sdk-go *]"
)

var handler libHandler

// Mirrors the no_antithesis_sdk implementation of random.GetRandom.
func osRandom() uint64 {
	var tmp [8]byte
	crand.Read(tmp[:])
	return binary.LittleEndian.Uint64(tmp[:])
}

type localHandler struct {
	outputFile *os.File // can be nil
}

func (h *localHandler) output(message string) {
	msg_len := len(message)
	if msg_len == 0 {
		return
	}
	if h.outputFile != nil {
		h.outputFile.WriteString(message + "\n")
	}
}

func (h *localHandler) random() uint64 {
	return osRandom()
}

func (h *localHandler) notify(edge uint64) bool {
	return false
}

func (h *localHandler) notify_v2(edge uint64, hits uint64) (uint64, bool) {
	return 0, false
}

func (h *localHandler) lease_generation() (uint64, bool) {
	return 0, false
}

func (h *localHandler) init_coverage(num_edges uint64, symbols string) uint64 {
	return 0
}

func init() {
	handler = init_in_antithesis()
	if handler == nil {
		// Otherwise fallback to the local handler.
		handler = openLocalHandler()
	}

	// The version message is the first output of every run, emitted when the
	// handler comes up.
	emitVersionMessage()

	if sdkRunningInDegradedMode {
		emitDegradedModeProperty()
	}
}

func emitVersionMessage() {
	languageBlock := map[string]any{
		"name":    "Go",
		"version": runtime.Version(),
	}
	versionBlock := map[string]any{
		"language":         languageBlock,
		"sdk_version":      SDK_Version,
		"protocol_version": Protocol_Version,
	}
	if data, err := json.Marshal(map[string]any{"antithesis_sdk": versionBlock}); err == nil {
		handler.output(string(data))
	}
}

// If `localOutputEnvVar` is set to a non-empty path, attempt to open that path for appending
// to serve as the log file of the local handler.
// Otherwise, we don't have a log file, and logging is a no-op in the local handler.
func openLocalHandler() *localHandler {
	path, is_set := os.LookupEnv(localOutputEnvVar)
	if !is_set || len(path) == 0 {
		return &localHandler{nil}
	}

	// Open the file for writing (create if needed and possible, use O_APPEND to support concurrent writers)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("%s Failed to open path %s: %v", errorLogLinePrefix, path, err)
		file = nil
	}

	return &localHandler{file}
}

const degradedModeProperty = "Go application under test compiled with CGO enabled"

func emitDegradedModeProperty() {
	Json_data(map[string]any{
		"antithesis_assert": map[string]any{
			"hit":          true,
			"must_hit":     true,
			"assert_type":  "always",
			"display_type": "Always",
			"message":      degradedModeProperty,
			"id":           degradedModeProperty,
			"condition":    false,
			"location": map[string]any{
				"class":        "",
				"function":     "",
				"file":         "",
				"begin_line":   0,
				"begin_column": 0,
			},
			"details": map[string]any{
				"info": "This binary was built without CGO enabled, which is required for " +
					"coverage-guided fuzzing and thread pausing. Rebuild with CGO_ENABLED=1 so the " +
					"SDK can link " + defaultNativeLibraryPath + " to enable them.",
			},
		},
	})
}
