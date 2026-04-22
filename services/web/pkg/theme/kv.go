package theme

import (
	"bytes"
	"encoding/json"
	"strings"

	"dario.cat/mergo"
	"github.com/spf13/afero"
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

// PatchKV injects the given values into to v.
func PatchKV(v map[string]any, values KV) KV {
	if v == nil {
		v = KV{}
	}
	for k, val := range values {
		t := v
		path := strings.Split(k, ".")
		for i, p := range path {
			if i == len(path)-1 {
				switch val {
				// if the value is nil, we delete the key
				case nil:
					delete(t, p)
				default:
					t[p] = val
				}
				break
			}

			if _, ok := t[p]; !ok {
				t[p] = map[string]any{}
			}

			t = t[p].(map[string]any)
		}
	}
	return v
}

// LoadKV loads a key-value map from the given file system.
func LoadKV(fsys afero.Fs, p string) (KV, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var kv KV
	err = json.NewDecoder(f).Decode(&kv)
	if err != nil {
		return nil, err
	}

	return kv, nil
}

// WriteKV writes the given key-value map to the file system.
func WriteKV(fsys afero.Fs, p string, kv KV) error {
	data, err := json.Marshal(kv)
	if err != nil {
		return err
	}

	return afero.WriteReader(fsys, p, bytes.NewReader(data))
}

// UpdateKV updates the key-value map at the given path with the given values.
func UpdateKV(fsys afero.Fs, p string, values KV) error {
	var kv KV

	existing, err := LoadKV(fsys, p)
	if err == nil {
		kv = existing
	}

	kv = PatchKV(kv, values)

	return WriteKV(fsys, p, kv)
}
