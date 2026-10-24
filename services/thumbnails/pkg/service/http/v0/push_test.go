package svc

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("the push endpoint", func() {
	Describe("the configured maximum input resolution", func() {
		DescribeTable("refuses an image above it before decoding",
			func(source []byte) {
				rec := postTo(testService(100, 100), "/unsafe/64x64/filters:format(png)/", source, "image/png")

				Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
			},
			Entry("both axes too large", pngSource(500, 500)),
			Entry("one axis too large", pngSource(500, 10)),
		)

		It("lets an image within it through", func() {
			rec := postTo(testService(100, 100), "/unsafe/32x32/filters:format(png)/", pngSource(50, 50), "image/png")

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
		})

	})

	Describe("the configured maximum thumbnail size", func() {
		It("refuses a box above it", func() {
			rec := postTo(testServiceOutput(2048, 2048), "/unsafe/4096x4096/", pngSource(10, 10), "image/png")

			Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
		})

		It("lets a box at it through", func() {
			rec := postTo(testServiceOutput(2048, 2048), "/unsafe/2048x2048/", pngSource(10, 10), "image/png")

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
		})
	})

	Describe("the configured maximum posted image size", func() {
		It("refuses a body above it", func() {
			rec := postTo(testServiceBody("1KB"), "/unsafe/50x50/", jpegSource(300, 300), "image/jpeg")

			Expect(rec.Code).To(Equal(http.StatusRequestEntityTooLarge), rec.Body.String())
		})

		It("lets a body within it through", func() {
			rec := postTo(testServiceBody("10MB"), "/unsafe/50x50/", jpegSource(300, 300), "image/jpeg")

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
		})
	})

	Describe("the configured maximum number of animation frames", func() {
		It("refuses a gif with more frames than allowed, before decoding them", func() {
			rec := postTo(testServiceFrames(3), "/unsafe/50x50/", gifSource(20, 20, 4), "image/gif")

			Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
		})

		It("lets a gif at the limit through", func() {
			rec := postTo(testServiceFrames(3), "/unsafe/50x50/", gifSource(20, 20, 3), "image/gif")

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(decodedGIF(rec).Image).To(HaveLen(3))
		})

		It("refuses a frame that declares more than the resolution limit, whatever the screen says", func() {
			rec := postTo(testService(100, 100), "/unsafe/50x50/", gifFrameBomb(), "image/gif")

			Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
		})
	})

	Describe("the concurrency limiter", func() {
		It("hands out as many slots as configured and then refuses", func() {
			l := newConcurrencyLimiter(2)

			ok, _ := l.acquire()
			Expect(ok).To(BeTrue())
			ok, _ = l.acquire()
			Expect(ok).To(BeTrue())

			ok, retryAfter := l.acquire()
			Expect(ok).To(BeFalse(), "the third request exceeds the limit")
			Expect(retryAfter).To(BeNumerically(">", 0), "a refused request needs a hint when to come back")

			l.release()
			ok, _ = l.acquire()
			Expect(ok).To(BeTrue(), "a released slot can be taken again")
		})

		It("is unlimited when no limit is configured", func() {
			var l *concurrencyLimiter
			for range 100 {
				ok, _ := l.acquire()
				Expect(ok).To(BeTrue())
			}
			l.release()
		})

		It("answers 429 with a Retry-After while saturated", func() {
			s := Thumbnails{limiter: newConcurrencyLimiter(1)}
			ok, _ := s.limiter.acquire()
			Expect(ok).To(BeTrue())
			defer s.limiter.release()

			// the handler is called without the router, so the box it reads from
			// the path is handed in, and the image has to be in before a slot is
			// asked for
			body, contentType := multipartBody(jpegSource(50, 50), "image/jpeg")
			req := httptest.NewRequest(http.MethodPost, "/unsafe/64x64/filters:format(png)/", body)
			req.Header.Set("Content-Type", contentType)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("width", "64")
			rctx.URLParams.Add("height", "64")
			rctx.URLParams.Add("*", "filters:format(png)/")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			s.pushHandler(rec, req)

			Expect(rec.Code).To(Equal(http.StatusTooManyRequests))
			Expect(rec.Header().Get("Retry-After")).ToNot(BeEmpty())
		})
	})
})

var _ = Describe("the backend a build compiled in", func() {
	DescribeTable("answers a request without a format filter in the input's own format, where it can write it",
		func(source []byte, want string) {
			src, err := loadSource(source)
			Expect(err).ToNot(HaveOccurred())
			defer src.Close()

			Expect(extForSourceFormat(src)).To(Equal(want))
		},
		Entry("png", pngSource(10, 10), "png"),
		Entry("jpeg", jpegSource(10, 10), "jpg"),
		Entry("gif", gifSource(10, 10, 1), "gif"),
		Entry("a format it cannot write becomes a jpeg", bmpSource(10, 10), "jpg"),
	)

	DescribeTable("reads the declared size off the header, before any pixel exists",
		func(source []byte, wantWidth, wantHeight int) {
			src, err := loadSource(source)
			Expect(err).ToNot(HaveOccurred())
			defer src.Close()

			Expect(src.Width()).To(Equal(wantWidth))
			Expect(src.Height()).To(Equal(wantHeight))
		},
		Entry("png", pngSource(300, 200), 300, 200),
		Entry("jpeg", jpegSource(120, 90), 120, 90),
	)

	It("refuses a body it cannot read at all", func() {
		_, err := loadSource([]byte("nonsense"))
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("splitFilter", func() {
	DescribeTable("splits a filter into its name and arguments",
		func(token, wantName, wantArgs string, wantOK bool) {
			name, args, ok := splitFilter(token)

			Expect(ok).To(Equal(wantOK))
			Expect(name).To(Equal(wantName))
			Expect(args).To(Equal(wantArgs))
		},
		Entry("with arguments", "format(jpeg)", "format", "jpeg", true),
		Entry("without arguments", "no_upscale()", "no_upscale", "", true),
		Entry("without the closing paren", "format(jpeg", "", "", false),
		Entry("without the opening paren", "formatjpeg)", "", "", false),
		Entry("not a filter at all", "plain", "", "", false),
		Entry("empty", "", "", "", false),
	)
})

var _ = Describe("outputFormatFromFilters", func() {
	DescribeTable("reads the format filter out of a segment",
		func(segment, wantExt string, wantHasFormat, wantErr bool) {
			ext, hasFormat, err := outputFormatFromFilters(segment)

			if wantErr {
				Expect(err).To(HaveOccurred())

				return
			}

			Expect(err).ToNot(HaveOccurred())
			Expect(ext).To(Equal(wantExt))
			Expect(hasFormat).To(Equal(wantHasFormat))
		},
		Entry("jpeg maps onto our jpg", "format(jpeg)", "jpg", true, false),
		Entry("png", "format(png)", "png", true, false),
		Entry("gif", "format(gif)", "gif", true, false),
		Entry("webp, which only the libvips build writes", "format(webp)", "webp", true, false),
		Entry("behind another filter", "no_upscale():format(png)", "png", true, false),
		Entry("no format filter at all", "no_upscale()", "", false, false),
		Entry("an empty segment", "", "", false, false),
		Entry("a format nobody writes", "format(tiff)", "", false, true),
	)
})
