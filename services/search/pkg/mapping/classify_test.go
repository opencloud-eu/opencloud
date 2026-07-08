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

	It("classifies identical schemas as equal", func() {
		c := Classify(parse(code), parse(code), nil)
		Expect(c.Verdict).To(Equal(VerdictEqual))
		Expect(c.NewFields).To(BeEmpty())
		Expect(c.Reasons).To(BeEmpty())
	})

	It("classifies a new top-level field as additive", func() {
		stored := parse(code)
		delete(stored, "Size")

		c := Classify(stored, parse(code), nil)
		Expect(c.Verdict).To(Equal(VerdictAdditive))
		Expect(c.NewFields).To(ConsistOf("Size"))
		Expect(c.Reasons).To(BeEmpty())
	})

	It("classifies a new nested field as additive", func() {
		stored := parse(code)
		delete(stored["photo"].(map[string]any)["properties"].(map[string]any), "cameraMake")

		c := Classify(stored, parse(code), nil)
		Expect(c.Verdict).To(Equal(VerdictAdditive))
		Expect(c.NewFields).To(ConsistOf("photo.cameraMake"))
	})

	It("lists every leaf of a new subtree", func() {
		stored := parse(code)
		delete(stored, "photo")

		c := Classify(stored, parse(code), nil)
		Expect(c.Verdict).To(Equal(VerdictAdditive))
		Expect(c.NewFields).To(ConsistOf("photo.cameraMake", "photo.cameraModel"))
	})

	It("breaks when a new field already has data in the index", func() {
		stored := parse(code)
		delete(stored, "Size")

		c := Classify(stored, parse(code), hasData("Size"))
		Expect(c.Verdict).To(Equal(VerdictBreaking))
		Expect(c.Reasons).To(ConsistOf(ContainSubstring("Size")))
	})

	It("breaks when a new nested field already has data in the index", func() {
		stored := parse(code)
		delete(stored["photo"].(map[string]any)["properties"].(map[string]any), "cameraMake")

		c := Classify(stored, parse(code), hasData("photo.cameraMake"))
		Expect(c.Verdict).To(Equal(VerdictBreaking))
		Expect(c.Reasons).To(ConsistOf(ContainSubstring("photo.cameraMake")))
	})

	It("breaks when a new subtree already has data below it", func() {
		stored := parse(code)
		delete(stored, "photo")

		// the callback is consulted with the subtree root
		c := Classify(stored, parse(code), hasData("photo"))
		Expect(c.Verdict).To(Equal(VerdictBreaking))
		Expect(c.Reasons).To(ConsistOf(ContainSubstring("photo")))
	})

	It("breaks on a changed field definition", func() {
		stored := parse(code)
		stored["Size"].(map[string]any)["type"] = "keyword"

		c := Classify(stored, parse(code), nil)
		Expect(c.Verdict).To(Equal(VerdictBreaking))
		Expect(c.Reasons).To(ConsistOf(ContainSubstring("Size")))
	})

	It("breaks on a field that was removed from the code schema", func() {
		reduced := parse(code)
		delete(reduced, "Size")

		c := Classify(parse(code), reduced, nil)
		Expect(c.Verdict).To(Equal(VerdictBreaking))
		Expect(c.Reasons).To(ConsistOf(ContainSubstring("removed or renamed")))
	})

	It("breaks on a changed object attribute", func() {
		stored := parse(code)
		stored["photo"].(map[string]any)["dynamic"] = true

		c := Classify(stored, parse(code), nil)
		Expect(c.Verdict).To(Equal(VerdictBreaking))
		Expect(c.Reasons).To(ConsistOf(ContainSubstring("dynamic")))
	})

	It("lets breaking win over additive", func() {
		stored := parse(code)
		delete(stored, "Size")
		stored["Name"].(map[string]any)["type"] = "text"

		c := Classify(stored, parse(code), nil)
		Expect(c.Verdict).To(Equal(VerdictBreaking))
		Expect(c.NewFields).To(ConsistOf("Size"))
		Expect(c.Reasons).To(ConsistOf(ContainSubstring("Name")))
	})
})
