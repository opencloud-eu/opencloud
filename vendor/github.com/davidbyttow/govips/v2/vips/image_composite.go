package vips

// #include "image.h"
import "C"

import "runtime"

// CompositeMulti composites the given overlay image on top of the associated image with provided blending mode.
func (r *ImageRef) CompositeMulti(ins []*ImageComposite) error {
	defer runtime.KeepAlive(r)
	out, err := vipsComposite(toVipsCompositeStructs(r, ins))
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Composite composites the given overlay image on top of the associated image with provided blending mode.
func (r *ImageRef) Composite(overlay *ImageRef, mode BlendMode, x, y int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenComposite2(r.image, overlay.image, mode, &Composite2Options{X: &x, Y: &y})
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Insert draws the image on top of the associated image at the given coordinates.
func (r *ImageRef) Insert(sub *ImageRef, x, y int, expand bool, background *ColorRGBA) error {
	defer runtime.KeepAlive(r)
	insertOpts := &InsertOptions{Expand: &expand}
	if background != nil {
		insertOpts.Background = []float64{float64(background.R), float64(background.G), float64(background.B), float64(background.A)}
	}
	out, err := vipsGenInsert(r.image, sub.image, x, y, insertOpts)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Join joins this image with another in the direction specified
func (r *ImageRef) Join(in *ImageRef, dir Direction) error {
	defer runtime.KeepAlive(r)
	out, err := vipsJoin(r.image, in.image, dir)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// ArrayJoin joins an array of images together wrapping at each n images
func (r *ImageRef) ArrayJoin(images []*ImageRef, across int) error {
	defer runtime.KeepAlive(r)
	allImages := append([]*ImageRef{r}, images...)
	inputs := make([]*C.VipsImage, len(allImages))
	for i := range inputs {
		inputs[i] = allImages[i].image
	}
	out, err := vipsGenArrayjoin(inputs, &ArrayjoinOptions{Across: &across})
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}
