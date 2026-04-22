package vips

// #include "image.h"
import "C"

import (
	"fmt"
	"runtime"
)

// GetICCProfile retrieves the ICC profile data (if any) from the image.
func (r *ImageRef) GetICCProfile() []byte {
	defer runtime.KeepAlive(r)
	bytes, _ := vipsGetICCProfile(r.image)
	return bytes
}

// RemoveICCProfile removes the ICC Profile information from the image.
// Typically, browsers and other software assume images without profile to be in the sRGB color space.
func (r *ImageRef) RemoveICCProfile() error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCopy(r.image, nil)
	if err != nil {
		return err
	}

	vipsRemoveICCProfile(out)

	r.setImage(out)
	return nil
}

// TransformICCProfileWithFallback transforms from the embedded ICC profile of the image to the ICC profile at the given path.
// The fallback ICC profile is used if the image does not have an embedded ICC profile.
func (r *ImageRef) TransformICCProfileWithFallback(targetProfilePath, fallbackProfilePath string) error {
	defer runtime.KeepAlive(r)
	if err := ensureLoadICCPath(&targetProfilePath); err != nil {
		return err
	}
	if err := ensureLoadICCPath(&fallbackProfilePath); err != nil {
		return err
	}

	depth := 16
	if r.BandFormat() == BandFormatUchar || r.BandFormat() == BandFormatChar || r.BandFormat() == BandFormatNotSet {
		depth = 8
	}

	out, err := vipsICCTransform(r.image, targetProfilePath, fallbackProfilePath, IntentPerceptual, depth, true)
	if err != nil {
		govipsLog("govips", LogLevelError, fmt.Sprintf("failed to do icc transform: %v", err.Error()))
		return err
	}

	r.setImage(out)
	return nil
}

// TransformICCProfile transforms from the embedded ICC profile of the image to the icc profile at the given path.
func (r *ImageRef) TransformICCProfile(outputProfilePath string) error {
	return r.TransformICCProfileWithFallback(outputProfilePath, SRGBIEC6196621ICCProfilePath)
}

// OptimizeICCProfile optimizes the ICC color profile of the image.
// For two color channel images, it sets a grayscale profile.
// For color images, it sets a CMYK or non-CMYK profile based on the image metadata.
func (r *ImageRef) OptimizeICCProfile() error {
	defer runtime.KeepAlive(r)
	inputProfile := r.determineInputICCProfile()
	if !r.HasICCProfile() && (inputProfile == "") {
		// No embedded ICC profile in the input image and no input profile determined, nothing to do.
		return nil
	}

	r.optimizedIccProfile = SRGBV2MicroICCProfilePath
	if r.Bands() <= 2 {
		r.optimizedIccProfile = SGrayV2MicroICCProfilePath
	}

	if err := ensureLoadICCPath(&r.optimizedIccProfile); err != nil {
		return err
	}

	embedded := r.HasICCProfile() && (inputProfile == "")

	depth := 16
	if r.BandFormat() == BandFormatUchar || r.BandFormat() == BandFormatChar || r.BandFormat() == BandFormatNotSet {
		depth = 8
	}

	out, err := vipsICCTransform(r.image, r.optimizedIccProfile, inputProfile, IntentPerceptual, depth, embedded)
	if err != nil {
		govipsLog("govips", LogLevelError, fmt.Sprintf("failed to do icc transform: %v", err.Error()))
		return err
	}

	r.setImage(out)
	return nil
}

func (r *ImageRef) determineInputICCProfile() (inputProfile string) {
	if r.Interpretation() == InterpretationCMYK {
		if !r.HasICCProfile() {
			inputProfile = "cmyk"
		}
	}
	return
}
