//go:build enable_antithesis_sdk && linux && amd64 && !cgo

package internal

import (
	"log"
	"os"
	"path/filepath"
)

const sdkRunningInDegradedMode = true

type inAntithesisWithoutCgoHandler struct {
	outputFile *os.File
}

func (h *inAntithesisWithoutCgoHandler) output(message string) {
	if len(message) == 0 {
		return
	}
	h.outputFile.WriteString(message + "\n")
}

func (h *inAntithesisWithoutCgoHandler) random() uint64 {
	return osRandom()
}

func (h *inAntithesisWithoutCgoHandler) notify(edge uint64) bool {
	return false
}

func (h *inAntithesisWithoutCgoHandler) init_coverage(num_edges uint64, symbols string) uint64 {
	return 0
}

func (h *inAntithesisWithoutCgoHandler) notify_v2(edge uint64, hits uint64) (uint64, bool) {
	return 0, false
}

func (h *inAntithesisWithoutCgoHandler) lease_generation() (uint64, bool) {
	return 0, false
}

func init_in_antithesis() libHandler {
	dir, is_set := os.LookupEnv(outputDirEnvVar)
	if !is_set || len(dir) == 0 {
		return nil
	}

	path := filepath.Join(dir, fallbackOutputFilename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("%s Failed to open path %s: %v", errorLogLinePrefix, path, err)
		return nil
	}

	return &inAntithesisWithoutCgoHandler{file}
}
