package gofakeit

import (
	"bytes"
	"errors"
	img "image"
	imgCol "image/color"
	"image/jpeg"
	"image/png"
)

// Image generates a random rgba image
func Image(width int, height int) *img.RGBA { return image(GlobalFaker, width, height) }

// Image generates a random rgba image
func (f *Faker) Image(width int, height int) *img.RGBA { return image(f, width, height) }

func image(f *Faker, width int, height int) *img.RGBA {
	upLeft := img.Point{0, 0}
	lowRight := img.Point{width, height}

	img := img.NewRGBA(img.Rectangle{upLeft, lowRight})

	// Set color for each pixel
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, imgCol.RGBA{uint8(number(f, 0, 255)), uint8(number(f, 0, 255)), uint8(number(f, 0, 255)), 0xff})
		}
	}

	return img
}

// ImageJpeg generates a random rgba jpeg image
func ImageJpeg(width int, height int) []byte { return imageJpeg(GlobalFaker, width, height) }

// ImageJpeg generates a random rgba jpeg image
func (f *Faker) ImageJpeg(width int, height int) []byte { return imageJpeg(f, width, height) }

func imageJpeg(f *Faker, width int, height int) []byte {
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, image(f, width, height), nil)
	return buf.Bytes()
}

// ImagePng generates a random rgba png image
func ImagePng(width int, height int) []byte { return imagePng(GlobalFaker, width, height) }

// ImagePng generates a random rgba png image
func (f *Faker) ImagePng(width int, height int) []byte { return imagePng(f, width, height) }

func imagePng(f *Faker, width int, height int) []byte {
	buf := new(bytes.Buffer)
	png.Encode(buf, image(f, width, height))
	return buf.Bytes()
}

func addImageLookup() {
	AddFuncLookup("imagejpeg", Info{
		Display:     "Image JPEG",
		Category:    "image",
		Description: "Image file format known for its efficient compression and compatibility",
		Example:     "file.jpeg - bytes",
		Output:      "[]byte",
		ContentType: "image/jpeg",
		Aliases:     []string{"jpeg", "jpg", "format", "compression", "compatibility", "photo"},
		Keywords:    []string{"image", "efficient", "file", "bytes", "known", "rgba", "pixel", "width", "height"},
		Params: []Param{
			{Field: "width", Display: "Width", Type: "int", Default: "500", Description: "Image width in px"},
			{Field: "height", Display: "Height", Type: "int", Default: "500", Description: "Image height in px"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			width, err := info.GetInt(m, "width")
			if err != nil {
				return nil, err
			}
			if width < 10 || width >= 1000 {
				return nil, errors.New("invalid image width, must be greater than 10, less than 1000")
			}

			height, err := info.GetInt(m, "height")
			if err != nil {
				return nil, err
			}
			if height < 10 || height >= 1000 {
				return nil, errors.New("invalid image height, must be greater than 10, less than 1000")
			}

			return imageJpeg(f, width, height), nil
		},
	})

	AddFuncLookup("imagepng", Info{
		Display:     "Image PNG",
		Category:    "image",
		Description: "Image file format known for its lossless compression and support for transparency",
		Example:     "file.png - bytes",
		Output:      "[]byte",
		ContentType: "image/png",
		Aliases:     []string{"png", "format", "lossless", "compression", "transparency", "graphic"},
		Keywords:    []string{"image", "support", "file", "bytes", "known", "rgba", "pixel", "width", "height"},
		Params: []Param{
			{Field: "width", Display: "Width", Type: "int", Default: "500", Description: "Image width in px"},
			{Field: "height", Display: "Height", Type: "int", Default: "500", Description: "Image height in px"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			width, err := info.GetInt(m, "width")
			if err != nil {
				return nil, err
			}
			if width < 10 || width >= 1000 {
				return nil, errors.New("invalid image width, must be greater than 10, less than 1000")
			}

			height, err := info.GetInt(m, "height")
			if err != nil {
				return nil, err
			}
			if height < 10 || height >= 1000 {
				return nil, errors.New("invalid image height, must be greater than 10, less than 1000")
			}

			return imagePng(f, width, height), nil
		},
	})
}
