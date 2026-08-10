package mapping

import "strings"

func addLowercaseSiblings(m map[string]any, overrides map[string]FieldOpts) {
	for key, opts := range overrides {
		if !opts.caseInsensitive() || !isCasedType(opts) {
			continue
		}
		parent, leaf, ok := resolveLeaf(m, key)
		if !ok {
			continue
		}
		addLowercaseSibling(parent, leaf)
	}
}

func isCasedType(opts FieldOpts) bool {
	return opts.Type == "" || opts.Type == TypeKeyword || opts.Type == TypePath
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
