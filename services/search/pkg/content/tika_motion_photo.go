package content

import (
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"sort"
	"strconv"
	"strings"
)

// motionPhotoVideoName is the resource name tika gives the appended video when
// it emits it as an embedded attachment. The extension follows the detected
// type and is absent for legacy MicroVideo, which declares no mime type.
const motionPhotoVideoName = "motion-photo"

// getMotionPhoto reads Google Motion Photo XMP, which Tika exposes under the
// canonical Camera/Container prefixes. It covers both the current MotionPhoto
// scheme and the legacy MicroVideo scheme. videoSize (the embedded video's byte
// length, needed to range-fetch it) is required, so the facet is dropped without it.
func (t Tika) getMotionPhoto(meta map[string][]string) *libregraph.MotionPhoto {
	// per the spec only a MotionPhoto/MicroVideo marker of 1 means motion
	// photo, every other value is "treat as a still image". An absent marker
	// is tolerated on purpose, the byte-level video check below decides.
	if v, err := getFirstValue(meta, "Camera:MotionPhoto", "Camera:MicroVideo"); err == nil && v != "1" {
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

	if size, ok := motionPhotoVideoSize(meta); ok {
		initMotionPhoto()
		motionPhoto.SetVideoSize(size)
	}

	if motionPhoto == nil || !motionPhoto.HasVideoSize() {
		return nil
	}
	return motionPhoto
}

// motionPhotoVideoSize returns the embedded video's byte length: the length of
// the Container item whose semantic is "MotionPhoto", or for legacy files the
// MicroVideo offset (bytes from EOF to the video start, which equals its
// length). The current scheme wins when both are present, like everywhere else.
func motionPhotoVideoSize(meta map[string][]string) (int64, bool) {
	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	// map order is random, the first matching container item must be stable
	sort.Strings(keys)
	for _, k := range keys {
		if vals := meta[k]; !strings.HasSuffix(k, "/Item:Semantic") || len(vals) == 0 || vals[0] != "MotionPhoto" {
			continue
		}
		if v, err := getFirstValue(meta, strings.TrimSuffix(k, "/Item:Semantic")+"/Item:Length"); err == nil {
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i, true
			}
		}
	}
	if v, err := getFirstValue(meta, "Camera:MicroVideoOffset"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

// isMotionPhotoVideo reports whether meta describes the video tika extracted
// from a motion photo. Tika only emits it when the bytes the xmp advertises are
// really there, so its presence is what confirms the facet: a shared motion
// photo can keep the xmp and lose the appended video.
func isMotionPhotoVideo(meta map[string][]string) bool {
	name, err := getFirstValue(meta, "tk:resource-name")
	if err != nil {
		return false
	}
	return name == motionPhotoVideoName || strings.HasPrefix(name, motionPhotoVideoName+".")
}
