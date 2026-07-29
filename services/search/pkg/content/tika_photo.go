package content

import (
	"math"
	"strconv"
	"time"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func (t Tika) getPhoto(meta map[string][]string) *libregraph.Photo {
	var photo *libregraph.Photo
	initPhoto := func() {
		if photo == nil {
			photo = libregraph.NewPhoto()
		}
	}

	if v, err := getFirstValue(meta, "tiff:Make"); err == nil {
		initPhoto()
		photo.SetCameraMake(v)
	}

	if v, err := getFirstValue(meta, "tiff:Model"); err == nil {
		initPhoto()
		photo.SetCameraModel(v)
	}

	if v, err := getFirstValue(meta, "exif:FNumber"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initPhoto()
			photo.SetFNumber(i)
		}
	}

	if v, err := getFirstValue(meta, "exif:FocalLength"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initPhoto()
			photo.SetFocalLength(i)
		}
	}

	if v, err := getFirstValue(meta, "Base ISO"); err == nil {
		if i, err := strconv.ParseInt(v, 0, 32); err == nil {
			initPhoto()
			photo.SetIso(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "tiff:Orientation"); err == nil {
		if i, err := strconv.ParseInt(v, 0, 32); err == nil {
			initPhoto()
			photo.SetOrientation(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "exif:DateTimeOriginal"); err == nil {
		layout := "2006-01-02T15:04:05"
		if t, err := time.Parse(layout, v); err == nil {
			initPhoto()
			photo.SetTakenDateTime(t)
		}
	}

	if v, err := getFirstValue(meta, "exif:ExposureTime"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initPhoto()
			photo.SetExposureNumerator(1)
			photo.SetExposureDenominator(math.Round(1 / i))
		}
	}

	return photo
}
