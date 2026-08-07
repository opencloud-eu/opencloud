// Package binder binds OpenCloud yaml config files to config structs, as a leaf
// package that avoids importing the aggregate service config in pkg/config.
package binder

import (
	"io/fs"
	"os"
	"path"
	"strings"

	gofig "github.com/gookit/config/v2"
	gooyaml "github.com/gookit/config/v2/yaml"
	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
)

// decoderConfigTagName sets the tag name to be used from the config structs
// currently we only support "yaml" because we only support config loading
// from yaml files and the yaml parser has no simple way to set a custom tag name to use
var decoderConfigTagName = "yaml"

// BindSourcesToStructs assigns any config value from a config file / env variable to struct `dst`.
func BindSourcesToStructs(service string, dst any) error {
	fileSystem := os.DirFS("/")
	filePath := strings.TrimLeft(path.Join(defaults.BaseConfigPath(), service+".yaml"), "/")
	return BindSourcesToStructsFS(fileSystem, filePath, service, dst)
}

// BindSourcesToStructsFS is like BindSourcesToStructs but reads from the given fs.FS and path.
func BindSourcesToStructsFS(fileSystem fs.FS, filePath, service string, dst any) error {
	cnf := gofig.NewWithOptions(service)
	cnf.WithOptions(func(options *gofig.Options) {
		options.ParseEnv = true
		options.DecoderConfig.TagName = decoderConfigTagName
	})
	cnf.AddDriver(gooyaml.Driver)

	yamlContent, err := fs.ReadFile(fileSystem, filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}
	_ = cnf.LoadSources("yaml", yamlContent)

	err = cnf.BindStruct("", &dst)
	if err != nil {
		return err
	}

	return nil
}
