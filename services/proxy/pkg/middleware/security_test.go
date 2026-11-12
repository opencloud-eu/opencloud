package middleware

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"

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

func writeCSPFile(t *testing.T, path string, frameSrc string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	content := "directives:\n  frame-src:\n    - '" + frameSrc + "'\n"
	assert.NilError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestLoadCSPConfigFromLocations(t *testing.T) {
	dir := t.TempDir()
	writeCSPFile(t, filepath.Join(dir, "csp.yaml"), "https://global.test/")
	writeCSPFile(t, filepath.Join(dir, "apps", "a", "csp.yaml"), "https://a.test/")
	writeCSPFile(t, filepath.Join(dir, "apps", "b", "csp.yaml"), "https://b.test/")

	t.Run("single path", func(t *testing.T) {
		cspConfig, err := LoadCSPConfig(&config.Config{CSPConfigFileLocation: []string{filepath.Join(dir, "csp.yaml")}})
		assert.NilError(t, err)
		assert.Assert(t, cmp.Contains(cspConfig.Directives["frame-src"], "'self'"))
		assert.Assert(t, cmp.Contains(cspConfig.Directives["frame-src"], "https://global.test/"))
	})

	t.Run("list of path and glob", func(t *testing.T) {
		location := []string{filepath.Join(dir, "csp.yaml"), filepath.Join(dir, "apps", "*", "csp.yaml")}
		cspConfig, err := LoadCSPConfig(&config.Config{CSPConfigFileLocation: location})
		assert.NilError(t, err)
		frameSrc := cspConfig.Directives["frame-src"]
		assert.Assert(t, cmp.Contains(frameSrc, "'self'"))
		assert.DeepEqual(t, frameSrc[len(frameSrc)-3:], []string{"https://global.test/", "https://a.test/", "https://b.test/"})
	})

	t.Run("glob without matches is ignored", func(t *testing.T) {
		cspConfig, err := LoadCSPConfig(&config.Config{CSPConfigFileLocation: []string{filepath.Join(dir, "none", "*", "csp.yaml")}})
		assert.NilError(t, err)
		assert.Assert(t, cmp.Contains(cspConfig.Directives["frame-src"], "'self'"))
	})

	t.Run("missing path fails", func(t *testing.T) {
		_, err := LoadCSPConfig(&config.Config{CSPConfigFileLocation: []string{filepath.Join(dir, "missing.yaml")}})
		assert.Assert(t, os.IsNotExist(err))
	})

	t.Run("override replaces the default and ignores locations", func(t *testing.T) {
		cspConfig, err := LoadCSPConfig(&config.Config{
			CSPConfigFileLocation:         []string{filepath.Join(dir, "apps", "a", "csp.yaml")},
			CSPConfigFileOverrideLocation: []string{filepath.Join(dir, "csp.yaml")},
		})
		assert.NilError(t, err)
		assert.DeepEqual(t, cspConfig.Directives["frame-src"], []string{"https://global.test/"})
	})

	t.Run("override list of path and glob", func(t *testing.T) {
		location := []string{filepath.Join(dir, "csp.yaml"), filepath.Join(dir, "apps", "*", "csp.yaml")}
		cspConfig, err := LoadCSPConfig(&config.Config{CSPConfigFileOverrideLocation: location})
		assert.NilError(t, err)
		assert.DeepEqual(t, cspConfig.Directives["frame-src"], []string{"https://global.test/", "https://a.test/", "https://b.test/"})
	})

	t.Run("override without any file fails", func(t *testing.T) {
		_, err := LoadCSPConfig(&config.Config{CSPConfigFileOverrideLocation: []string{filepath.Join(dir, "none", "*", "csp.yaml")}})
		assert.ErrorContains(t, err, "no CSP configuration file found")
	})
}
