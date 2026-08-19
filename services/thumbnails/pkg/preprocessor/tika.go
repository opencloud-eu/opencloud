package preprocessor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// maxPreviewLength bounds a served preview; a var so tests can lower it.
var maxPreviewLength int64 = 100 * 1024 * 1024

// maxJPEGSegments bounds the header segments walked before the SOF marker;
// generous so a big ICC profile split across many APP2 markers still resolves.
const maxJPEGSegments = 128

// TikaDecoder extracts an embedded preview image from a file via a Tika server.
type TikaDecoder struct {
	tikaURL string
	// filename lets Tika route by extension; content alone sniffs raws as plain image/tiff.
	filename string
}

func (d TikaDecoder) Convert(r io.Reader) (any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	preview, err := tikaExtractPreview(context.Background(), d.tikaURL, d.filename, data)
	if err != nil {
		return nil, err
	}
	return ForType("image/jpeg", nil).Convert(bytes.NewReader(preview))
}

var tikaHTTPClient = &http.Client{Timeout: 30 * time.Second}

// tikaExtractPreview unpacks the file via Tika and returns its largest renderable embedded JPEG.
func tikaExtractPreview(ctx context.Context, tikaURL, filename string, data []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimRight(tikaURL, "/")+"/unpack", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "application/zip")
	if filename != "" {
		// quoted: unquoted values with spaces/specials would truncate the extension Tika routes on
		req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	}

	resp, err := tikaHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tika unpack request failed: %w", err)
	}
	defer resp.Body.Close()
	// Tika answers 204 when the file carries no embedded resources: an expected previewless raw
	if resp.StatusCode == http.StatusNoContent {
		return nil, thumbnailerErrors.ErrNoImageFromRawFile
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tika unpack returned %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPreviewLength))
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("tika unpack response is not a valid zip: %w", err)
	}

	var best []byte
	for _, f := range zr.File {
		if f.UncompressedSize64 == 0 || f.UncompressedSize64 > uint64(maxPreviewLength) || int(f.UncompressedSize64) <= len(best) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		jpg, err := io.ReadAll(io.LimitReader(rc, maxPreviewLength))
		_ = rc.Close()
		if err != nil || !isRenderableJPEG(jpg) {
			continue
		}
		best = jpg
	}
	if best == nil {
		return nil, thumbnailerErrors.ErrNoImageFromRawFile
	}
	return best, nil
}

// isRenderableJPEG accepts only DCT processes common decoders render; DNG raw
// payloads are lossless JPEG (SOF3) behind the same SOI marker.
func isRenderableJPEG(buf []byte) bool {
	if len(buf) < 4 || buf[0] != 0xff || buf[1] != 0xd8 {
		return false
	}
	i := 2
	for segments := 0; i+4 <= len(buf) && segments < maxJPEGSegments; segments++ {
		if buf[i] != 0xff {
			return false
		}
		marker := buf[i+1]
		switch {
		case marker == 0xff: // fill byte
			i++
			continue
		case marker >= 0xd0 && marker <= 0xd7: // RST, no length field
			i += 2
			continue
		}
		switch marker {
		case 0xc0, 0xc1, 0xc2: // baseline, extended sequential, progressive
			return true
		case 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf:
			// lossless, differential and arithmetic processes
			return false
		case 0xd9, 0xda: // EOI or scan start without a SOF
			return false
		}
		segLen := int(buf[i+2])<<8 | int(buf[i+3])
		if segLen < 2 {
			return false
		}
		i += 2 + segLen
	}
	return false
}
