package content

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/go-tika/tika"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

// Tika is used to extract content from a resource.
// Supports both Apache Tika and open_taki (drop-in replacement).
// When open_taki is detected, uses the v2 protocol for enhanced extraction.
type Tika struct {
	*Basic
	Retriever
	tika                       *tika.Client
	tikaURL                    string
	httpClient                 *http.Client
	ContentExtractionSizeLimit uint64
	CleanStopWords             bool
	isTaki                     bool
}

// NewTikaExtractor creates a new Tika instance.
func NewTikaExtractor(gatewaySelector pool.Selectable[gateway.GatewayAPIClient], logger log.Logger, cfg *config.Config) (*Tika, error) {
	basic, err := NewBasicExtractor(logger)
	if err != nil {
		return nil, err
	}

	tikaURL := cfg.Extractor.Tika.TikaURL

	tk := tika.NewClient(nil, tikaURL)
	tkv, err := tk.Version(context.Background())
	if err != nil {
		return nil, err
	}

	// Detect open_taki by checking the health endpoint
	isTaki := false
	hc := &http.Client{Timeout: 5 * time.Second}
	if resp, err := hc.Get(tikaURL + "/tika"); err == nil {
		defer resp.Body.Close()
		var health map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&health) == nil {
			if name, ok := health["name"].(string); ok && name == "open_taki" {
				isTaki = true
				logger.Info().Msgf("open_taki detected (version: %v), using v2 protocol", health["version"])
			}
		}
	}

	if !isTaki {
		logger.Info().Msgf("Tika version: %s", tkv)
	}

	return &Tika{
		Basic:                      basic,
		Retriever:                  newCS3Retriever(gatewaySelector, logger, cfg.Extractor.CS3AllowInsecure),
		tika:                       tk,
		tikaURL:                    tikaURL,
		httpClient:                 &http.Client{Timeout: 5 * time.Minute},
		ContentExtractionSizeLimit: cfg.ContentExtractionSizeLimit,
		CleanStopWords:             cfg.Extractor.Tika.CleanStopWords,
		isTaki:                     isTaki,
	}, nil
}

// Extract loads a resource from its underlying storage, passes it to tika/taki
// and processes the result into a Document.
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

	if t.isTaki {
		return t.extractTakiV2(ctx, ri, data, doc)
	}
	return t.extractTikaV1(ctx, ri, data, doc)
}

// extractTikaV1 is the original Tika extraction path (unchanged behavior).
func (t Tika) extractTikaV1(ctx context.Context, ri *provider.ResourceInfo, data io.ReadCloser, doc Document) (Document, error) {
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

// takiV2Response matches open_taki's v2 JSON response format.
type takiV2Response struct {
	Content     string `json:"X-TIKA:content"`
	ContentType string `json:"Content-Type"`
	Method      string `json:"X-TAKI:method"`
	Meta        *struct {
		Title    string `json:"title"`
		Author   string `json:"author"`
		Created  string `json:"created"`
		Language string `json:"language"`
		DocType  string `json:"doc_type"`
	} `json:"X-TAKI:meta"`
	Entities []Entity  `json:"X-TAKI:entities"`
	Summary  string    `json:"X-TAKI:summary"`
	Embed    []float64 `json:"X-TAKI:embedding"`
	Routing  *struct {
		ContentTarget string `json:"content_target"`
		MetaTarget    string `json:"meta_target"`
		VectorTarget  string `json:"vector_target"`
		SourceRef     string `json:"source_ref"`
	} `json:"X-TAKI:routing"`
}

// extractTakiV2 uses the open_taki v2 protocol for enhanced extraction.
func (t Tika) extractTakiV2(ctx context.Context, ri *provider.ResourceInfo, data io.ReadCloser, doc Document) (Document, error) {
	body, err := io.ReadAll(data)
	if err != nil {
		return doc, fmt.Errorf("reading resource data: %w", err)
	}

	sourceRef := storagespace.FormatResourceID(ri.Id)

	req, err := http.NewRequestWithContext(ctx, "PUT",
		t.tikaURL+"/rmeta/text", bytes.NewReader(body))
	if err != nil {
		return doc, err
	}

	req.Header.Set("Content-Type", ri.MimeType)
	req.Header.Set("X-Taki-Protocol", "v2")
	req.Header.Set("X-Taki-Features", "meta,entities,summary,embedding")
	req.Header.Set("X-Taki-Source-Ref", sourceRef)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		t.logger.Warn().Err(err).Msg("taki v2 request failed, falling back to v1")
		return t.extractTikaV1(ctx, ri, io.NopCloser(bytes.NewReader(body)), doc)
	}
	defer resp.Body.Close()

	var results []takiV2Response
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.logger.Warn().Err(err).Msg("taki v2 response parse failed, falling back to v1")
		return t.extractTikaV1(ctx, ri, io.NopCloser(bytes.NewReader(body)), doc)
	}

	if len(results) == 0 {
		return doc, nil
	}

	r := results[0]

	// Content — always populate (bleve needs it for fulltext)
	doc.Content = strings.TrimSpace(r.Content)

	// Title — prefer taki meta over Tika's title field
	if r.Meta != nil && r.Meta.Title != "" {
		doc.Title = r.Meta.Title
	}

	// Build TakiExtraction with routing + enrichments
	taki := &TakiExtraction{
		Method:  r.Method,
		Summary: r.Summary,
	}

	if r.Entities != nil {
		taki.Entities = r.Entities
	}

	if r.Embed != nil {
		taki.Embed = r.Embed
	}

	if r.Routing != nil {
		taki.Routing = &Routing{
			ContentTarget: r.Routing.ContentTarget,
			MetaTarget:    r.Routing.MetaTarget,
			VectorTarget:  r.Routing.VectorTarget,
		}
	}

	doc.Taki = taki

	if t.CleanStopWords && r.Meta != nil && r.Meta.Language != "" {
		doc.Content = CleanString(doc.Content, r.Meta.Language)
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
