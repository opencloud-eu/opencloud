package mapping

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/types/known/timestamppb"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
)

var _ = Describe("FromProto", func() {
	It("maps an audio facet including proto3-JSON int64 strings", func() {
		in := &searchmsg.Audio{
			Artist:  conversions.ToPointer("Saxon"),
			Album:   conversions.ToPointer("Wheels of Steel"),
			Bitrate: conversions.ToPointer(int64(320)),
			Year:    conversions.ToPointer(int32(1980)),
		}
		out := FromProto[libregraph.Audio](in)
		Expect(out).ToNot(BeNil())
		Expect(out.GetArtist()).To(Equal("Saxon"))
		Expect(out.GetAlbum()).To(Equal("Wheels of Steel"))
		Expect(out.GetBitrate()).To(Equal(int64(320)))
		Expect(out.GetYear()).To(Equal(int32(1980)))
	})

	It("maps a timestamp to time.Time and float32 to float64", func() {
		taken := timestamppb.New(mustTime("2023-07-21T10:11:12Z"))
		in := &searchmsg.Photo{
			CameraMake:    conversions.ToPointer("Nikon"),
			FNumber:       conversions.ToPointer(float32(2.8)),
			TakenDateTime: taken,
		}
		out := FromProto[libregraph.Photo](in)
		Expect(out).ToNot(BeNil())
		Expect(out.GetCameraMake()).To(Equal("Nikon"))
		Expect(out.GetFNumber()).To(BeNumerically("~", 2.8, 0.0001))
		Expect(out.GetTakenDateTime()).To(Equal(mustTime("2023-07-21T10:11:12Z")))
	})

	It("round-trips through ToProto", func() {
		lg := &libregraph.Audio{Artist: conversions.ToPointer("Motörhead"), Bitrate: conversions.ToPointer(int64(256))}
		pb := ToProto[searchmsg.Audio](lg)
		Expect(pb).ToNot(BeNil())
		Expect(pb.GetArtist()).To(Equal("Motörhead"))
		Expect(pb.GetBitrate()).To(Equal(int64(256)))
		Expect(FromProto[libregraph.Audio](pb)).To(Equal(lg))
	})

	It("returns nil for nil input and for an empty message", func() {
		Expect(FromProto[libregraph.Audio]((*searchmsg.Audio)(nil))).To(BeNil())
		Expect(FromProto[libregraph.Audio](&searchmsg.Audio{})).To(BeNil())
	})
})

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	Expect(err).ToNot(HaveOccurred())
	return t
}
