package mapping

import (
	"reflect"
	"strings"
)

// addSearchSiblings writes the _lowercase and _words siblings next to their
// base values, for every keyword/path field of t that has them (see
// SearchSiblings).
func addSearchSiblings(m map[string]any, t reflect.Type, overrides map[string]FieldOpts) {
	for key, siblings := range SearchSiblings(t, overrides) {
		parent, leaf, ok := resolveLeaf(m, key)
		if !ok {
			continue
		}
		if siblings.Lowercase {
			addLowercaseSibling(parent, leaf)
		}
		if siblings.Words {
			addWordsSibling(parent, leaf)
		}
	}
}

// Siblings says which search-only siblings a field carries.
type Siblings struct {
	Lowercase bool
	Words     bool
}

// SearchSiblings lists the fields of t (json names, nested as parent.child)
// that carry a _lowercase or _words sibling, from the effective field type and
// the overrides. It is the one place that decides, the renderers, the
// document writer and the query lowering all follow it.
func SearchSiblings(t reflect.Type, overrides map[string]FieldOpts) map[string]Siblings {
	out := map[string]Siblings{}
	for key, goType := range collectFields(t, "") {
		opts := overrides[key]
		eff := opts.Type
		if eff == "" {
			eff = inferType(goType)
		}
		if eff != TypeKeyword && eff != TypePath {
			continue
		}
		siblings := Siblings{
			Lowercase: opts.caseInsensitive(),
			Words:     eff == TypeKeyword && opts.wordBroken(),
		}
		if siblings.Lowercase || siblings.Words {
			out[key] = siblings
		}
	}
	return out
}

// addWordsSibling copies the value to a <leaf>_words sibling; the words
// analyzer does the splitting and lowercasing. No-op for non-strings.
func addWordsSibling(parent map[string]any, leaf string) {
	switch v := parent[leaf].(type) {
	case string, []any, []string:
		parent[leaf+WordsSuffix] = v
	}
}

func resolveLeaf(m map[string]any, dottedPath string) (map[string]any, string, bool) {
	parts := strings.Split(dottedPath, ".")
	parent := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := parent[p].(map[string]any)
		if !ok {
			return nil, "", false
		}
		parent = next
	}
	return parent, parts[len(parts)-1], true
}

// addLowercaseSibling writes a <leaf>_lowercase sibling; no-op for non-strings.
func addLowercaseSibling(parent map[string]any, leaf string) {
	switch v := parent[leaf].(type) {
	case string:
		parent[leaf+LowercaseSuffix] = strings.ToLower(v)
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, strings.ToLower(s))
			}
		}
		parent[leaf+LowercaseSuffix] = out
	case []string:
		out := make([]string, len(v))
		for i, s := range v {
			out[i] = strings.ToLower(s)
		}
		parent[leaf+LowercaseSuffix] = out
	}
}
