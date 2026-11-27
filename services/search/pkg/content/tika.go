package content

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/go-tika/tika"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

// ErrUnparsable marks a file tika answered for but could not parse, unlike an
// unreachable tika it fails the same way on the next attempt more often than not
var ErrUnparsable = errors.New("tika failed to parse")

// tika answers its version without parsing anything, even while busy with a file
var probeTimeout = 5 * time.Second

// rmetaThumbnailPreset asks for the recursive metadata with Tika's thumbnail
// preset, so Tika names the image a client would show and renders one where
// the format needs that, a pdf page for one. The index then learns about the
// same image the thumbnailer serves.
const rmetaThumbnailPreset = "preset/thumbnail/"

// Tika is used to extract content from a resource,
// it uses apache tika to retrieve all the data.
type Tika struct {
	*Basic
	Retriever
	tika                       *tika.Client
	tikaURL                    string
	ContentExtractionSizeLimit uint64
	CleanStopWords             bool
	Timeout                    time.Duration
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
		Timeout:                    cfg.Extractor.Tika.Timeout,
	}, nil
}

// Extract loads a resource from its underlying storage, passes it to tika and processes the result into a Document.
func (t Tika) Extract(ctx context.Context, ri *provider.ResourceInfo) (Document, error) {
	doc, err := t.Basic.Extract(ctx, ri)
	if err != nil {
		return doc, err
	}

	if !t.Remote(ri) {
		return doc, nil
	}
	// the head of a file over the limit still holds its metadata (exif, image
	// size), its content is left out and tika may fail on the cut
	oversized := ri.GetSize() > t.ContentExtractionSizeLimit

	// a tika that hangs instead of refusing connections would block the walk
	attemptCtx := ctx
	if t.Timeout > 0 {
		var cancel context.CancelFunc
		attemptCtx, cancel = context.WithTimeout(ctx, t.Timeout)
		defer cancel()
	}

	data, err := t.Retrieve(attemptCtx, ri.GetId())
	if err != nil {
		return doc, t.attemptError(ctx, attemptCtx, ri, err)
	}
	defer data.Close()

	var input io.Reader = data
	handler := rmetaThumbnailPreset + "text"
	if oversized {
		input = io.LimitReader(data, int64(min(t.ContentExtractionSizeLimit, math.MaxInt64)))
		handler = rmetaThumbnailPreset + "ignore"
	}
	metas, err := t.tika.MetaRecursiveType(attemptCtx, input, handler)
	if isStatus(err, http.StatusNotFound) {
		return doc, fmt.Errorf("tika serves no /rmeta/%s, its configuration has to activate the thumbnail preset: %w", handler, err)
	}
	if err != nil {
		err = t.attemptError(ctx, attemptCtx, ri, unreachable(err))
		if oversized && !errors.Is(err, ErrUnreachable) {
			// a failure on the cut says nothing about the file, a retry would fail the same way
			t.logger.Info().Err(err).Interface("ResourceID", ri.GetId()).Str("Name", ri.GetName()).Msg("no metadata from the head of a file over the content extraction size limit")
			return doc, nil
		}
		return doc, err
	}
	if oversized {
		t.addFacets(&doc, metas)
		doc.Preview = getPreview(metas)
		return doc, nil
	}
	// a file with content always yields metadata, none means tika failed on it
	if len(metas) == 0 {
		return doc, fmt.Errorf("%w %q: no metadata", ErrUnparsable, ri.GetName())
	}
	// tika answers a failed parse (ocr crash, broken pdf) with 200 and empty
	// content, only this key tells it apart from a file without text
	cause, err := getFirstValue(metas[0],
		"tk:exception:container-exception",     // tika 4
		"X-TIKA:EXCEPTION:container_exception", // tika 3 legacy
	)
	if err == nil {
		// the value is a java stack trace, its first line names the cause
		cause, _, _ = strings.Cut(cause, "\n")
		return doc, fmt.Errorf("%w %q: %s", ErrUnparsable, ri.GetName(), cause)
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

	t.addFacets(&doc, metas)
	doc.Preview = getPreview(metas)

	if langCode := t.detectLanguage(attemptCtx, doc.Content); langCode != "" && t.CleanStopWords {
		doc.Content = CleanString(doc.Content, langCode)
	}

	return doc, nil
}

// Remote tells whether extracting ri goes to tika, only a file with content does
func (t Tika) Remote(ri *provider.ResourceInfo) bool {
	return ri.GetType() == provider.ResourceType_RESOURCE_TYPE_FILE && ri.GetSize() > 0
}

// Probe asks tika for its version, which tells a tika that is down or hangs
// from one busy with a hard file
func (t Tika) Probe(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	_, err := t.tika.Version(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	return nil
}

// attemptError turns an attempt that ran out of time into a failure of the file:
// tika accepted the request, so a retry would most likely run as long again and
// stack more work on a tika that may still be parsing. Unless tika does not even
// answer its version, then it hangs and is unreachable.
func (t Tika) attemptError(ctx, attemptCtx context.Context, ri *provider.ResourceInfo, err error) error {
	if ctx.Err() != nil || !errors.Is(attemptCtx.Err(), context.DeadlineExceeded) {
		return err
	}
	err = fmt.Errorf("extracting %q took longer than %s: %w", ri.GetName(), t.Timeout, context.DeadlineExceeded)
	probeErr := t.Probe(ctx)
	if probeErr != nil {
		return fmt.Errorf("%w, %w", probeErr, err)
	}
	return err
}

// addFacets takes the facets of the file itself, not of its embedded parts
// (cover art, clips)
func (t Tika) addFacets(doc *Document, metas []map[string][]string) {
	if len(metas) == 0 {
		return
	}
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
