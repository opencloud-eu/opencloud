package envutil

import (
	"os"

	"github.com/gookit/goutil/internal/comfunc"
)

// SetEnvMap set multi ENV(string-map) to os
func SetEnvMap(mp map[string]string) {
	for key, value := range mp {
		_ = os.Setenv(key, value)
	}
}

// SetEnvs set multi k-v ENV pairs to os
func SetEnvs(kvPairs ...string) {
	if len(kvPairs)%2 == 1 {
		panic("envutil.SetEnvs: odd argument count")
	}

	for i := 0; i < len(kvPairs); i += 2 {
		_ = os.Setenv(kvPairs[i], kvPairs[i+1])
	}
}

// UnsetEnvs from os
func UnsetEnvs(keys ...string) {
	for _, key := range keys {
		_ = os.Unsetenv(key)
	}
}

// LoadText parse multiline text to ENV. Can use to load .env file contents.
//
// Usage:
// 	envutil.LoadText(fsutil.ReadFile(".env"))
func LoadText(text string) {
	envMp := SplitText2map(text)
	for key, value := range envMp {
		_ = os.Setenv(key, value)
	}
}

// LoadString set line to ENV. e.g.: KEY=VALUE
func LoadString(line string) bool {
	k, v := comfunc.SplitLineToKv(line, "=")
	if len(k) > 0 {
		return os.Setenv(k, v) == nil
	}
	return false
}
