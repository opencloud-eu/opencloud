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

// ManualActionRequiredError builds the operator-facing error for a breaking
// schema change, shared by both engines. index is the index name (OpenSearch)
// or path (bleve).
func ManualActionRequiredError(index string, reasons []string) error {
	return fmt.Errorf(
		"%w: search index %s was built with a different schema (%s). "+
			"There is no in-place migration: stop the service, delete %s, "+
			"start the service (an empty index with the new schema is created), "+
			"then rebuild the content by running: opencloud search index --all-spaces. "+
			"To bring the instance up without search until a maintenance window, "+
			"set OC_EXCLUDE_RUN_SERVICES=search; until the service is back, search "+
			"and features built on it (e.g. the search bar and the tag list) are "+
			"unavailable",
		ErrManualActionRequired, index, strings.Join(reasons, "; "), index,
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
// generated from code. Both sides must be generic JSON-decoded values
// (map[string]any, []any, float64), not marshaled Go structs, so values
// compare structurally.
//
// dataFields reports whether the index holds data at or below a dotted field
// path even though it is absent from the stored schema (bleve indexes dynamic
// fields without a schema trace). Engines that record dynamic fields in the
// live schema (OpenSearch) pass nil.
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
			c.breaking(fmt.Sprintf("field %s is new in the code schema but the index already contains data for it (previously indexed dynamically)", path))
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
