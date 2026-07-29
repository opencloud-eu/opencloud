package content

import (
	"math"
	"strconv"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func (t Tika) getAudio(meta map[string][]string) *libregraph.Audio {
	var audio *libregraph.Audio
	initAudio := func() {
		if audio == nil {
			audio = libregraph.NewAudio()
		}
	}

	if v, err := getFirstValue(meta, "xmpDM:album"); err == nil {
		initAudio()
		audio.SetAlbum(v)
	}

	if v, err := getFirstValue(meta, "xmpDM:albumArtist"); err == nil {
		initAudio()
		audio.SetAlbumArtist(v)
	}

	if v, err := getFirstValue(meta, "xmpDM:artist"); err == nil {
		initAudio()
		audio.SetArtist(v)
	}

	if v, err := getFirstValue(meta, "audio:bitrate"); err == nil {
		// tika emits bits per second, graph wants kbps
		if bps, err := strconv.ParseInt(v, 10, 64); err == nil {
			initAudio()
			audio.SetBitrate(bps / 1000)
		}
	}

	if v, err := getFirstValue(meta, "xmpDM:composer"); err == nil {
		initAudio()
		audio.SetComposers(v)
	}

	if v, err := getFirstValue(meta, "xmpDM:copyright"); err == nil {
		initAudio()
		audio.SetCopyright(v)
	}

	if v, err := getFirstValue(meta, "xmpDM:discNumber"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initAudio()
			audio.SetDisc(int32(i))
		}

	}

	if v, err := getFirstValue(meta, "audio:disc-count"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initAudio()
			audio.SetDiscCount(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "xmpDM:duration"); err == nil {
		// Tika emits fractional seconds.
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			initAudio()
			audio.SetDuration(int64(math.Round(f * 1000)))
		}
	}

	if v, err := getFirstValue(meta, "xmpDM:genre"); err == nil {
		initAudio()
		audio.SetGenre(v)
	}

	if v, err := getFirstValue(meta, "audio:has-drm"); err == nil {
		if b, err := strconv.ParseBool(v); err == nil {
			initAudio()
			audio.SetHasDrm(b)
		}
	}

	if v, err := getFirstValue(meta, "audio:is-variable-bitrate"); err == nil {
		if b, err := strconv.ParseBool(v); err == nil {
			initAudio()
			audio.SetIsVariableBitrate(b)
		}
	}

	if v, err := getFirstValue(meta, "dc:title"); err == nil {
		initAudio()
		audio.SetTitle(v)
	}

	if v, err := getFirstValue(meta, "xmpDM:trackNumber"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initAudio()
			audio.SetTrack(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "audio:track-count"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initAudio()
			audio.SetTrackCount(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "xmpDM:releaseDate"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initAudio()
			audio.SetYear(int32(i))
		}
	}

	return audio
}
