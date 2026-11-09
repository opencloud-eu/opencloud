package command

import (
	"fmt"
	"path/filepath"

	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/lookup"
)

const posixDriver = "posix"

// isPosixStorage tells whether the storage at basePath uses the posix layout.
// The layout is detected from the data on disk, so the CLI works even when
// STORAGE_USERS_DRIVER is not set in the shell it runs in (e.g. it is only
// set in the env file of the service). Only if the layout can't be detected,
// e.g. because the storage is still empty, the configured driver is used.
func isPosixStorage(basePath string, configuredDriver string) bool {
	configured := configuredDriver == posixDriver

	var detected bool
	switch {
	case globMatches(filepath.Join(basePath, "spaces", "*", "*", "nodes")):
		detected = false
	case globMatches(filepath.Join(basePath, "*", "*", lookup.MetadataDir)):
		detected = true
	default:
		return configured
	}

	if detected != configured {
		fmt.Printf("storage at %s uses the %s layout, ignoring configured driver %q\n", basePath, layoutName(detected), configuredDriver)
	}
	return detected
}

func globMatches(pattern string) bool {
	matches, err := filepath.Glob(pattern)
	return err == nil && len(matches) > 0
}

func layoutName(posix bool) string {
	if posix {
		return "posix"
	}
	return "decomposed"
}
