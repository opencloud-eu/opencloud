package content

import (
	"math"
	"strconv"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func (t Tika) getLivePhoto(meta map[string][]string) *libregraph.LivePhoto {
	// ContentId pairs the two halves and is what makes it a live photo, so without
	// it there is no facet. The video exposes it via the QuickTime item list; the
	// still image carries it in the Apple maker note (tag 0x0011), which tika
	// surfaces generically under "Content Identifier" (the HEIC maker note is
	// parsed since metadata-extractor 2.21.0). A file is only ever one half, so
	// reading both keys covers both.
	contentID, err := getFirstValue(meta, "com.apple.quicktime.content.identifier", "Content Identifier")
	if err != nil || contentID == "" {
		return nil
	}
	livePhoto := libregraph.NewLivePhoto(contentID)

	// tika emits still-image-time already in microseconds
	if v, err := getFirstValue(meta, "quicktime:still-image-time"); err == nil {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			livePhoto.SetStillImageTimeUs(int64(math.Round(f)))
		}
	}

	if v, err := getFirstValue(meta, "com.apple.quicktime.live-photo.auto"); err == nil {
		if b, err := strconv.ParseBool(v); err == nil {
			livePhoto.SetAuto(b)
		}
	}

	if v, err := getFirstValue(meta, "com.apple.quicktime.live-photo.vitality-score"); err == nil {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			livePhoto.SetVitalityScore(f)
		}
	}

	if v, err := getFirstValue(meta, "com.apple.quicktime.live-photo.vitality-scoring-version"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			livePhoto.SetVitalityScoringVersion(i)
		}
	}

	return livePhoto
}
