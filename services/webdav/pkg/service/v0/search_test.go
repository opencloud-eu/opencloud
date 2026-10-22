package svc

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

func TestSearch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Search Suite")
}

var _ = Describe("SpacesSearchRegex", func() {
	DescribeTable("path matching",
		func(path, expectedSpace string, shouldMatch bool) {
			matches := spacesSearchRegex.FindStringSubmatch(path)
			if shouldMatch {
				Expect(matches).ToNot(BeNil(), "Expected path %q to match", path)
				Expect(matches[1]).To(Equal(expectedSpace), "Expected space to be %q", expectedSpace)
			} else {
				Expect(matches).To(BeNil(), "Expected path %q not to match", path)
			}
		},
		Entry("standard dav spaces path", "/dav/spaces/12345", "12345", true),
		Entry("remote.php dav spaces path", "/remote.php/dav/spaces/12345", "12345", true),
		Entry("standard dav spaces path with subpaths", "/dav/spaces/12345/some/folder", "12345", true),
		Entry("remote.php dav spaces path with subpaths", "/remote.php/dav/spaces/12345/some/folder", "12345", true),
		Entry("standard dav spaces path without space", "/dav/spaces/", "", false),
		Entry("remote.php dav spaces path without space", "/remote.php/dav/spaces/", "", false),
		Entry("prefix match only", "/dav/spaces", "", false),
		Entry("unrelated path", "/dav/files/123", "", false),
	)
})

var _ = Describe("searchRequestOf", func() {
	DescribeTable("maps pattern, limit and offset of the report",
		func(search reportSearchFilesSearch, want *searchsvc.SearchRequest) {
			got, err := searchRequestOf(search)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(BeComparableTo(want, protocmp.Transform()))
		},
		Entry("the pattern alone, the service applies its default page size",
			reportSearchFilesSearch{Pattern: "notes"}, &searchsvc.SearchRequest{Query: "notes"}),
		Entry("a limit", reportSearchFilesSearch{Pattern: "notes", Limit: 25},
			&searchsvc.SearchRequest{Query: "notes", PageSize: conversions.ToPointer(int32(25))}),
		Entry("no limit", reportSearchFilesSearch{Pattern: "notes", Limit: -1},
			&searchsvc.SearchRequest{Query: "notes", PageSize: conversions.ToPointer(int32(-1))}),
		Entry("an offset pages the merged list", reportSearchFilesSearch{Pattern: "notes", Limit: 25, Offset: 50},
			&searchsvc.SearchRequest{Query: "notes", PageSize: conversions.ToPointer(int32(25)), From: 50}),
	)

	DescribeTable("rejects",
		func(search reportSearchFilesSearch) {
			_, err := searchRequestOf(search)
			Expect(err).To(HaveOccurred())
		},
		Entry("a negative offset", reportSearchFilesSearch{Pattern: "notes", Offset: -1}),
		Entry("a limit below -1", reportSearchFilesSearch{Pattern: "notes", Limit: -2}),
	)
})
