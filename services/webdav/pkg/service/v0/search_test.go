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
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
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

var _ = Describe("matchToPropResponse", func() {
	propsOf := func(entity *searchmsg.Entity) map[string]string {
		entity.Ref = &searchmsg.Reference{ResourceId: &searchmsg.ResourceID{StorageId: "storage", SpaceId: "space"}, Path: "./photo.jpg"}
		entity.Id = &searchmsg.ResourceID{StorageId: "storage", SpaceId: "space", OpaqueId: "photo"}
		entity.LastModifiedTime = timestamppb.Now()
		res, err := matchToPropResponse(context.Background(), "", "https://cloud.example.com", &searchmsg.Match{Entity: entity}, &userv1beta1.User{Id: &userv1beta1.UserId{OpaqueId: "alice"}}, thumbnail.SourceTypes{})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Propstat).To(HaveLen(1))
		props := map[string]string{}
		for _, p := range res.Propstat[0].Prop {
			props[p.XMLName.Local] = string(p.InnerXML)
		}
		return props
	}

	It("returns the photo and location metadata", func() {
		props := propsOf(&searchmsg.Entity{
			Photo: &searchmsg.Photo{
				TakenDateTime:       timestamppb.New(time.Date(2024, 5, 1, 12, 30, 0, 0, time.UTC)),
				CameraMake:          proto.String("Acme"),
				CameraModel:         proto.String("X100"),
				FNumber:             proto.Float32(1.8),
				FocalLength:         proto.Float32(26),
				Iso:                 proto.Int32(400),
				Orientation:         proto.Int32(6),
				ExposureNumerator:   proto.Float32(1),
				ExposureDenominator: proto.Float32(250),
			},
			Location: &searchmsg.GeoCoordinates{Latitude: proto.Float64(48.137154), Longitude: proto.Float64(11.576124), Altitude: proto.Float64(519.5)},
		})
		Expect(props).To(HaveKeyWithValue("oc:photo-taken-date-time", "2024-05-01T12:30:00Z"))
		Expect(props).To(HaveKeyWithValue("oc:photo-camera-make", "Acme"))
		Expect(props).To(HaveKeyWithValue("oc:photo-camera-model", "X100"))
		Expect(props).To(HaveKeyWithValue("oc:photo-fnumber", "1.80"))
		Expect(props).To(HaveKeyWithValue("oc:photo-focal-length", "26.00"))
		Expect(props).To(HaveKeyWithValue("oc:photo-iso", "400"))
		Expect(props).To(HaveKeyWithValue("oc:photo-orientation", "6"))
		Expect(props).To(HaveKeyWithValue("oc:photo-exposure-numerator", "1"))
		Expect(props).To(HaveKeyWithValue("oc:photo-exposure-denominator", "250"))
		Expect(props).To(HaveKeyWithValue("oc:photo-location-latitude", "48.137154"))
		Expect(props).To(HaveKeyWithValue("oc:photo-location-longitude", "11.576124"))
		Expect(props).To(HaveKeyWithValue("oc:photo-location-altitude", "519.5"))
	})

	It("keeps a zero coordinate but skips empty photo values", func() {
		props := propsOf(&searchmsg.Entity{
			Photo:    &searchmsg.Photo{CameraMake: proto.String(""), Iso: proto.Int32(0)},
			Location: &searchmsg.GeoCoordinates{Latitude: proto.Float64(0), Longitude: proto.Float64(0)},
		})
		Expect(props).To(HaveKeyWithValue("oc:photo-location-latitude", "0.000000"))
		Expect(props).To(HaveKeyWithValue("oc:photo-location-longitude", "0.000000"))
		Expect(props).ToNot(HaveKey("oc:photo-location-altitude"))
		Expect(props).ToNot(HaveKey("oc:photo-camera-make"))
		Expect(props).ToNot(HaveKey("oc:photo-iso"))
	})

	It("adds no photo props to a match without them", func() {
		for name := range propsOf(&searchmsg.Entity{}) {
			Expect(name).ToNot(HavePrefix("oc:photo-"))
		}
	})
})
