package middleware

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	gofig "github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
	yamlv3 "gopkg.in/yaml.v3"
)

// LoadCSPConfig loads CSP header configuration from a yaml file.
func LoadCSPConfig(proxyCfg *config.Config) (*config.CSP, error) {
	yamlContent, customYamlContents, err := loadCSPYaml(proxyCfg)
	if err != nil {
		return nil, err
	}
	return loadCSPConfig(yamlContent, customYamlContents...)
}

// loadCSPConfig merges the custom yaml contents into the preset, in the given order.
func loadCSPConfig(presetYamlContent []byte, customYamlContents ...[]byte) (*config.CSP, error) {
	// substitute env vars and load to struct
	cspLoader := gofig.NewWithOptions("csp", gofig.ParseEnv)
	cspLoader.AddDriver(yaml.Driver)

	presetMap := map[string]any{}
	err := yamlv3.Unmarshal(presetYamlContent, &presetMap)
	if err != nil {
		return nil, err
	}
	mergedMap := presetMap
	for _, customYamlContent := range customYamlContents {
		customMap := map[string]any{}
		err = yamlv3.Unmarshal(customYamlContent, &customMap)
		if err != nil {
			return nil, err
		}
		mergedMap = deepMerge(mergedMap, customMap)
	}
	mergedYamlContent, err := yamlv3.Marshal(mergedMap)
	if err != nil {
		return nil, err
	}

	err = cspLoader.LoadSources("yaml", mergedYamlContent)
	if err != nil {
		return nil, err
	}

	// read yaml
	cspConfig := config.CSP{}
	err = cspLoader.BindStruct("", &cspConfig)
	if err != nil {
		return nil, err
	}

	return &cspConfig, nil
}

// deepMerge recursively merges map2 into map1.
// - nested maps are merged recursively
// - slices are concatenated, preserving order and avoiding duplicates
// - scalar or type-mismatched values from map2 overwrite map1
func deepMerge(map1, map2 map[string]any) map[string]any {
	if map1 == nil {
		out := make(map[string]any, len(map2))
		for k, v := range map2 {
			out[k] = v
		}
		return out
	}

	for k, v2 := range map2 {
		if v1, ok := map1[k]; ok {
			// both maps -> recurse
			if m1, ok1 := v1.(map[string]any); ok1 {
				if m2, ok2 := v2.(map[string]any); ok2 {
					map1[k] = deepMerge(m1, m2)
					continue
				}
			}

			// both slices -> merge unique
			if s1, ok1 := v1.([]any); ok1 {
				if s2, ok2 := v2.([]any); ok2 {
					merged := append([]any{}, s1...)
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

func sliceContains(slice []any, val any) bool {
	for _, v := range slice {
		if reflect.DeepEqual(v, val) {
			return true
		}
	}
	return false
}

func loadCSPYaml(proxyCfg *config.Config) ([]byte, [][]byte, error) {
	if len(proxyCfg.CSPConfigFileOverrideLocation) > 0 {
		overrideCSPYamls, err := readCSPConfigFiles(proxyCfg.CSPConfigFileOverrideLocation)
		if err != nil {
			return nil, nil, err
		}
		if len(overrideCSPYamls) == 0 {
			return nil, nil, fmt.Errorf("no CSP configuration file found for override location %q", strings.Join(proxyCfg.CSPConfigFileOverrideLocation, ","))
		}
		return overrideCSPYamls[0], overrideCSPYamls[1:], nil
	}
	if len(proxyCfg.CSPConfigFileLocation) == 0 {
		return []byte(config.DefaultCSPConfig), nil, nil
	}
	customCSPYamls, err := readCSPConfigFiles(proxyCfg.CSPConfigFileLocation)
	return []byte(config.DefaultCSPConfig), customCSPYamls, err
}

func readCSPConfigFiles(locations []string) ([][]byte, error) {
	files, err := resolveCSPConfigFiles(locations)
	if err != nil {
		return nil, err
	}
	yamlContents := make([][]byte, 0, len(files))
	for _, file := range files {
		yamlContent, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		yamlContents = append(yamlContents, yamlContent)
	}
	return yamlContents, nil
}

func resolveCSPConfigFiles(locations []string) ([]string, error) {
	files := []string{}
	for _, location := range locations {
		// keep plain paths so that a missing file still fails
		if !strings.ContainsAny(location, `*?[`) {
			files = append(files, location)
			continue
		}
		matches, err := filepath.Glob(location)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	return files, nil
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
