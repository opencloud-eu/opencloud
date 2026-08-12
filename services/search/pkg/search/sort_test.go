package search_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/types/known/timestamppb"

	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = Describe("SortIndexField", func() {
	DescribeTable("maps graph field names to index fields",
		func(name string, wantField string, wantOK bool) {
			field, ok := search.SortIndexField(name)
			Expect(ok).To(Equal(wantOK))
			Expect(field).To(Equal(wantField))
		},
		// top-level fields are exposed under their graph names
		Entry("name", "name", "Name", true),
		Entry("size", "size", "Size", true),
		Entry("lastModifiedDateTime", "lastModifiedDateTime", "Mtime", true),
		Entry("mimeType", "mimeType", "MimeType", true),
		// facet fields keep their dotted names
		Entry("photo.takenDateTime", "photo.takenDateTime", "photo.takenDateTime", true),
		Entry("photo.iso", "photo.iso", "photo.iso", true),
		Entry("photo.cameraModel", "photo.cameraModel", "photo.cameraModel", true),
		Entry("audio.artist", "audio.artist", "audio.artist", true),
		Entry("audio.year", "audio.year", "audio.year", true),
		Entry("image.width", "image.width", "image.width", true),
		Entry("location.latitude", "location.latitude", "location.latitude", true),
		// bare facets (message-typed, no scalar value) are not sortable
		Entry("audio (bare facet)", "audio", "", false),
		Entry("photo (bare facet)", "photo", "", false),
		Entry("location (bare facet)", "location", "", false),
		Entry("location (dotted but not scalar)", "location.", "", false),
		// multivalued fields are not sortable
		Entry("tags (repeated)", "tags", "", false),
		Entry("Tags (index name, repeated)", "Tags", "", false),
		// internal index fields are not exposed under their index names
		Entry("Name (index name)", "Name", "", false),
		Entry("Mtime (index name)", "Mtime", "", false),
		Entry("RootID", "RootID", "", false),
		Entry("Deleted", "Deleted", "", false),
		// unknown fields
		Entry("unknown", "definitelyNotAField", "", false),
		Entry("unknown facet field", "photo.definitelyNotAField", "", false),
		Entry("empty", "", "", false),
	)
})

var _ = Describe("CompareMatches", func() {
	match := func(mutate func(e *searchmsg.Entity)) *searchmsg.Match {
		e := &searchmsg.Entity{}
		mutate(e)
		return &searchmsg.Match{Entity: e}
	}
	strPtr := func(s string) *string { return &s }
	asc := func(name string) []*searchsvc.SortProperty {
		return []*searchsvc.SortProperty{{Name: name}}
	}
	desc := func(name string) []*searchsvc.SortProperty {
		return []*searchsvc.SortProperty{{Name: name, IsDescending: true}}
	}

	It("compares string fields lexicographically", func() {
		a := match(func(e *searchmsg.Entity) { e.Name = "a.jpg" })
		b := match(func(e *searchmsg.Entity) { e.Name = "b.jpg" })
		Expect(search.CompareMatches(a, b, asc("name"))).To(Equal(-1))
		Expect(search.CompareMatches(b, a, asc("name"))).To(Equal(1))
		Expect(search.CompareMatches(a, b, desc("name"))).To(Equal(1))
	})

	It("compares lowercase-analyzed fields case-insensitively, like the index sorts them", func() {
		upper := match(func(e *searchmsg.Entity) { e.Name = "B.jpg" })
		lower := match(func(e *searchmsg.Entity) { e.Name = "a.jpg" })
		Expect(search.CompareMatches(lower, upper, asc("name"))).To(Equal(-1))
		Expect(search.CompareMatches(upper, lower, asc("name"))).To(Equal(1))
	})

	It("compares case-preserved fields case-sensitively", func() {
		upper := match(func(e *searchmsg.Entity) { e.Audio = &searchmsg.Audio{Artist: strPtr("Beatles")} })
		lower := match(func(e *searchmsg.Entity) { e.Audio = &searchmsg.Audio{Artist: strPtr("abba")} })
		// byte order: uppercase sorts before lowercase
		Expect(search.CompareMatches(upper, lower, asc("audio.artist"))).To(Equal(-1))
	})

	It("compares numeric fields numerically", func() {
		small := match(func(e *searchmsg.Entity) { e.Size = 9 })
		big := match(func(e *searchmsg.Entity) { e.Size = 10 })
		Expect(search.CompareMatches(small, big, asc("size"))).To(Equal(-1))
		Expect(search.CompareMatches(small, big, desc("size"))).To(Equal(1))
	})

	It("compares timestamps", func() {
		older := match(func(e *searchmsg.Entity) {
			e.Photo = &searchmsg.Photo{TakenDateTime: &timestamppb.Timestamp{Seconds: 100}}
		})
		newer := match(func(e *searchmsg.Entity) {
			e.Photo = &searchmsg.Photo{TakenDateTime: &timestamppb.Timestamp{Seconds: 200}}
		})
		Expect(search.CompareMatches(older, newer, asc("photo.takenDateTime"))).To(Equal(-1))
		Expect(search.CompareMatches(older, newer, desc("photo.takenDateTime"))).To(Equal(1))
	})

	It("compares lastModifiedDateTime via the entity's lastModifiedTime", func() {
		older := match(func(e *searchmsg.Entity) {
			e.LastModifiedTime = &timestamppb.Timestamp{Seconds: 100}
		})
		newer := match(func(e *searchmsg.Entity) {
			e.LastModifiedTime = &timestamppb.Timestamp{Seconds: 200}
		})
		Expect(search.CompareMatches(older, newer, asc("lastModifiedDateTime"))).To(Equal(-1))
	})

	It("sorts matches missing the field after those that have it, in both directions", func() {
		has := match(func(e *searchmsg.Entity) {
			e.Photo = &searchmsg.Photo{TakenDateTime: &timestamppb.Timestamp{Seconds: 100}}
		})
		missing := match(func(e *searchmsg.Entity) {})
		Expect(search.CompareMatches(has, missing, asc("photo.takenDateTime"))).To(Equal(-1))
		Expect(search.CompareMatches(missing, has, asc("photo.takenDateTime"))).To(Equal(1))
		Expect(search.CompareMatches(has, missing, desc("photo.takenDateTime"))).To(Equal(-1))
	})

	It("falls through to the next sort property on ties", func() {
		a := match(func(e *searchmsg.Entity) { e.Size = 5; e.Name = "a" })
		b := match(func(e *searchmsg.Entity) { e.Size = 5; e.Name = "b" })
		orderBy := []*searchsvc.SortProperty{{Name: "size"}, {Name: "name"}}
		Expect(search.CompareMatches(a, b, orderBy)).To(Equal(-1))
	})

	It("returns 0 for full ties and empty orderBy", func() {
		a := match(func(e *searchmsg.Entity) { e.Size = 5 })
		b := match(func(e *searchmsg.Entity) { e.Size = 5 })
		Expect(search.CompareMatches(a, b, asc("size"))).To(Equal(0))
		Expect(search.CompareMatches(a, b, nil)).To(Equal(0))
	})
})
