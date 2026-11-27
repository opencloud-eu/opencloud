package command

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/audit/pkg/config"
)

// an installation that turns on the audit stream and keeps its file settings
// would otherwise stop writing the file without a word
func TestIgnoredLegacySettings(t *testing.T) {
	assert.Empty(t, ignoredLegacySettings(config.Auditlog{LogToConsole: true, Format: "json"}), "the defaults are no file settings")
	assert.Equal(t, []string{"AUDIT_LOG_TO_FILE", "AUDIT_FILEPATH"},
		ignoredLegacySettings(config.Auditlog{LogToFile: true, FilePath: "/var/log/audit.log"}))
	assert.Equal(t, []string{"AUDIT_FILEPATH"}, ignoredLegacySettings(config.Auditlog{FilePath: "/var/log/audit.log"}))
}
