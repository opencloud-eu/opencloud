# Imaging

This is pure Go code that makes working with images actually
useable on top of the Go stdlib. In addition to the usual PNG/JPEG/WebP/TIFF/BMP/GIF
formats that have been supported forever, this package adds support for 
animated PNG, animated WebP, Google's new "jpegli" JPEG variant
and all the netPBM image formats.

Additionally, this package support color management via ICC profiles and CICP
metadata. Opening non-sRGB images automatically converts them to sRGB, so you
don't have to think about it. It has full support for ICC v2 and v4 profiles
embedded in all the image formats and is extensively tested against the
little-cms library. 

It also supports loading image metadata in EXIF format and automatically
supports the EXIF orientation flag -- on image load the image is transformed
based on that tag automatically.

It automatically falls back to ImageMagick when available, for image formats
it does not support.

Finally, it provides basic image processing functions
(resize, rotate, crop, brightness/contrast adjustments, etc.).

## Installation

    go get -u github.com/kovidgoyal/imaging

## Documentation

https://pkg.go.dev/github.com/kovidgoyal/imaging

## Quickstart

```go
img, metadata, err := imaging.OpenAll(path, options...)
img.Resize(128, 128, imaging.Lanczos)
img.SaveAsPNG(path, mode)
```

There are also convenience scripts that demonstrate this library in action,
note that these are mainly for development and as such they only use the pure
Go code and do not fallback to ImageMagick:

```sh
./to-png some-image.whatever some-image.png
./to-frames some-animated-image.whatever some-animated-image.apng
```

Imaging supports image resizing using various resampling filters. The most notable ones:
- `Lanczos` - A high-quality resampling filter for photographic images yielding sharp results.
- `CatmullRom` - A sharp cubic filter that is faster than Lanczos filter while providing similar results.
- `MitchellNetravali` - A cubic filter that produces smoother results with less ringing artifacts than CatmullRom.
- `Linear` - Bilinear resampling filter, produces smooth output. Faster than cubic filters.
- `Box` - Simple and fast averaging filter appropriate for downscaling. When upscaling it's similar to NearestNeighbor.
- `NearestNeighbor` - Fastest resampling filter, no antialiasing.

The full list of supported filters:  NearestNeighbor, Box, Linear, Hermite, MitchellNetravali, CatmullRom, BSpline, Gaussian, Lanczos, Hann, Hamming, Blackman, Bartlett, Welch, Cosine. Custom filters can be created using ResampleFilter struct.

**Resampling filters comparison**

Original image:

![srcImage](testdata/branches.png)

The same image resized from 600x400px to 150x100px using different resampling filters.
From faster (lower quality) to slower (higher quality):

Filter                    | Resize result
--------------------------|---------------------------------------------
`imaging.NearestNeighbor` | ![dstImage](testdata/out_resize_nearest.png)
`imaging.Linear`          | ![dstImage](testdata/out_resize_linear.png)
`imaging.CatmullRom`      | ![dstImage](testdata/out_resize_catrom.png)
`imaging.Lanczos`         | ![dstImage](testdata/out_resize_lanczos.png)


### Gaussian Blur

```go
dstImage := imaging.Blur(srcImage, 0.5)
```

Sigma parameter allows to control the strength of the blurring effect.

Original image                     | Sigma = 0.5                            | Sigma = 1.5
-----------------------------------|----------------------------------------|---------------------------------------
![srcImage](testdata/flowers_small.png) | ![dstImage](testdata/out_blur_0.5.png) | ![dstImage](testdata/out_blur_1.5.png)

### Sharpening

```go
dstImage := imaging.Sharpen(srcImage, 0.5)
```

`Sharpen` uses gaussian function internally. Sigma parameter allows to control the strength of the sharpening effect.

Original image                     | Sigma = 0.5                               | Sigma = 1.5
-----------------------------------|-------------------------------------------|------------------------------------------
![srcImage](testdata/flowers_small.png) | ![dstImage](testdata/out_sharpen_0.5.png) | ![dstImage](testdata/out_sharpen_1.5.png)

### Gamma correction

```go
dstImage := imaging.AdjustGamma(srcImage, 0.75)
```

Original image                     | Gamma = 0.75                             | Gamma = 1.25
-----------------------------------|------------------------------------------|-----------------------------------------
![srcImage](testdata/flowers_small.png) | ![dstImage](testdata/out_gamma_0.75.png) | ![dstImage](testdata/out_gamma_1.25.png)

### Contrast adjustment

```go
dstImage := imaging.AdjustContrast(srcImage, 20)
```

Original image                     | Contrast = 15                              | Contrast = -15
-----------------------------------|--------------------------------------------|-------------------------------------------
![srcImage](testdata/flowers_small.png) | ![dstImage](testdata/out_contrast_p15.png) | ![dstImage](testdata/out_contrast_m15.png)

### Brightness adjustment

```go
dstImage := imaging.AdjustBrightness(srcImage, 20)
```

Original image                     | Brightness = 10                              | Brightness = -10
-----------------------------------|----------------------------------------------|---------------------------------------------
![srcImage](testdata/flowers_small.png) | ![dstImage](testdata/out_brightness_p10.png) | ![dstImage](testdata/out_brightness_m10.png)

### Saturation adjustment

```go
dstImage := imaging.AdjustSaturation(srcImage, 20)
```

Original image                     | Saturation = 30                              | Saturation = -30
-----------------------------------|----------------------------------------------|---------------------------------------------
![srcImage](testdata/flowers_small.png) | ![dstImage](testdata/out_saturation_p30.png) | ![dstImage](testdata/out_saturation_m30.png)

### Hue adjustment

```go
dstImage := imaging.AdjustHue(srcImage, 20)
```

Original image                     | Hue = 60                                     | Hue = -60
-----------------------------------|----------------------------------------------|---------------------------------------------
![srcImage](testdata/flowers_small.png) | ![dstImage](testdata/out_hue_p60.png) | ![dstImage](testdata/out_hue_m60.png)


## Acknowledgements

This is a fork of the un-maintained distraction/imaging project. The color
management code was started out from mandykoh/prism and used some code from
go-andiamo/iccarus but it was almost completely re-written from scratch.
