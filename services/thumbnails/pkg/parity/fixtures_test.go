package parity

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// partType matters: imagor's upload loader accepts image/* only.
type source struct {
	name     string
	filename string
	partType string
	data     []byte
}

func encodePNG(w io.Writer, img image.Image) error { return png.Encode(w, img) }

func gradient(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x * 255 / max(w-1, 1)), uint8(y * 255 / max(h-1, 1)), 128, 255})
		}
	}
	return img
}

func pngSource(w, h int) source {
	GinkgoHelper()

	var buf bytes.Buffer
	Expect(png.Encode(&buf, gradient(w, h))).To(Succeed())

	return source{
		name: "png", filename: "source.png", partType: "image/png",
		data: buf.Bytes(),
	}
}

func jpegSource(w, h int) source {
	GinkgoHelper()

	var buf bytes.Buffer
	Expect(jpeg.Encode(&buf, gradient(w, h), &jpeg.Options{Quality: 90})).To(Succeed())

	return source{
		name: "jpeg", filename: "source.jpg", partType: "image/jpeg",
		data: buf.Bytes(),
	}
}

func animatedGIFSource(w, h, frames int) source {
	GinkgoHelper()

	g := &gif.GIF{}
	for i := 0; i < frames; i++ {
		frame := image.NewPaletted(image.Rect(0, 0, w, h), palette.Plan9)
		draw.Draw(frame, frame.Bounds(), gradient(w, h), image.Point{}, draw.Src)
		g.Image = append(g.Image, frame)
		g.Delay = append(g.Delay, 10)
	}

	var buf bytes.Buffer
	Expect(gif.EncodeAll(&buf, g)).To(Succeed())

	return source{
		name: "gif", filename: "source.gif", partType: "image/gif",
		data: buf.Bytes(),
	}
}

// the same picture as icoSource without the ICO container.
func alphaPNGSource(size int) source {
	GinkgoHelper()

	var buf bytes.Buffer
	Expect(encodePNG(&buf, transparentHalf(size))).To(Succeed())

	return source{
		name: "png+alpha", filename: "source.png", partType: "image/png",
		data: buf.Bytes(),
	}
}

func transparentHalf(size int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for x := size / 2; x < size; x++ {
		for y := 0; y < size; y++ {
			img.Set(x, y, color.NRGBA{200, 40, 40, 255})
		}
	}
	return img
}

// an ICO is what Tika hands over for a windows executable. Left half fully
// transparent, so a dropped alpha channel shows up in the answer.
func icoSource(size int) source {
	GinkgoHelper()

	var png bytes.Buffer
	Expect(encodePNG(&png, transparentHalf(size))).To(Succeed())

	// ICONDIR, one ICONDIRENTRY, then the png payload at offset 22
	var out bytes.Buffer
	side := byte(size)
	if size >= 256 {
		side = 0
	}
	out.Write([]byte{0, 0, 1, 0, 1, 0})
	out.Write([]byte{side, side, 0, 0, 1, 0, 32, 0})
	_ = binary.Write(&out, binary.LittleEndian, uint32(png.Len()))
	_ = binary.Write(&out, binary.LittleEndian, uint32(22))
	out.Write(png.Bytes())

	return source{
		name: "ico", filename: "source.ico", partType: "image/vnd.microsoft.icon",
		data: out.Bytes(),
	}
}

// a lossless 300x200 webp, embedded because the pure Go build has no webp
// encoder to make one with, and both builds have to be able to post it.
const webpFixture = "UklGRiQAAABXRUJQVlA4TBgAAAAvK8ExAAdQvCoUuf8BAEX6/58i+p/6/zU="

func webpSource() source {
	GinkgoHelper()

	data, err := base64.StdEncoding.DecodeString(webpFixture)
	Expect(err).ToNot(HaveOccurred())

	return source{
		name: "webp", filename: "source.webp", partType: "image/webp",
		data: data,
	}
}

func hasAlpha(data []byte) bool {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}
	bounds := img.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			if _, _, _, a := img.At(x, y).RGBA(); a < 0xffff {
				return true
			}
		}
	}
	return false
}

func frameCount(data []byte, format string) int {
	if format != "gif" {
		return 1
	}
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return 0
	}
	return len(g.Image)
}
