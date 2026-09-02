package content

import (
	"math"
	"strconv"
	"strings"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func (t Tika) getVideo(meta map[string][]string) *libregraph.Video {
	// the tiff:/xmpDM: keys below are shared with images and audio; the file's
	// content type is what tells a video apart.
	if ct, err := getFirstValue(meta, "Content-Type"); err != nil || !strings.HasPrefix(ct, "video/") {
		return nil
	}

	var video *libregraph.Video
	initVideo := func() {
		if video == nil {
			video = libregraph.NewVideo()
		}
	}

	if v, err := getFirstValue(meta, "tiff:ImageWidth"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initVideo()
			video.SetWidth(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "tiff:ImageLength"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initVideo()
			video.SetHeight(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "xmpDM:duration"); err == nil {
		// Tika emits fractional seconds.
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			initVideo()
			video.SetDuration(int64(math.Round(f * 1000)))
		}
	}

	// video:fourcc lands with tika 4.1 (TIKA-4838); the xmpDM compressor name
	// ("AVC Coding") is no FourCC, so there is no earlier source
	if v, err := getFirstValue(meta, "video:fourcc"); err == nil {
		initVideo()
		video.SetFourCC(v)
	}

	// tika emits bits per second, matching the graph video facet
	if v, err := getFirstValue(meta, "video:bitrate"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil && i > 0 {
			initVideo()
			video.SetBitrate(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "video:frame-rate"); err == nil {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && !math.IsInf(f, 0) {
			initVideo()
			video.SetFrameRate(f)
		}
	}

	if v, err := getFirstValue(meta, "xmpDM:audioSampleRate"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initVideo()
			video.SetAudioSamplesPerSecond(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "audio:bits-per-sample"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initVideo()
			video.SetAudioBitsPerSample(int32(i))
		}
	}

	// prefer the numeric track count, fall back to the xmpDM enum
	if v, err := getFirstValue(meta, "audio:channels"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil && i > 0 {
			initVideo()
			video.SetAudioChannels(int32(i))
		}
	} else if c, ok := audioChannelCount(meta); ok {
		initVideo()
		video.SetAudioChannels(c)
	}

	// audio:fourcc also lands with tika 4.1 (TIKA-4838); until then the audio
	// codec of a video file is unavailable
	if v, err := getFirstValue(meta, "audio:fourcc"); err == nil {
		initVideo()
		video.SetAudioFormat(v)
	}

	return video
}

// audioChannelCount maps tika's xmpDM:audioChannelType enum to a channel count.
// A numeric channel count from tika would drop this mapping and cover >2 channels.
func audioChannelCount(meta map[string][]string) (int32, bool) {
	v, err := getFirstValue(meta, "xmpDM:audioChannelType")
	if err != nil {
		return 0, false
	}
	switch v {
	case "Mono":
		return 1, true
	case "Stereo":
		return 2, true
	case "5.1":
		return 6, true
	case "7.1":
		return 8, true
	}
	return 0, false
}
