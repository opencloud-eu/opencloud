package svc

import (
	"errors"
	"image"
	"math"

	"github.com/kovidgoyal/imaging"
)

// fitInBox is the size an image of srcW x srcH takes inside the box, aspect
// ratio preserved and the scaled side rounded the way vips and imagor round it.
func fitInBox(srcW, srcH, boxW, boxH int) (int, int) {
	scale := math.Min(float64(boxW)/float64(srcW), float64(boxH)/float64(srcH))

	return max(int(math.Round(float64(srcW)*scale)), 1), max(int(math.Round(float64(srcH)*scale)), 1)
}

// autoAxis resolves imagor's zero axis: a side given as 0 is computed from the
// other one, so the aspect ratio decides it, and two zero sides mean the image
// is returned at its own size. The bool reports whether anything is left to
// resize, and a resolved box is reached by a plain resize for every operation,
// because with a free axis there is nothing to crop to.
func autoAxis(srcW, srcH, boxW, boxH int) (int, int, bool) {
	switch {
	case boxW == 0 && boxH == 0:
		return srcW, srcH, false
	case boxW == 0:
		return max(int(math.Round(float64(srcW)*float64(boxH)/float64(srcH))), 1), boxH, true
	default:
		return boxW, max(int(math.Round(float64(srcH)*float64(boxW)/float64(srcW))), 1), true
	}
}

// errBoxTooLarge is what a resize answers when a free axis resolves past the
// configured maximum thumbnail size.
var errBoxTooLarge = errors.New("the resolved box exceeds the maximum thumbnail size")

// freeAxis is autoAxis for the request, held to the maximum thumbnail size. The
// box the request names was checked before anything was read, but a side the
// aspect ratio decides is only known once the image is: 0x2048 on a 7680x1
// source is a box of 15728640x2048. 0x0 leaves both sides to the source, so the
// source size is held to it too, even though nothing is resized.
func (r resizeRequest) freeAxis(srcW, srcH int) (int, int, bool, error) {
	w, h, resize := autoAxis(srcW, srcH, r.Width, r.Height)
	if (r.MaxOutWidth > 0 && w > r.MaxOutWidth) || (r.MaxOutHeight > 0 && h > r.MaxOutHeight) {
		return 0, 0, false, errBoxTooLarge
	}

	return w, h, resize, nil
}

// fitIn scales img into the box with its aspect ratio kept and never enlarges
// it, which is imagor's fit-in without the upscale() filter. Not imaging.Fit:
// that truncates the scaled side where vips and imagor round it.
func fitIn(img image.Image, boxW, boxH int) image.Image {
	b := img.Bounds()

	width, height := fitInBox(b.Dx(), b.Dy(), boxW, boxH)
	if width >= b.Dx() && height >= b.Dy() {
		return img
	}

	return imaging.Resize(img, width, height, imaging.Lanczos)
}
