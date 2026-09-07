package content

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/go-tika/tika"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

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
		title, err := getFirstValue(meta, "dc:title")
		if err != nil {
			title, err = getFirstValue(meta, "title")
		}
		if err == nil {
			doc.Title = strings.TrimSpace(fmt.Sprintf("%s %s", doc.Title, title))
		}

		// tika 4 renamed the meta prefix from X-TIKA: to tk:
		if content, err := getFirstValue(meta, "tk:content"); err == nil {
			doc.Content = strings.TrimSpace(fmt.Sprintf("%s %s", doc.Content, content))
		} else if content, err := getFirstValue(meta, "X-TIKA:content"); err == nil {
			doc.Content = strings.TrimSpace(fmt.Sprintf("%s %s", doc.Content, content))
		}

		// keep facets from earlier entries, an embedded resource's meta
		// (e.g. cover art) must not reset them
		if v := t.getLocation(meta); v != nil {
			doc.Location = v
		}
		if v := t.getImage(meta); v != nil {
			doc.Image = v
		}
		if v := t.getPhoto(meta); v != nil {
			doc.Photo = v
		}
		if v := t.getAudio(meta); v != nil {
			doc.Audio = v
		}
		if v := t.getLivePhoto(meta); v != nil {
			doc.LivePhoto = v
		}
	}

	if len(metas) > 0 {
		// the video facet says the file is a video, so it comes from the file
		// itself: the clip tika extracts from a motion photo must not make its
		// image look like one
		doc.Video = t.getVideo(metas[0])
	}

	// a motion photo is the xmp on the file itself plus the video tika extracted
	// from it. The xmp alone proves nothing: a share can keep it and strip the
	// appended video.
	if len(metas) > 0 {
		if i := slices.IndexFunc(metas[1:], isVideo); i >= 0 {
			doc.MotionPhoto = t.getMotionPhoto(metas[0], metas[i+1])
		}
	}

	if langCode := t.detectLanguage(ctx, doc.Content); langCode != "" && t.CleanStopWords {
		doc.Content = CleanString(doc.Content, langCode)
	}

	return doc, nil
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
