package theme

import (
	"encoding/json"
	"io/fs"

	"dario.cat/mergo"
)

// KV is a generic key-value map.
type KV map[string]any

// MergeKV merges the given key-value maps.
func MergeKV(values ...KV) (KV, error) {
	var kv KV

	for _, v := range values {
		err := mergo.Merge(&kv, v, mergo.WithOverride)
		if err != nil {
			return nil, err
		}
	}

	return kv, nil
}

// LoadKV loads a key-value map from the given file system.
func LoadKV(fsys fs.FS, p string) (KV, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	var kv KV
	err = json.NewDecoder(f).Decode(&kv)
	if err != nil {
		return nil, err
	}

	return kv, nil
}
