package binder

import (
	"testing"
	"testing/fstest"

	"gotest.tools/v3/assert"
)

type TestConfig struct {
	A string `yaml:"a"`
	B string `yaml:"b"`
	C string `yaml:"c"`
}

func TestBindSourcesToStructs(t *testing.T) {
	// setup test env
	yaml := `
a: "${FOO_VAR|no-foo}"
b: "${BAR_VAR|no-bar}"
c: "${CODE_VAR|code}"
`
	filePath := "etc/opencloud/foo.yaml"
	fs := fstest.MapFS{
		filePath: {Data: []byte(yaml)},
	}
	// perform test
	c := TestConfig{}
	err := BindSourcesToStructsFS(fs, filePath, "foo", &c)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, c.A, "no-foo")
	assert.Equal(t, c.B, "no-bar")
	assert.Equal(t, c.C, "code")
}

func TestBindSourcesToStructs_UnknownFile(t *testing.T) {
	// setup test env
	filePath := "etc/opencloud/foo.yaml"
	fs := fstest.MapFS{}
	// perform test
	c := TestConfig{}
	err := BindSourcesToStructsFS(fs, filePath, "foo", &c)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, c.A, "")
	assert.Equal(t, c.B, "")
	assert.Equal(t, c.C, "")
}
