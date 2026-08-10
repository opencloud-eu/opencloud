package mapping

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PrepareForIndex casing", func() {
	It("adds lowercased siblings for CaseInsensitive keyword and path fields", func() {
		True := true
		type doc struct {
			Name string   `json:"Name"`
			Path string   `json:"Path"`
			Tags []string `json:"Tags"`
		}
		d := doc{Name: "Report FINAL", Path: "/Foo/Bar", Tags: []string{"Work", "Urgent"}}
		m, err := PrepareForIndex(d, map[string]FieldOpts{
			"Name": {CaseInsensitive: &True},
			"Path": {Type: TypePath, CaseInsensitive: &True},
			"Tags": {CaseInsensitive: &True},
		})
		Expect(err).ToNot(HaveOccurred())

		// Originals stay for the case-preserved base fields and the cascade.
		Expect(m["Name"]).To(Equal("Report FINAL"))
		Expect(m["Path"]).To(Equal("/Foo/Bar"))

		Expect(m["Name_lowercase"]).To(Equal("report final"))
		Expect(m["Path_lowercase"]).To(Equal("/foo/bar"))
		Expect(m["Tags_lowercase"]).To(Equal([]any{"work", "urgent"}))
	})

	It("writes no sibling without CaseInsensitive", func() {
		type doc struct {
			ID string `json:"ID"`
		}
		m, err := PrepareForIndex(doc{ID: "ABC"}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(m).ToNot(HaveKey("ID" + LowercaseSuffix))
	})

	It("writes an empty sibling for an empty array, like a non-empty one", func() {
		True := true
		type doc struct {
			Tags []string `json:"Tags"`
		}
		m, err := PrepareForIndex(doc{Tags: []string{}}, map[string]FieldOpts{
			"Tags": {CaseInsensitive: &True},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(m).To(HaveKey("Tags" + LowercaseSuffix))
		Expect(m["Tags"+LowercaseSuffix]).To(BeEmpty())
	})
})
