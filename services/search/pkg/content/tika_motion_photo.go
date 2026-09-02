package content

import (
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"strconv"
	"strings"
)

// getMotionPhoto reads Google Motion Photo XMP, which Tika exposes under the
// canonical Camera/Container prefixes. It covers both the current MotionPhoto
// scheme and the legacy MicroVideo scheme. videoSize (needed to range-fetch the
// video) comes from the video tika extracted, and is required: without it the
// facet is dropped.
func (t Tika) getMotionPhoto(meta, video map[string][]string) *libregraph.MotionPhoto {
	// the marker is what makes this a motion photo rather than a picture that
	// happens to carry a video: per the spec only a value of 1 counts, every
	// other value means "treat as a still image".
	if v, err := getFirstValue(meta, "Camera:MotionPhoto", "Camera:MicroVideo"); err != nil || v != "1" {
		return nil
	}

	var motionPhoto *libregraph.MotionPhoto
	initMotionPhoto := func() {
		if motionPhoto == nil {
			motionPhoto = libregraph.NewMotionPhoto()
		}
	}

	if v, err := getFirstValue(meta, "Camera:MotionPhotoVersion", "Camera:MicroVideoVersion"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initMotionPhoto()
			motionPhoto.SetVersion(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "Camera:MotionPhotoPresentationTimestampUs", "Camera:MicroVideoPresentationTimestampUs"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			initMotionPhoto()
			motionPhoto.SetPresentationTimestampUs(i)
		}
	}

	if v, err := getFirstValue(video, "Content-Length"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			initMotionPhoto()
			motionPhoto.SetVideoSize(i)
		}
	}

	if motionPhoto == nil || !motionPhoto.HasVideoSize() {
		return nil
	}
	return motionPhoto
}

// isVideo reports whether meta describes a video. Tika emits the video appended
// to a motion photo as an embedded document, and it only does so when the bytes
// the xmp advertises are really there: a shared motion photo can keep the xmp
// and lose the video.
func isVideo(meta map[string][]string) bool {
	v, err := getFirstValue(meta, "Content-Type")
	return err == nil && strings.HasPrefix(v, "video/")
}
