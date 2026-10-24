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

	It("copies the value to a words sibling by default", func() {
		type doc struct {
			Name string `json:"Name"`
		}
		m, err := PrepareForIndex(doc{Name: "Report FINAL"}, nil)
		Expect(err).ToNot(HaveOccurred())
		// the analyzer splits and lowercases, the value goes over as is
		Expect(m["Name"]).To(Equal("Report FINAL"))
		Expect(m["Name_lowercase"]).To(Equal("report final"))
		Expect(m["Name_words"]).To(Equal("Report FINAL"))
	})

	It("writes the lowercased sibling by default", func() {
		type doc struct {
			ID string `json:"ID"`
		}
		m, err := PrepareForIndex(doc{ID: "ABC"}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(m["ID"+LowercaseSuffix]).To(Equal("abc"))
	})

	It("writes no sibling with CaseInsensitive off", func() {
		False := false
		type doc struct {
			ID string `json:"ID"`
		}
		m, err := PrepareForIndex(doc{ID: "ABC"}, map[string]FieldOpts{"ID": {CaseInsensitive: &False}})
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
