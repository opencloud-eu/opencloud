package middleware

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestLoadCSPConfig(t *testing.T) {
	// setup test env
	presetYaml := `
directives:
  frame-src:
    - '''self'''
    - 'https://embed.diagrams.net/'
    - 'https://${ONLYOFFICE_DOMAIN|onlyoffice.opencloud.test}/'
    - 'https://${COLLABORA_DOMAIN|collabora.opencloud.test}/'
`

	customYaml := `
directives:
  img-src:
    - '''self'''
    - 'data:'
  frame-src:
    - 'https://some.custom.domain/'
`
	config, err := loadCSPConfig([]byte(presetYaml), []byte(customYaml))
	if err != nil {
		t.Error(err)
	}
	assert.Assert(t, cmp.Contains(config.Directives["frame-src"], "'self'"))
	assert.Assert(t, cmp.Contains(config.Directives["frame-src"], "https://embed.diagrams.net/"))
	assert.Assert(t, cmp.Contains(config.Directives["frame-src"], "https://onlyoffice.opencloud.test/"))
	assert.Assert(t, cmp.Contains(config.Directives["frame-src"], "https://collabora.opencloud.test/"))

	assert.Assert(t, cmp.Contains(config.Directives["img-src"], "'self'"))
	assert.Assert(t, cmp.Contains(config.Directives["img-src"], "data:"))
}
