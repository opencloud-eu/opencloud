package svc

import (
	"bytes"
	"image"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("the push endpoint", func() {
	DescribeTable("answers in the format the filters ask for",
		func(path string, source []byte, wantType string) {
			rec := post(path, source)

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(rec.Header().Get("Content-Type")).To(Equal(wantType))
			Expect(rec.Body.Len()).ToNot(BeZero())
		},
		Entry("jpeg", "/unsafe/fit-in/100x100/filters:format(jpeg)/", pngSource(200, 200), "image/jpeg"),
		Entry("png", "/unsafe/fit-in/100x100/filters:format(png)/", pngSource(200, 200), "image/png"),
		Entry("gif", "/unsafe/fit-in/100x100/filters:format(gif)/", gifSource(200, 200, 3), "image/gif"),
		Entry("jpeg while filling", "/unsafe/100x100/filters:format(jpeg)/", pngSource(200, 200), "image/jpeg"),
		Entry("a filter it does not know is ignored",
			"/unsafe/fit-in/100x100/filters:no_upscale():format(jpeg)/", pngSource(200, 200), "image/jpeg"),
		Entry("without a trailing slash", "/unsafe/fit-in/100x100/filters:format(png)", pngSource(200, 200), "image/png"),
	)

	DescribeTable("keeps the input's own format when no format filter is given",
		func(source []byte, wantType string) {
			rec := post("/unsafe/fit-in/100x100/filters:no_upscale()/", source)

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(rec.Header().Get("Content-Type")).To(Equal(wantType))
		},
		Entry("a gif stays a gif", gifSource(200, 200, 3), "image/gif"),
		Entry("a png stays a png", pngSource(200, 200), "image/png"),
		Entry("a jpeg stays a jpeg", jpegSource(200, 200), "image/jpeg"),
		Entry("a format it cannot write becomes a jpeg", bmpSource(200, 200), "image/jpeg"),
	)

	DescribeTable("sizes the result the way the operation says",
		func(path string, source []byte, want image.Rectangle) {
			rec := post(path, source)

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(decodedBounds(rec)).To(Equal(want))
		},
		Entry("fill center-crops to the exact box",
			"/unsafe/100x100/filters:format(png)/", pngSource(200, 100), image.Rect(0, 0, 100, 100)),
		Entry("fill crops a landscape source too",
			"/unsafe/100x100/filters:format(png)/", pngSource(400, 200), image.Rect(0, 0, 100, 100)),
		Entry("fill enlarges a small source, like imagor's default resize",
			"/unsafe/200x200/filters:format(png)/", pngSource(50, 50), image.Rect(0, 0, 200, 200)),
		Entry("no_upscale caps fill at the source size",
			"/unsafe/200x200/filters:no_upscale():format(png)/", pngSource(50, 50), image.Rect(0, 0, 50, 50)),
		Entry("fit-in inscribes and letterboxes",
			"/unsafe/fit-in/100x100/filters:format(png)/", pngSource(200, 100), image.Rect(0, 0, 100, 50)),
		Entry("fit-in never enlarges, imagor needs upscale() for that",
			"/unsafe/fit-in/200x200/filters:format(png)/", pngSource(50, 50), image.Rect(0, 0, 50, 50)),
		Entry("stretch distorts to the exact box",
			"/unsafe/stretch/64x32/filters:format(png)/", pngSource(200, 100), image.Rect(0, 0, 64, 32)),
	)

	It("fits a source inside the box without exceeding it", func() {
		rec := post("/unsafe/fit-in/100x100/filters:format(png)/", pngSource(400, 200))

		Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
		bounds := decodedBounds(rec)
		Expect(bounds.Dx()).To(BeNumerically("<=", 100))
		Expect(bounds.Dy()).To(BeNumerically("<=", 100))
	})

	Describe("an animated gif", func() {
		It("keeps its frames when fitted into the box", func() {
			rec := post("/unsafe/fit-in/50x50/filters:format(gif)/", gifSource(200, 100, 4))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			out := decodedGIF(rec)
			Expect(out.Image).To(HaveLen(4))
			for i, frame := range out.Image {
				Expect(frame.Bounds().Dx()).To(BeNumerically("<=", 50), "frame %d", i)
				Expect(frame.Bounds().Dy()).To(BeNumerically("<=", 50), "frame %d", i)
			}
		})

		It("keeps its frames when filled to the exact box", func() {
			rec := post("/unsafe/64x64/filters:format(gif)/", gifSource(200, 100, 3))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			out := decodedGIF(rec)
			Expect(out.Image).To(HaveLen(3))
			for i, frame := range out.Image {
				Expect(frame.Bounds()).To(Equal(image.Rect(0, 0, 64, 64)), "frame %d", i)
			}
		})

		// A stretch past the source size with no_upscale() set, which is what
		// webdav sends, used to leave frames outside the logical screen and fail
		// the encode with "image block is out of bounds".
		It("grows its logical screen when stretched past the source", func() {
			rec := post("/unsafe/stretch/200x160/filters:no_upscale():format(gif)/", gifSource(100, 80, 3))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			out := decodedGIF(rec)
			Expect(out.Image).To(HaveLen(3))
			for i, frame := range out.Image {
				Expect(frame.Bounds()).To(Equal(image.Rect(0, 0, 200, 160)), "frame %d", i)
			}
		})

		It("flattens to the first frame when the output is still", func() {
			rec := post("/unsafe/50x50/filters:format(png)/", gifSource(200, 100, 3))

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(rec.Header().Get("Content-Type")).To(Equal("image/png"))
			Expect(decodedBounds(rec)).To(Equal(image.Rect(0, 0, 50, 50)))
		})
	})

	// Measured against shumc/imagor:1.9.11: everything behind the box is imagor's
	// tail, and a side given as 0 is computed from the other one.
	DescribeTable("reads imagor's path the way imagor reads it",
		func(path string, source []byte, want image.Rectangle) {
			rec := post(path, source)

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(decodedBounds(rec)).To(Equal(want))
		},
		Entry("an empty filters segment is a filters segment",
			"/unsafe/50x50/filters:", pngSource(200, 200), image.Rect(0, 0, 50, 50)),
		Entry("an empty filters segment with an operation",
			"/unsafe/fit-in/50x50/filters:", pngSource(200, 100), image.Rect(0, 0, 50, 25)),
		Entry("an image key behind the box is ignored",
			"/unsafe/50x50/foo.png", pngSource(200, 200), image.Rect(0, 0, 50, 50)),
		Entry("an image key behind the filters is ignored",
			"/unsafe/50x50/filters:format(png)/foo.png", pngSource(200, 200), image.Rect(0, 0, 50, 50)),
		Entry("a zero width follows the height",
			"/unsafe/0x100/filters:format(png)/", pngSource(300, 200), image.Rect(0, 0, 150, 100)),
		Entry("a zero height follows the width",
			"/unsafe/100x0/filters:format(png)/", pngSource(300, 200), image.Rect(0, 0, 100, 67)),
		Entry("a free axis ignores the operation, there is nothing to crop to",
			"/unsafe/fit-in/0x100/filters:format(png)/", pngSource(300, 200), image.Rect(0, 0, 150, 100)),
		Entry("a free axis enlarges, like imagor",
			"/unsafe/0x200/filters:format(png)/", pngSource(150, 100), image.Rect(0, 0, 300, 200)),
		Entry("a zero box leaves the source alone",
			"/unsafe/0x0/filters:format(png)/", pngSource(300, 200), image.Rect(0, 0, 300, 200)),
	)

	It("keeps an animated gif animated on a free axis", func() {
		rec := post("/unsafe/0x40/filters:format(gif)/", gifSource(200, 100, 3))

		Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
		out := decodedGIF(rec)
		Expect(out.Image).To(HaveLen(3))
		Expect(out.Image[0].Bounds()).To(Equal(image.Rect(0, 0, 80, 40)))
	})

	DescribeTable("refuses what it cannot answer",
		func(path string, source []byte, partType string, want int) {
			rec := postTo(testMux(0, 0), path, source, partType)

			Expect(rec.Code).To(Equal(want), rec.Body.String())
		},
		Entry("dimensions that are not numbers",
			"/unsafe/fit-in/abcxdef/filters:format(jpeg)/", pngSource(100, 100), "image/png", http.StatusBadRequest),
		Entry("a negative box",
			"/unsafe/fit-in/-50x50/filters:format(jpeg)/", pngSource(100, 100), "image/png", http.StatusBadRequest),
		Entry("a format nobody writes",
			"/unsafe/fit-in/50x50/filters:format(tiff)/", pngSource(100, 100), "image/png", http.StatusBadRequest),
		Entry("bytes that are not an image",
			"/unsafe/fit-in/50x50/filters:format(jpeg)/", []byte("not an image"), "image/png", http.StatusNotAcceptable),
		Entry("an untyped part, which imagor's upload loader also refuses",
			"/unsafe/fit-in/50x50/filters:format(png)/", pngSource(100, 100), "application/octet-stream", http.StatusNotAcceptable),
	)

	It("refuses a body that is not multipart", func() {
		req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/50x50/filters:format(jpeg)/", bytes.NewReader([]byte("plain text")))
		rec := httptest.NewRecorder()
		testMux(0, 0).ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusBadRequest))
	})

	It("refuses a multipart body whose only field is not the image", func() {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		Expect(writer.WriteField("other_field", "some value")).To(Succeed())
		Expect(writer.Close()).To(Succeed())

		req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/50x50/filters:format(jpeg)/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		testMux(0, 0).ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusBadRequest), rec.Body.String())
	})

	Describe("the configured maximum input resolution", func() {
		DescribeTable("refuses an image above it before decoding",
			func(source []byte) {
				rec := postTo(testMux(100, 100), "/unsafe/64x64/filters:format(png)/", source, "image/png")

				Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
			},
			Entry("both axes too large", pngSource(500, 500)),
			Entry("one axis too large", pngSource(500, 10)),
		)

		It("lets an image within it through", func() {
			rec := postTo(testMux(100, 100), "/unsafe/32x32/filters:format(png)/", pngSource(50, 50), "image/png")

			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
		})

		// a tiny file that declares huge dimensions must not reach a decoder
		DescribeTable("refuses a dimension bomb from its header alone",
			func(source []byte, path string) {
				rec := postTo(testMux(7680, 7680), path, source, "image/png")

				Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity), rec.Body.String())
			},
			Entry("a jpeg", dimensionBomb(20000, 20000), "/unsafe/64x64/filters:format(png)/"),
			Entry("a gif", gifDimensionBomb(), "/unsafe/64x64/filters:format(gif)/"),
		)
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

			req := httptest.NewRequest(http.MethodPost, "/unsafe/64x64/filters:format(png)/", nil)
			rec := httptest.NewRecorder()
			s.pushHandler(rec, req)

			Expect(rec.Code).To(Equal(http.StatusTooManyRequests))
			Expect(rec.Header().Get("Retry-After")).ToNot(BeEmpty())
		})
	})
})

var _ = Describe("inputFormatViaImaging", func() {
	DescribeTable("names the format a body is in",
		func(source []byte, want string) {
			Expect(inputFormatViaImaging(source)).To(Equal(want))
		},
		Entry("png", pngSource(10, 10), "png"),
		Entry("jpeg", jpegSource(10, 10), "jpg"),
		Entry("gif", gifSource(10, 10, 1), "gif"),
		Entry("a format it cannot write falls back to jpeg", bmpSource(10, 10), "jpg"),
		Entry("something undecodable falls back to jpeg", []byte("nonsense"), "jpg"),
	)
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
