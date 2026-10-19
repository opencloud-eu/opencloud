package content

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/go-tika/tika"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

// rmetaThumbnailPath asks for the recursive metadata with Tika's thumbnail
// preset, so Tika names the image a client would show and renders one where
// the format needs that, a pdf page for one.
const rmetaThumbnailPath = "/rmeta/preset/thumbnail/text"

// Tika is used to extract content from a resource,
// it uses apache tika to retrieve all the data.
type Tika struct {
	*Basic
	Retriever
	tika                       *tika.Client
	tikaURL                    string
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
		tikaURL:                    cfg.Extractor.Tika.TikaURL,
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

	metas, err := t.metaRecursive(ctx, ri.Id)
	if err != nil {
		return doc, err
	}
	if len(metas) == 0 {
		return doc, nil
	}

	for _, meta := range metas {
		title, err := getFirstValue(meta, "dc:title")
		if err != nil {
			title, err = getFirstValue(meta, "title")
		}
		if err == nil {
			doc.Title = strings.TrimSpace(fmt.Sprintf("%s %s", doc.Title, title))
		}

		if content, err := getFirstValue(meta,
			"tk:content",     // tika 4
			"X-TIKA:content", // tika 3 legacy
		); err == nil {
			doc.Content = strings.TrimSpace(fmt.Sprintf("%s %s", doc.Content, content))
		}
	}

	// facets describe the file itself, not its embedded parts (cover art, clips)
	m0 := metas[0]
	doc.Location = t.getLocation(m0)
	doc.Image = t.getImage(m0)
	doc.Photo = t.getPhoto(m0)
	doc.Audio = t.getAudio(m0)
	doc.LivePhoto = t.getLivePhoto(m0)
	doc.Video = t.getVideo(m0)

	// a motion photo is the file's own xmp plus the video tika extracted from
	// it; the xmp alone proves nothing, a share can strip the appended clip
	if i := slices.IndexFunc(metas[1:], isVideo); i >= 0 {
		doc.MotionPhoto = t.getMotionPhoto(m0, metas[i+1])
	}

	doc.Preview = getPreview(metas)

	if langCode := t.detectLanguage(ctx, doc.Content); langCode != "" && t.CleanStopWords {
		doc.Content = CleanString(doc.Content, langCode)
	}

	return doc, nil
}

// metaRecursive is Tika's recursive metadata, asked for with the thumbnail
// preset so the index learns about the same image the thumbnailer serves.
func (t Tika) metaRecursive(ctx context.Context, id *provider.ResourceId) ([]map[string][]string, error) {
	data, err := t.Retrieve(ctx, id)
	if err != nil {
		return nil, err
	}
	defer data.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, t.tikaURL+rmetaThumbnailPath, data)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, fmt.Errorf("tika serves no %s, its configuration has to activate the thumbnail preset", rmetaThumbnailPath)
	default:
		return nil, fmt.Errorf("tika %s: %s", rmetaThumbnailPath, res.Status)
	}

	return decodeMetas(res.Body)
}

// decodeMetas reads Tika's per-part metadata, whose values are a string or a
// list of them.
func decodeMetas(r io.Reader) ([]map[string][]string, error) {
	var parts []map[string]any
	if err := json.NewDecoder(r).Decode(&parts); err != nil {
		return nil, err
	}

	metas := make([]map[string][]string, 0, len(parts))
	for _, part := range parts {
		meta := make(map[string][]string, len(part))
		for key, value := range part {
			switch typed := value.(type) {
			case string:
				meta[key] = []string{typed}
			case []any:
				for _, entry := range typed {
					s, ok := entry.(string)
					if !ok {
						return nil, fmt.Errorf("field %q holds %v of type %T, want a string", key, entry, entry)
					}
					meta[key] = append(meta[key], s)
				}
			default:
				return nil, fmt.Errorf("field %q holds %v of type %v, want a string or a list of them", key, value, reflect.TypeOf(value))
			}
		}
		metas = append(metas, meta)
	}

	return metas, nil
}

// detectLanguage asks tika for the language of content. Tika 4 moved the
// endpoint from /language/string to /language, so try the new path first and
// fall back for an older tika.
func (t Tika) detectLanguage(ctx context.Context, content string) string {
	for _, path := range []string{"/language", "/language/string"} {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, t.tikaURL+path, strings.NewReader(content))
		if err != nil {
			return ""
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return ""
		}
		lang, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err == nil && res.StatusCode == http.StatusOK && len(lang) > 0 {
			return string(lang)
		}
	}
	return ""
}
