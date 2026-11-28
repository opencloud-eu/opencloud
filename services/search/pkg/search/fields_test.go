package search_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = Describe("FieldType", func() {
	DescribeTable("reports the mapping type of a field",
		func(field, want string) {
			Expect(search.FieldType(field)).To(Equal(want))
		},
		Entry("embedded document field", "Size", mapping.TypeNumeric),
		Entry("resource field", "Type", mapping.TypeNumeric),
		Entry("keyword", "Name", mapping.TypeKeyword),
		Entry("overridden type", "Path", mapping.TypePath),
		Entry("full text", "Content", mapping.TypeFulltext),
		Entry("time", "Mtime", mapping.TypeDatetime),
		Entry("facet keyword", "audio.artist", mapping.TypeKeyword),
		Entry("facet int", "audio.year", mapping.TypeNumeric),
		Entry("facet bool", "audio.hasDrm", mapping.TypeBool),
		Entry("facet float", "photo.focalLength", mapping.TypeNumeric),
		Entry("facet time", "photo.takenDateTime", mapping.TypeDatetime),
		Entry("facet object", "audio", mapping.TypeObject),
		Entry("geopoint", "location", mapping.TypeGeopoint),
		Entry("unknown", "audio.nonexistent", ""),
	)
})

var _ = Describe("AggregatableFieldType", func() {
	DescribeTable("leaves out the internal fields",
		func(field, want string) {
			Expect(search.AggregatableFieldType(field)).To(Equal(want))
		},
		Entry("name", "Name", mapping.TypeKeyword),
		Entry("size", "Size", mapping.TypeNumeric),
		Entry("modification time", "Mtime", mapping.TypeDatetime),
		Entry("mime type", "MimeType", mapping.TypeKeyword),
		Entry("tags", "Tags", mapping.TypeKeyword),
		Entry("a facet", "audio.artist", mapping.TypeKeyword),
		Entry("who favorited", "Favorites", ""),
		Entry("an id", "ID", ""),
		Entry("the parent id", "ParentID", ""),
		Entry("the space root", "RootID", ""),
		Entry("the path", "Path", ""),
		Entry("the content", "Content", ""),
		Entry("a flag", "Hidden", ""),
		Entry("the resource type", "Type", ""),
		Entry("the title", "Title", ""),
		Entry("the deleted flag", "Deleted", ""),
		Entry("unknown", "audio.nonexistent", ""),
	)
})
