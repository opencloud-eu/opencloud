package svc

import (
	"context"
	"testing"
	"time"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
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

var _ = Describe("multistatusResponse", func() {
	It("carries the facets of a hit in the PROPFIND shape", func() {
		id := &searchmsg.ResourceID{StorageId: "storage", SpaceId: "space", OpaqueId: "file"}
		res, err := multistatusResponse(context.Background(), "", "", []*searchmsg.Match{{Entity: &searchmsg.Entity{
			Ref:       &searchmsg.Reference{ResourceId: id},
			Id:        id,
			Audio:     &searchmsg.Audio{},
			Photo:     &searchmsg.Photo{FNumber: proto.Float32(1.8), TakenDateTime: timestamppb.New(time.Date(2018, 1, 1, 12, 34, 56, 0, time.UTC))},
			LivePhoto: &searchmsg.LivePhoto{ContentId: proto.String("a&b"), StillImageTimeUs: proto.Int64(1233333)},
		}}}, &userv1beta1.User{Id: &userv1beta1.UserId{}})
		Expect(err).ToNot(HaveOccurred())

		Expect(string(res)).To(ContainSubstring("<oc:photo><oc:f-number>1.8</oc:f-number><oc:taken-date-time>2018-01-01T12:34:56Z</oc:taken-date-time></oc:photo>"))
		Expect(string(res)).To(ContainSubstring("<oc:live-photo><oc:content-id>a&amp;b</oc:content-id><oc:still-image-time-us>1233333</oc:still-image-time-us></oc:live-photo>"))
		Expect(string(res)).ToNot(ContainSubstring("oc:audio"))
	})
})
