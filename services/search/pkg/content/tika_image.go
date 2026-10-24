package content

import (
	"strconv"
	"strings"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func (t Tika) getImage(meta map[string][]string) *libregraph.Image {
	// tiff:ImageWidth/Length are also set for videos; the content type is what
	// tells an image apart.
	if ct, err := getFirstValue(meta, "Content-Type"); err != nil || !strings.HasPrefix(ct, "image/") {
		return nil
	}

	var image *libregraph.Image
	initImage := func() {
		if image == nil {
			image = libregraph.NewImage()
		}
	}

	if v, err := getFirstValue(meta, "tiff:ImageWidth"); err == nil {
		if i, err := strconv.ParseInt(v, 0, 32); err == nil {
			initImage()
			image.SetWidth(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "tiff:ImageLength"); err == nil {
		if i, err := strconv.ParseInt(v, 0, 32); err == nil {
			initImage()
			image.SetHeight(int32(i))
		}
	}

	return image
}
