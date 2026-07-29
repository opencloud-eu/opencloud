package content

import (
	"context"
	"errors"
	"io"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var _ = Describe("getMotionPhoto", func() {
	It("maps the current MotionPhoto XMP scheme (container item length)", func() {
		mp := Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhotoVersion":                 {"1"},
			"Camera:MotionPhotoPresentationTimestampUs": {"1500000"},
			"Container:Directory/Item[2]/Item:Semantic": {"MotionPhoto"},
			"Container:Directory/Item[2]/Item:Length":   {"1048576"},
		})
		Expect(mp).ToNot(BeNil())
		Expect(mp.Version).To(Equal(libregraph.PtrInt32(1)))
		Expect(mp.PresentationTimestampUs).To(Equal(libregraph.PtrInt64(1500000)))
		Expect(mp.VideoSize).To(Equal(libregraph.PtrInt64(1048576)))
	})

	It("maps the legacy MicroVideo XMP scheme (offset is the length)", func() {
		mp := Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MicroVideoVersion":                 {"1"},
			"Camera:MicroVideoPresentationTimestampUs": {"1500000"},
			"Camera:MicroVideoOffset":                  {"2097152"},
		})
		Expect(mp).ToNot(BeNil())
		Expect(mp.Version).To(Equal(libregraph.PtrInt32(1)))
		Expect(mp.PresentationTimestampUs).To(Equal(libregraph.PtrInt64(1500000)))
		Expect(mp.VideoSize).To(Equal(libregraph.PtrInt64(2097152)))
	})

	It("drops the facet without a video size", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhotoVersion": {"1"},
		})).To(BeNil())
	})

	It("returns nil when no motion photo metadata is present", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{})).To(BeNil())
	})

	It("treats a zero MotionPhoto marker as a still image", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhoto":                        {"0"},
			"Camera:MotionPhotoVersion":                 {"1"},
			"Container:Directory/Item[2]/Item:Semantic": {"MotionPhoto"},
			"Container:Directory/Item[2]/Item:Length":   {"1048576"},
		})).To(BeNil())
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MicroVideo":       {"0"},
			"Camera:MicroVideoOffset": {"2097152"},
		})).To(BeNil())
	})

	It("treats undefined marker values as a still image", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhoto":                        {"2"},
			"Container:Directory/Item[2]/Item:Semantic": {"MotionPhoto"},
			"Container:Directory/Item[2]/Item:Length":   {"1048576"},
		})).To(BeNil())
	})

	It("prefers the current scheme when both are present", func() {
		mp := Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhotoVersion":                 {"2"},
			"Camera:MicroVideoVersion":                  {"1"},
			"Camera:MicroVideoOffset":                   {"2097152"},
			"Container:Directory/Item[2]/Item:Semantic": {"MotionPhoto"},
			"Container:Directory/Item[2]/Item:Length":   {"1048576"},
		})
		Expect(mp).ToNot(BeNil())
		Expect(mp.Version).To(Equal(libregraph.PtrInt32(2)))
		Expect(mp.VideoSize).To(Equal(libregraph.PtrInt64(1048576)))
	})
})

var _ = Describe("looksLikeMP4", func() {
	It("recognizes an ftyp box and rejects everything else", func() {
		Expect(looksLikeMP4([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'})).To(BeTrue())
		Expect(looksLikeMP4([]byte("JFIF garbage"))).To(BeFalse())
		Expect(looksLikeMP4([]byte{0, 0, 0})).To(BeFalse())
	})
})

// rangeStub serves RetrieveRange from a string and records the requested range.
type rangeStub struct {
	data      string
	err       error
	gotOffset int64
	gotLength int64
	calls     int
}

func (r *rangeStub) Retrieve(context.Context, *provider.ResourceId) (io.ReadCloser, error) {
	return nil, errors.New("unused")
}

func (r *rangeStub) RetrieveRange(_ context.Context, _ *provider.ResourceId, offset, length int64) (io.ReadCloser, error) {
	r.calls++
	r.gotOffset, r.gotLength = offset, length
	if r.err != nil {
		return nil, r.err
	}
	return io.NopCloser(strings.NewReader(r.data)), nil
}

var _ = Describe("motionPhotoHasVideo", func() {
	var (
		retriever *rangeStub
		tika      Tika
		ri        *provider.ResourceInfo
	)

	BeforeEach(func() {
		retriever = &rangeStub{}
		basic, err := NewBasicExtractor(log.NewLogger())
		Expect(err).ToNot(HaveOccurred())
		tika = Tika{Basic: basic, Retriever: retriever}
		ri = &provider.ResourceInfo{Size: 100}
	})

	It("confirms a video that starts with an ftyp box", func() {
		retriever.data = "\x00\x00\x00\x18ftypisom"
		Expect(tika.motionPhotoHasVideo(context.Background(), ri, 40)).To(BeTrue())
		Expect(retriever.gotOffset).To(Equal(int64(60)), "the video starts at size-videoSize")
		Expect(retriever.gotLength).To(Equal(int64(motionPhotoVideoSignatureLen)))
	})

	It("rejects trailing bytes without an MP4 signature", func() {
		retriever.data = "not a video "
		Expect(tika.motionPhotoHasVideo(context.Background(), ri, 40)).To(BeFalse())
	})

	It("rejects degenerate video sizes without reading", func() {
		Expect(tika.motionPhotoHasVideo(context.Background(), ri, 0)).To(BeFalse())
		Expect(tika.motionPhotoHasVideo(context.Background(), ri, -1)).To(BeFalse())
		Expect(tika.motionPhotoHasVideo(context.Background(), ri, 100)).To(BeFalse())
		Expect(tika.motionPhotoHasVideo(context.Background(), ri, 101)).To(BeFalse())
		Expect(retriever.calls).To(BeZero())
	})

	It("drops the facet when the range read fails", func() {
		retriever.err = errors.New("nope")
		Expect(tika.motionPhotoHasVideo(context.Background(), ri, 40)).To(BeFalse())
	})
})
