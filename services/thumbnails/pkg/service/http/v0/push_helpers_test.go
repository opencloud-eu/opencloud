package svc

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config/defaults"
	"golang.org/x/image/bmp"
)

// pngSource is a png of the given size, its pixels varying so a resize is visible.
func pngSource(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

// jpegSource is a jpeg of the given size.
func jpegSource(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: 64, B: uint8(y % 256), A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

// bmpSource is a bmp, a format this service reads but cannot write.
func bmpSource(width, height int) []byte {
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

// gifSource is an animated gif whose frames are solid colours, so preserved
// animation can be told apart from a flattened one.
func gifSource(width, height, frames int) []byte {
	pal := color.Palette{color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255}}
	g := &gif.GIF{}
	for i := range frames {
		frame := image.NewPaletted(image.Rect(0, 0, width, height), pal)
		for y := range height {
			for x := range width {
				frame.SetColorIndex(x, y, uint8(i%len(pal)))
			}
		}
		g.Image = append(g.Image, frame)
		g.Delay = append(g.Delay, 10)
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

// dimensionBomb is a tiny valid jpeg whose SOF0 header claims huge dimensions,
// so the guard has to refuse it before a pixel buffer is allocated.
func dimensionBomb(width, height uint16) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, 8, 8)), &jpeg.Options{Quality: 10}); err != nil {
		panic(err)
	}

	b := buf.Bytes()
	for i := 0; i+9 < len(b); i++ {
		if b[i] == 0xff && b[i+1] == 0xc0 {
			b[i+5], b[i+6] = byte(height>>8), byte(height)
			b[i+7], b[i+8] = byte(width>>8), byte(width)

			return b
		}
	}

	panic("no SOF0 marker in the encoded jpeg")
}

// gifDimensionBomb declares an oversized logical screen in its header.
func gifDimensionBomb() []byte {
	g := []byte("GIF89a")
	g = append(g, 0x20, 0x4e, 0x20, 0x4e, 0xf0, 0x00, 0x00)
	g = append(g, bytes.Repeat([]byte{0}, 6)...)

	return g
}

// multipartBody wraps the source the way webdav does, with the part typed.
func multipartBody(source []byte, partType string) (*bytes.Buffer, string) {
	GinkgoHelper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="image"; filename="test.png"`)
	if partType != "" {
		h.Set("Content-Type", partType)
	}

	part, err := writer.CreatePart(h)
	Expect(err).ToNot(HaveOccurred())
	_, err = part.Write(source)
	Expect(err).ToNot(HaveOccurred())
	Expect(writer.Close()).To(Succeed())

	return body, writer.FormDataContentType()
}

// testService is the real service with the configured input limits; 0 disables a
// bound.
func testService(maxWidth, maxHeight int) Service {
	GinkgoHelper()

	cfg := defaults.DefaultConfig()
	defaults.EnsureDefaults(cfg)
	cfg.Thumbnail.MaxInputWidth, cfg.Thumbnail.MaxInputHeight = maxWidth, maxHeight

	return NewService(Logger(log.NopLogger()), Config(cfg))
}

// post sends a typed source to a path without input limits.
func post(path string, source []byte) *httptest.ResponseRecorder {
	GinkgoHelper()

	return postTo(testService(0, 0), path, source, "image/png")
}

// postTo sends a source with an explicit part type to a path on the given service.
func postTo(handler http.Handler, path string, source []byte, partType string) *httptest.ResponseRecorder {
	GinkgoHelper()

	body, contentType := multipartBody(source, partType)
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", contentType)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

// decodedBounds is the size of the image in a response, whatever format it is in.
func decodedBounds(rec *httptest.ResponseRecorder) image.Rectangle {
	GinkgoHelper()

	img, _, err := image.Decode(bytes.NewReader(rec.Body.Bytes()))
	Expect(err).ToNot(HaveOccurred(), "the response should be a decodable image")

	return img.Bounds()
}

// decodedGIF is the animation in a response.
func decodedGIF(rec *httptest.ResponseRecorder) *gif.GIF {
	GinkgoHelper()

	g, err := gif.DecodeAll(bytes.NewReader(rec.Body.Bytes()))
	Expect(err).ToNot(HaveOccurred(), "the response should be a decodable gif")

	return g
}
