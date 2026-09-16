package opensearch

import (
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// The ext object is dynamic and its siblings are typed by name through dynamic
// templates, so a new property needs no mapping change on our side.

func openExtensionTemplates() []map[string]any {
	template := func(sibling string, m map[string]any) map[string]any {
		return map[string]any{
			"ext_" + strings.TrimPrefix(sibling, "@"): map[string]any{
				"path_match": mapping.OpenExtensionsRoot + ".*",
				"match":      sibling,
				"mapping":    m,
			},
		}
	}
	return []map[string]any{
		template(mapping.SiblingKeyword, map[string]any{"type": "keyword"}),
		template(mapping.SiblingLower, map[string]any{"type": "keyword", "doc_values": false}),
		template(mapping.SiblingNumber, map[string]any{"type": "double"}),
		template(mapping.SiblingBool, map[string]any{"type": "boolean"}),
		template(mapping.SiblingDate, map[string]any{"type": "date"}),
		template(mapping.SiblingGeo, map[string]any{"type": "geo_point"}),
	}
}

func openExtensionProperties() map[string]any {
	return map[string]any{
		mapping.OpenExtensionsStoredField: map[string]any{"type": "object", "enabled": false},
		mapping.OpenExtensionsRoot:        map[string]any{"type": "object", "dynamic": true},
	}
}

func addOpenExtensionSource(body map[string]any, values map[string]string) {
	if len(values) == 0 {
		return
	}
	stored := make(map[string]any, len(values))
	for key, raw := range values {
		stored[key] = raw
	}
	body[mapping.OpenExtensionsStoredField] = stored

	tree := map[string]any{}
	for _, leaf := range mapping.OpenExtensionLeaves(values) {
		var value any
		switch leaf.Sibling {
		case mapping.SiblingKeyword, mapping.SiblingLower:
			value = single(leaf.Strings)
		case mapping.SiblingNumber:
			value = single(leaf.Numbers)
		case mapping.SiblingBool:
			value = single(leaf.Bools)
		case mapping.SiblingDate:
			formatted := make([]string, len(leaf.Times))
			for i, t := range leaf.Times {
				formatted[i] = t.UTC().Format(time.RFC3339Nano)
			}
			value = single(formatted)
		case mapping.SiblingGeo:
			value = map[string]any{"lat": leaf.Geo.Latitude, "lon": leaf.Geo.Longitude}
		}
		setPath(tree, strings.Split(strings.TrimPrefix(leaf.Field, mapping.OpenExtensionsRoot+"."), "."), value)
	}
	body[mapping.OpenExtensionsRoot] = tree
}

func single[T any](values []T) any {
	if len(values) == 1 {
		return values[0]
	}
	return values
}

func setPath(tree map[string]any, path []string, value any) {
	for _, segment := range path[:len(path)-1] {
		next, ok := tree[segment].(map[string]any)
		if !ok {
			next = map[string]any{}
			tree[segment] = next
		}
		tree = next
	}
	tree[path[len(path)-1]] = value
}
