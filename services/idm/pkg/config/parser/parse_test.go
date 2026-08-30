package parser

import (
	"os"
	"testing"

	"github.com/opencloud-eu/opencloud/services/idm/pkg/config/defaults"
	"gotest.tools/v3/assert"
)

func TestParseConfigReadsAdminPasswordFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "admin-password")
	assert.NilError(t, err)
	_, err = file.WriteString("secret\n")
	assert.NilError(t, err)
	assert.NilError(t, file.Close())

	t.Setenv("IDM_ADMIN_PASSWORD_FILE", file.Name())
	t.Setenv("IDM_SVC_PASSWORD", "idm")
	t.Setenv("IDM_IDPSVC_PASSWORD", "idp")
	t.Setenv("IDM_REVASVC_PASSWORD", "reva")
	t.Setenv("OC_ADMIN_USER_ID", "admin")

	cfg := defaults.DefaultConfig()
	err = ParseConfig(cfg)

	assert.NilError(t, err)
	assert.Equal(t, cfg.ServiceUserPasswords.OCAdmin, "secret")
}
