package svc

import "math"

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
