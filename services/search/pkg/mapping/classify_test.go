package mapping

import (
	"encoding/json"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Classify", func() {
	code := `{
		"Name": {"type": "keyword"},
		"Size": {"type": "long"},
		"photo": {"properties": {"cameraMake": {"type": "keyword"}, "cameraModel": {"type": "keyword"}}}
	}`

	parse := func(s string) map[string]any {
		var m map[string]any
		Expect(json.Unmarshal([]byte(s), &m)).To(Succeed())
		return m
	}

	hasData := func(fields ...string) func(string) bool {
		return func(path string) bool { return slices.Contains(fields, path) }
	}

	// setup produces the (stored, code, dataFields) arguments for one case.
	type setup func() (stored, codeSchema map[string]any, dataFields func(string) bool)

	substrings := func(ss []string) []any {
		out := make([]any, len(ss))
		for i, s := range ss {
			out[i] = ContainSubstring(s)
		}
		return out
	}
	fields := func(ss []string) []any {
		out := make([]any, len(ss))
		for i, s := range ss {
			out[i] = s
		}
		return out
	}

	// newFields/reasons are asserted only when non-nil; an empty slice asserts
	// "none".
	DescribeTable("verdict",
		func(s setup, verdict Verdict, newFields, reasons []string) {
			stored, codeSchema, dataFields := s()

			c := Classify(stored, codeSchema, dataFields)
			Expect(c.Verdict).To(Equal(verdict))
			if newFields != nil {
				Expect(c.NewFields).To(ConsistOf(fields(newFields)...))
			}
			if reasons != nil {
				Expect(c.Reasons).To(ConsistOf(substrings(reasons)...))
			}
		},
		Entry("identical schemas are equal",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				return parse(code), parse(code), nil
			}), VerdictEqual, []string{}, []string{}),

		Entry("a new top-level field is additive",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				stored := parse(code)
				delete(stored, "Size")
				return stored, parse(code), nil
			}), VerdictAdditive, []string{"Size"}, nil),

		Entry("a new nested field is additive",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				stored := parse(code)
				delete(stored["photo"].(map[string]any)["properties"].(map[string]any), "cameraMake")
				return stored, parse(code), nil
			}), VerdictAdditive, []string{"photo.cameraMake"}, nil),

		Entry("a new subtree lists every leaf",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				stored := parse(code)
				delete(stored, "photo")
				return stored, parse(code), nil
			}), VerdictAdditive, []string{"photo.cameraMake", "photo.cameraModel"}, nil),

		Entry("a new field that already holds data is breaking",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				stored := parse(code)
				delete(stored, "Size")
				return stored, parse(code), hasData("Size")
			}), VerdictBreaking, nil, []string{"Size"}),

		Entry("a new nested field that already holds data is breaking",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				stored := parse(code)
				delete(stored["photo"].(map[string]any)["properties"].(map[string]any), "cameraMake")
				return stored, parse(code), hasData("photo.cameraMake")
			}), VerdictBreaking, nil, []string{"photo.cameraMake"}),

		Entry("a new subtree with data below it is breaking",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				stored := parse(code)
				delete(stored, "photo")
				return stored, parse(code), hasData("photo")
			}), VerdictBreaking, nil, []string{"photo"}),

		Entry("a changed field definition is breaking",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				stored := parse(code)
				stored["Size"].(map[string]any)["type"] = "keyword"
				return stored, parse(code), nil
			}), VerdictBreaking, nil, []string{"Size"}),

		Entry("a field removed from the code schema is breaking",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				reduced := parse(code)
				delete(reduced, "Size")
				return parse(code), reduced, nil
			}), VerdictBreaking, nil, []string{"removed or renamed"}),

		Entry("a changed object attribute is breaking",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				stored := parse(code)
				stored["photo"].(map[string]any)["dynamic"] = true
				return stored, parse(code), nil
			}), VerdictBreaking, nil, []string{"dynamic"}),

		Entry("breaking wins over additive",
			setup(func() (map[string]any, map[string]any, func(string) bool) {
				stored := parse(code)
				delete(stored, "Size")
				stored["Name"].(map[string]any)["type"] = "text"
				return stored, parse(code), nil
			}), VerdictBreaking, []string{"Size"}, []string{"Name"}),
	)
})
