package vips

// #include "image.h"
import "C"

import (
	"errors"
	"runtime"
)

// ToColorSpace changes the color space of the image to the interpretation supplied as the parameter.
func (r *ImageRef) ToColorSpace(interpretation Interpretation) error {
	defer runtime.KeepAlive(r)
	out, err := vipsToColorSpace(r.image, interpretation)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Flatten removes the alpha channel from the image and replaces it with the background color
func (r *ImageRef) Flatten(backgroundColor *Color) error {
	defer runtime.KeepAlive(r)
	opts := &FlattenOptions{}
	if backgroundColor != nil {
		opts.Background = []float64{float64(backgroundColor.R), float64(backgroundColor.G), float64(backgroundColor.B)}
	}
	out, err := vipsGenFlatten(r.image, opts)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Modulate the colors
func (r *ImageRef) Modulate(brightness, saturation, hue float64) error {
	defer runtime.KeepAlive(r)
	var err error
	var multiplications []float64
	var additions []float64

	colorspace := r.ColorSpace()
	if colorspace == InterpretationRGB {
		colorspace = InterpretationSRGB
	}

	multiplications = []float64{brightness, saturation, 1}
	additions = []float64{0, 0, hue}

	if r.HasAlpha() {
		multiplications = append(multiplications, 1)
		additions = append(additions, 0)
	}

	err = r.ToColorSpace(InterpretationLCH)
	if err != nil {
		return err
	}

	err = r.Linear(multiplications, additions)
	if err != nil {
		return err
	}

	err = r.ToColorSpace(colorspace)
	if err != nil {
		return err
	}

	return nil
}

// ModulateHSV modulates the image HSV values based on the supplier parameters.
func (r *ImageRef) ModulateHSV(brightness, saturation float64, hue int) error {
	defer runtime.KeepAlive(r)
	var err error
	var multiplications []float64
	var additions []float64

	colorspace := r.ColorSpace()
	if colorspace == InterpretationRGB {
		colorspace = InterpretationSRGB
	}

	if r.HasAlpha() {
		multiplications = []float64{1, saturation, brightness, 1}
		additions = []float64{float64(hue), 0, 0, 0}
	} else {
		multiplications = []float64{1, saturation, brightness}
		additions = []float64{float64(hue), 0, 0}
	}

	err = r.ToColorSpace(InterpretationHSV)
	if err != nil {
		return err
	}

	err = r.Linear(multiplications, additions)
	if err != nil {
		return err
	}

	err = r.ToColorSpace(colorspace)
	if err != nil {
		return err
	}

	return nil
}

// Invert inverts the image
func (r *ImageRef) Invert() error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenInvert(r.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Adjusts the image's gamma value.
// See https://www.libvips.org/API/current/libvips-conversion.html#vips-gamma
func (r *ImageRef) Gamma(gamma float64) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenGamma(r.image, &GammaOptions{Exponent: &gamma})
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Linear passes an image through a linear transformation (i.e. output = input * a + b).
// See https://libvips.github.io/libvips/API/current/libvips-arithmetic.html#vips-linear
func (r *ImageRef) Linear(a, b []float64) error {
	defer runtime.KeepAlive(r)
	if len(a) != len(b) {
		return errors.New("a and b must be of same length")
	}

	out, err := vipsGenLinear(r.image, a, b, nil)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Linear1 runs Linear() with a single constant.
// See https://libvips.github.io/libvips/API/current/libvips-arithmetic.html#vips-linear1
func (r *ImageRef) Linear1(a, b float64) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenLinear(r.image, []float64{a}, []float64{b}, nil)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Cast converts the image to a target band format
func (r *ImageRef) Cast(format BandFormat) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCast(r.image, format, nil)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}
