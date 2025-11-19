package middleware

import (
	"net/http"
	"os"
	"reflect"

	gofig "github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
	yamlv3 "gopkg.in/yaml.v3"
)

// LoadCSPConfig loads CSP header configuration from a yaml file.
func LoadCSPConfig(proxyCfg *config.Config) (*config.CSP, error) {
	yamlContent, customYamlContent, err := loadCSPYaml(proxyCfg)
	if err != nil {
		return nil, err
	}
	return loadCSPConfig(yamlContent, customYamlContent)
}

// LoadCSPConfig loads CSP header configuration from a yaml file.
func loadCSPConfig(presetYamlContent, customYamlContent []byte) (*config.CSP, error) {
	// substitute env vars and load to struct
	gofig.WithOptions(gofig.ParseEnv)
	gofig.AddDriver(yaml.Driver)

	presetMap := map[string]interface{}{}
	err := yamlv3.Unmarshal(presetYamlContent, &presetMap)
	if err != nil {
		return nil, err
	}
	customMap := map[string]interface{}{}
	err = yamlv3.Unmarshal(customYamlContent, &customMap)
	if err != nil {
		return nil, err
	}
	mergedMap := deepMerge(presetMap, customMap)
	mergedYamlContent, err := yamlv3.Marshal(mergedMap)
	if err != nil {
		return nil, err
	}

	err = gofig.LoadSources("yaml", mergedYamlContent)
	if err != nil {
		return nil, err
	}

	// read yaml
	cspConfig := config.CSP{}
	err = gofig.BindStruct("", &cspConfig)
	if err != nil {
		return nil, err
	}

	return &cspConfig, nil
}

// deepMerge recursively merges map2 into map1.
// - nested maps are merged recursively
// - slices are concatenated, preserving order and avoiding duplicates
// - scalar or type-mismatched values from map2 overwrite map1
func deepMerge(map1, map2 map[string]interface{}) map[string]interface{} {
	if map1 == nil {
		out := make(map[string]interface{}, len(map2))
		for k, v := range map2 {
			out[k] = v
		}
		return out
	}

	for k, v2 := range map2 {
		if v1, ok := map1[k]; ok {
			// both maps -> recurse
			if m1, ok1 := v1.(map[string]interface{}); ok1 {
				if m2, ok2 := v2.(map[string]interface{}); ok2 {
					map1[k] = deepMerge(m1, m2)
					continue
				}
			}

			// both slices -> merge unique
			if s1, ok1 := v1.([]interface{}); ok1 {
				if s2, ok2 := v2.([]interface{}); ok2 {
					merged := append([]interface{}{}, s1...)
					for _, item := range s2 {
						if !sliceContains(merged, item) {
							merged = append(merged, item)
						}
					}
					map1[k] = merged
					continue
				}
				// s1 is slice, v2 single -> append if missing
				if !sliceContains(s1, v2) {
					map1[k] = append(s1, v2)
				}
				continue
			}

			// default: overwrite
			map1[k] = v2
		} else {
			// new key -> just set
			map1[k] = v2
		}
	}

	return map1
}

func sliceContains(slice []interface{}, val interface{}) bool {
	for _, v := range slice {
		if reflect.DeepEqual(v, val) {
			return true
		}
	}
	return false
}

func loadCSPYaml(proxyCfg *config.Config) ([]byte, []byte, error) {
	if proxyCfg.CSPConfigFileOverrideLocation != "" {
		overrideCSPYaml, err := os.ReadFile(proxyCfg.CSPConfigFileOverrideLocation)
		return overrideCSPYaml, []byte{}, err
	}
	if proxyCfg.CSPConfigFileLocation == "" {
		return []byte(config.DefaultCSPConfig), nil, nil
	}
	customCSPYaml, err := os.ReadFile(proxyCfg.CSPConfigFileLocation)
	return []byte(config.DefaultCSPConfig), customCSPYaml, err
}

// Security is a middleware to apply security relevant http headers like CSP.
func Security(cspConfig *config.CSP) func(h http.Handler) http.Handler {
	cspBuilder := cspbuilder.Builder{
		Directives: cspConfig.Directives,
	}

	secureMiddleware := secure.New(secure.Options{
		BrowserXssFilter:             true,
		ContentSecurityPolicy:        cspBuilder.MustBuild(),
		ContentTypeNosniff:           true,
		CustomFrameOptionsValue:      "SAMEORIGIN",
		FrameDeny:                    true,
		ReferrerPolicy:               "strict-origin-when-cross-origin",
		STSSeconds:                   315360000,
		STSPreload:                   true,
		PermittedCrossDomainPolicies: "none",
		RobotTag:                     "none",
	})
	return func(next http.Handler) http.Handler {
		return secureMiddleware.Handler(next)
	}
}
