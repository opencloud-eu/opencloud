package content

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/go-tika/tika"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

// Tika is used to extract content from a resource,
// it uses apache tika to retrieve all the data.
type Tika struct {
	*Basic
	Retriever
	tika                       *tika.Client
	ContentExtractionSizeLimit uint64
	CleanStopWords             bool
}

// NewTikaExtractor creates a new Tika instance.
func NewTikaExtractor(gatewaySelector pool.Selectable[gateway.GatewayAPIClient], logger log.Logger, cfg *config.Config) (*Tika, error) {
	basic, err := NewBasicExtractor(logger)
	if err != nil {
		return nil, err
	}

	tk := tika.NewClient(nil, cfg.Extractor.Tika.TikaURL)
	tkv, err := tk.Version(context.Background())
	if err != nil {
		return nil, err
	}
	logger.Info().Msgf("Tika version: %s", tkv)

	return &Tika{
		Basic:                      basic,
		Retriever:                  newCS3Retriever(gatewaySelector, logger, cfg.Extractor.CS3AllowInsecure),
		tika:                       tika.NewClient(nil, cfg.Extractor.Tika.TikaURL),
		ContentExtractionSizeLimit: cfg.ContentExtractionSizeLimit,
		CleanStopWords:             cfg.Extractor.Tika.CleanStopWords,
	}, nil
}

// Extract loads a resource from its underlying storage, passes it to tika and processes the result into a Document.
func (t Tika) Extract(ctx context.Context, ri *provider.ResourceInfo) (Document, error) {
	doc, err := t.Basic.Extract(ctx, ri)
	if err != nil {
		return doc, err
	}

	if ri.Size == 0 {
		return doc, nil
	}

	if ri.Size > t.ContentExtractionSizeLimit {
		t.logger.Info().Interface("ResourceID", ri.Id).Str("Name", ri.Name).Msg("file exceeds content extraction size limit. skipping.")
		return doc, nil
	}

	if ri.Type != provider.ResourceType_RESOURCE_TYPE_FILE {
		return doc, nil
	}

	data, err := t.Retrieve(ctx, ri.Id)
	if err != nil {
		return doc, err
	}
	defer data.Close()

	metas, err := t.tika.MetaRecursive(ctx, data)
	if err != nil {
		return doc, err
	}

	for _, meta := range metas {
		if title, err := getFirstValue(meta, "title"); err == nil {
			doc.Title = strings.TrimSpace(fmt.Sprintf("%s %s", doc.Title, title))
		}

		if content, err := getFirstValue(meta, "X-TIKA:content"); err == nil {
			doc.Content = strings.TrimSpace(fmt.Sprintf("%s %s", doc.Content, content))
		}

		doc.Location = t.getLocation(meta)
		doc.Image = t.getImage(meta)
		doc.Photo = t.getPhoto(meta)

		if contentType, err := getFirstValue(meta, "Content-Type"); err == nil && strings.HasPrefix(contentType, "audio/") {
			doc.Audio = t.getAudio(meta)
		}
	}

	if langCode, _ := t.tika.LanguageString(ctx, doc.Content); langCode != "" && t.CleanStopWords {
		doc.Content = CleanString(doc.Content, langCode)
	}

	return doc, nil
}

func (t Tika) getImage(meta map[string][]string) *libregraph.Image {
	var image *libregraph.Image
	initImage := func() {
		if image == nil {
			image = libregraph.NewImage()
		}
	}

	if v, err := getFirstValue(meta, "tiff:ImageWidth"); err == nil {
		if i, err := strconv.ParseInt(v, 0, 32); err == nil {
			initImage()
			image.SetWidth(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "tiff:ImageLength"); err == nil {
		if i, err := strconv.ParseInt(v, 0, 32); err == nil {
			initImage()
			image.SetHeight(int32(i))
		}
	}

	return image
}

func (t Tika) getLocation(meta map[string][]string) *libregraph.GeoCoordinates {
	var location *libregraph.GeoCoordinates
	initLocation := func() {
		if location == nil {
			location = libregraph.NewGeoCoordinates()
		}
	}

	// TODO: location.Altitute: transform the following data to … feet above sea level.
	// "GPS:GPS Altitude":                          []string{"227.4 metres"},
	// "GPS:GPS Altitude Ref":                      []string{"Sea level"},

	if v, err := getFirstValue(meta, "geo:lat"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initLocation()
			location.SetLatitude(i)
		}
	}

	if v, err := getFirstValue(meta, "geo:long"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initLocation()
			location.SetLongitude(i)
		}
	}

	return location
}

func (t Tika) getPhoto(meta map[string][]string) *libregraph.Photo {
	var photo *libregraph.Photo
	initPhoto := func() {
		if photo == nil {
			photo = libregraph.NewPhoto()
		}
	}

	if v, err := getFirstValue(meta, "tiff:Make"); err == nil {
		initPhoto()
		photo.SetCameraMake(v)
	}

	if v, err := getFirstValue(meta, "tiff:Model"); err == nil {
		initPhoto()
		photo.SetCameraModel(v)
	}

	if v, err := getFirstValue(meta, "exif:FNumber"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initPhoto()
			photo.SetFNumber(i)
		}
	}

	if v, err := getFirstValue(meta, "exif:FocalLength"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initPhoto()
			photo.SetFocalLength(i)
		}
	}

	if v, err := getFirstValue(meta, "Base ISO"); err == nil {
		if i, err := strconv.ParseInt(v, 0, 32); err == nil {
			initPhoto()
			photo.SetIso(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "tiff:Orientation"); err == nil {
		if i, err := strconv.ParseInt(v, 0, 32); err == nil {
			initPhoto()
			photo.SetOrientation(int32(i))
		}
	}

	if v, err := getFirstValue(meta, "exif:DateTimeOriginal"); err == nil {
		layout := "2006-01-02T15:04:05"
		if t, err := time.Parse(layout, v); err == nil {
			initPhoto()
			photo.SetTakenDateTime(t)
		}
	}

	if v, err := getFirstValue(meta, "exif:ExposureTime"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initPhoto()
			photo.SetExposureNumerator(1)
			photo.SetExposureDenominator(math.Round(1 / i))
		}
	}

	return photo
}

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

	// TODO: audio.Bitrate: not provided by tika
	// TODO: audio.Composers: not provided by tika
	// TODO: audio.Copyright: not provided by tika for audio files?

	if v, err := getFirstValue(meta, "xmpDM:discNumber"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initAudio()
			audio.SetDisc(int32(i))
		}

	}

	//  TODO: audio.DiscCount: not provided by tika

	if v, err := getFirstValue(meta, "xmpDM:duration"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			initAudio()
			audio.SetDuration(i * 1000)
		}
	}

	if v, err := getFirstValue(meta, "xmpDM:genre"); err == nil {
		initAudio()
		audio.SetGenre(v)
	}

	// TODO: audio.HasDrm: not provided by tika
	// TODO: audio.IsVariableBitrate: not provided by tika

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

	// TODO: audio.TrackCount: not provided by tika

	if v, err := getFirstValue(meta, "xmpDM:releaseDate"); err == nil {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			initAudio()
			audio.SetYear(int32(i))
		}
	}

	return audio
}
