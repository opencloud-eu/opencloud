package mapping

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("extension fields", func() {
	key := func(name, property string) string {
		return "http://opencloud.eu/ns/extensions/" + name + "/" + property
	}

	It("names a sibling and resolves the KQL spelling", func() {
		Expect(OpenExtensionField("com.example.project", "priority", SiblingNumber)).To(Equal("ext.com.example.project.priority.@number"))

		field, ok := OpenExtensionFieldFromQuery("extensions.com.example.project.priority", SiblingNumber)
		Expect(ok).To(BeTrue())
		Expect(field).To(Equal("ext.com.example.project.priority.@number"))

		field, ok = OpenExtensionFieldFromQuery("Extensions.com.example.project.Priority", SiblingLower)
		Expect(ok).To(BeTrue(), "the prefix is case-insensitive, the rest is taken as written")
		Expect(field).To(Equal("ext.com.example.project.Priority.@lower"))
	})

	It("rejects keys that are not an extension property", func() {
		for _, key := range []string{"Name", "extensions", "extensions.", "extensions.priority", "extensions.project.priority", "extensions.com.example.", "extension.com.example.x"} {
			Expect(IsOpenExtensionQueryField(key)).To(BeFalse(), key)
		}
		Expect(IsOpenExtensionQueryField("extensions.com.example.priority")).To(BeTrue())
	})

	It("flattens stored extension properties into typed leaves", func() {
		const project = "com.example.project"
		leaves := OpenExtensionLeaves(map[string]string{
			key(project, "state"):    "s:Open",
			key(project, "priority"): "n:3",
			key(project, "done"):     "b:false",
			key(project, "tags"):     `S:["A","b"]`,
			key(project, "due"):      "d:2026-10-01T00:00:00Z",
			key(project, "site"):     "g:52.5,13.4",
			key(project, "broken"):   "n:not a number",
			"tags":                   "a,b",
		})

		byField := map[string]OpenExtensionLeaf{}
		for _, l := range leaves {
			byField[l.Field] = l
		}
		Expect(byField).To(HaveLen(8))
		Expect(byField["ext.com.example.project.state.@keyword"].Strings).To(Equal([]string{"Open"}))
		Expect(byField["ext.com.example.project.state.@lower"].Strings).To(Equal([]string{"open"}))
		Expect(byField["ext.com.example.project.tags.@keyword"].Strings).To(Equal([]string{"A", "b"}))
		Expect(byField["ext.com.example.project.tags.@lower"].Strings).To(Equal([]string{"a", "b"}))
		Expect(byField["ext.com.example.project.priority.@number"].Numbers).To(Equal([]float64{3}))
		Expect(byField["ext.com.example.project.done.@bool"].Bools).To(Equal([]bool{false}))
		Expect(byField["ext.com.example.project.due.@date"].Times).To(Equal([]time.Time{time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}))
		Expect(byField["ext.com.example.project.site.@geo"].Geo.Latitude).To(Equal(52.5))
		Expect(byField["ext.com.example.project.site.@geo"].Geo.Longitude).To(Equal(13.4))

		Expect(leaves[0].Field < leaves[len(leaves)-1].Field).To(BeTrue(), "leaves are sorted by field")
	})

	It("indexes a property under the sibling of its current kind only", func() {
		asNumber := OpenExtensionLeaves(map[string]string{key("x.y", "p"): "n:3"})
		asString := OpenExtensionLeaves(map[string]string{key("x.y", "p"): "s:3"})
		Expect(asNumber).To(HaveLen(1))
		Expect(asNumber[0].Field).To(Equal("ext.x.y.p.@number"))
		Expect(asString).To(HaveLen(2))
		Expect(asString[0].Field).To(Equal("ext.x.y.p.@keyword"))
		Expect(asString[1].Field).To(Equal("ext.x.y.p.@lower"))
	})
})
