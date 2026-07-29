package mapping

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// ErrManualActionRequired marks schema changes that cannot be applied in place.
var ErrManualActionRequired = errors.New("manual action required")

// ManualActionRequiredError reports a breaking schema change. Because the index
// is versioned by search.SchemaVersion, a released instance never hits this: a
// version bump builds a fresh index. It fires in development when the mapping is
// changed in a breaking way without bumping search.SchemaVersion, so the fix is
// to bump it (or revert the change).
func ManualActionRequiredError(index string, reasons []string) error {
	return fmt.Errorf(
		"%w: the search mapping in code differs from index %s in a breaking way:\n  - %s\n"+
			"bump search.SchemaVersion to build a fresh index, or revert the mapping change",
		ErrManualActionRequired, index, strings.Join(reasons, "\n  - "),
	)
}

type Verdict string

const (
	VerdictEqual    Verdict = "equal"
	VerdictAdditive Verdict = "additive"
	VerdictBreaking Verdict = "breaking"
)

// Classification is the outcome of diffing a stored index schema against the
// schema generated from code.
type Classification struct {
	Verdict Verdict
	// NewFields are dotted paths of fields that only exist in the code schema.
	NewFields []string
	// Reasons are human-readable breaking differences.
	Reasons []string
}

// Classify recursively compares a stored `properties` tree against the one
// generated from code. Both sides must be generic JSON-decoded values, not
// marshaled Go structs. dataFields reports whether the index holds data at or
// below a dotted field path absent from the stored schema (bleve dynamic
// fields); engines without that blind spot pass nil.
func Classify(stored, code map[string]any, dataFields func(path string) bool) Classification {
	c := Classification{Verdict: VerdictEqual}
	classifyProperties(stored, code, dataFields, "", &c)
	if c.Verdict == VerdictEqual && len(c.NewFields) > 0 {
		c.Verdict = VerdictAdditive
	}
	return c
}

func classifyProperties(stored, code map[string]any, dataFields func(string) bool, prefix string, c *Classification) {
	for _, k := range slices.Sorted(maps.Keys(stored)) {
		path := joinPath(prefix, k)
		codeNode, ok := code[k]
		if !ok {
			c.breaking(fmt.Sprintf("field %s exists in the index but not in the code schema (removed or renamed)", path))
			continue
		}
		classifyNode(stored[k], codeNode, dataFields, path, c)
	}

	for _, k := range slices.Sorted(maps.Keys(code)) {
		if _, ok := stored[k]; ok {
			continue
		}
		path := joinPath(prefix, k)
		if dataFields != nil && dataFields(path) {
			c.breaking(fmt.Sprintf("field %s is explicitly mapped now, but the index already holds data that was indexed dynamically for it, of an unknown type", path))
			continue
		}
		c.NewFields = append(c.NewFields, leafPaths(code[k], path)...)
	}
}

func classifyNode(stored, code any, dataFields func(string) bool, path string, c *Classification) {
	storedMap, sOK := stored.(map[string]any)
	codeMap, cOK := code.(map[string]any)
	if !sOK || !cOK {
		if !reflect.DeepEqual(stored, code) {
			c.breaking(fmt.Sprintf("field %s changed: index %s, code %s", path, compactJSON(stored), compactJSON(code)))
		}
		return
	}

	for _, k := range sortedUnionKeys(storedMap, codeMap) {
		if k == "properties" {
			continue
		}
		sv, sHas := storedMap[k]
		cv, cHas := codeMap[k]
		if sHas && cHas && reflect.DeepEqual(sv, cv) {
			continue
		}
		c.breaking(fmt.Sprintf("field %s: %s changed: index %s, code %s", path, k, optJSON(sv, sHas), optJSON(cv, cHas)))
	}

	storedProps, _ := storedMap["properties"].(map[string]any)
	codeProps, _ := codeMap["properties"].(map[string]any)
	if len(storedProps) > 0 || len(codeProps) > 0 {
		classifyProperties(storedProps, codeProps, dataFields, path, c)
	}
}

// leafPaths lists the dotted paths of all leaf fields at or below node.
func leafPaths(node any, path string) []string {
	if nodeMap, ok := node.(map[string]any); ok {
		if props, ok := nodeMap["properties"].(map[string]any); ok && len(props) > 0 {
			var leaves []string
			for _, k := range slices.Sorted(maps.Keys(props)) {
				leaves = append(leaves, leafPaths(props[k], path+"."+k)...)
			}
			return leaves
		}
	}
	return []string{path}
}

func (c *Classification) breaking(reason string) {
	c.Verdict = VerdictBreaking
	c.Reasons = append(c.Reasons, reason)
}

func joinPath(prefix, k string) string {
	if prefix == "" {
		return k
	}
	return prefix + "." + k
}

func sortedUnionKeys(a, b map[string]any) []string {
	keys := slices.Collect(maps.Keys(a))
	for k := range b {
		if _, ok := a[k]; !ok {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

func optJSON(v any, present bool) string {
	if !present {
		return "(unset)"
	}
	return compactJSON(v)
}

func compactJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
