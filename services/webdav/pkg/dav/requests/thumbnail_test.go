package requests_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/constants"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/dav/requests"
)

func thumbnailRequest(rawQuery string) *http.Request {
	req := httptest.NewRequest("GET", "/files/test.png?"+rawQuery, nil)

	return req.WithContext(context.WithValue(req.Context(), constants.ContextKeyPath, "/files/test.png"))
}

var _ = Describe("ParseThumbnailRequest", func() {
	DescribeTable("the legacy a flag",
		func(query string, want *bool) {
			tr, err := requests.ParseThumbnailRequest(thumbnailRequest(query))
			Expect(err).ToNot(HaveOccurred())
			if want == nil {
				Expect(tr.Aspect).To(BeNil())

				return
			}
			Expect(tr.Aspect).To(HaveValue(Equal(*want)))
		},
		Entry("stays unset when the client does not send it", "x=1024&y=1024", nil),
		Entry("preserves the aspect ratio with a=1", "x=1024&y=1024&a=1", conversions.ToPointer(true)),
		Entry("fills the box with a=0", "x=1024&y=1024&a=0", conversions.ToPointer(false)),
	)

	It("keeps the processor and both dimensions", func() {
		tr, err := requests.ParseThumbnailRequest(thumbnailRequest("x=320&y=320&processor=fit"))
		Expect(err).ToNot(HaveOccurred())
		Expect(tr.Processor).To(Equal("fit"))
		Expect(tr.Width).To(Equal(int32(320)))
		Expect(tr.Height).To(Equal(int32(320)))
	})

	DescribeTable("the requested box",
		func(query string, wantWidth, wantHeight int32) {
			tr, err := requests.ParseThumbnailRequest(thumbnailRequest(query))
			Expect(err).ToNot(HaveOccurred())
			Expect(tr.Width).To(Equal(wantWidth))
			Expect(tr.Height).To(Equal(wantHeight))
		},
		Entry("takes both axes when they are given", "x=448&y=448", int32(448), int32(448)),
		Entry("keeps the legacy default when neither is given", "preview=1", int32(requests.DefaultWidth), int32(requests.DefaultHeight)),
		Entry("leaves an omitted axis unconstrained", "y=512&processor=fit", int32(0), int32(512)),
		Entry("fits an unconstrained axis without a processor", "y=512", int32(0), int32(512)),
		Entry("treats a processor nobody knows like none at all", "y=512&processor=bogus", int32(0), int32(512)),
	)

	DescribeTable("a request it refuses",
		func(query string) {
			_, err := requests.ParseThumbnailRequest(thumbnailRequest(query))
			Expect(err).To(HaveOccurred())
		},
		Entry("an explicit zero", "x=0&y=512&processor=fit"),
		Entry("a negative size", "x=-1&y=512"),
		Entry("cropping to an unconstrained axis", "y=512&processor=thumbnail"),
	)
})
