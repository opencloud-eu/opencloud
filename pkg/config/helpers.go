package config

import (
	"io/fs"
	"os"
	"path"
	"strings"

	gofig "github.com/gookit/config/v2"
	gooyaml "github.com/gookit/config/v2/yaml"
	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
)

var (
	// decoderConfigTagName sets the tag name to be used from the config structs
	// currently we only support "yaml" because we only support config loading
	// from yaml files and the yaml parser has no simple way to set a custom tag name to use
	decoderConfigTagName = "yaml"
)

// BindSourcesToStructs assigns any config value from a config file / env variable to struct `dst`.
func BindSourcesToStructs(service string, dst any) error {
	fileSystem := os.DirFS("/")
	filePath := strings.TrimLeft(path.Join(defaults.BaseConfigPath(), service+".yaml"), "/")
	return bindSourcesToStructs(fileSystem, filePath, service, dst)
}

func bindSourcesToStructs(fileSystem fs.FS, filePath, service string, dst any) error {
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

// LocalEndpoint returns the local endpoint for a given protocol and address.
// Use it when configuring the reva runtime to get a service endpoint in the same
// runtime, e.g. a gateway talking to an authregistry service.
func LocalEndpoint(protocol, addr string) string {
	localEndpoint := addr
	switch protocol {
	case "tcp":
		parts := strings.SplitN(addr, ":", 2)
		if len(parts) == 2 {
			localEndpoint = "dns:127.0.0.1:" + parts[1]
		}
	case "unix":
		localEndpoint = "unix:" + addr
	}
	return localEndpoint
}
