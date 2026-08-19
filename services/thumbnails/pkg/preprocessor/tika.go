package preprocessor

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// maxPreviewLength bounds a served preview; a var so tests can lower it.
var maxPreviewLength int64 = 100 * 1024 * 1024

// maxJPEGSegments bounds the header segments walked before the SOF marker.
const maxJPEGSegments = 128

// TikaDecoder extracts a file's embedded preview image via a Tika server.
type TikaDecoder struct {
	tikaURL string
	// filename lets Tika route by extension; content alone sniffs raws as image/tiff.
	filename string
	// mimeType picks the strategy: audio/* takes the front cover, else the largest preview.
	mimeType string
}

func (d TikaDecoder) Convert(ctx context.Context, r io.Reader) (any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	pick := tikaLargestPreview
	if strings.HasPrefix(d.mimeType, "audio/") {
		pick = tikaFrontCover
	}
	preview, err := pick(ctx, d.tikaURL, d.filename, data)
	if err != nil {
		return nil, err
	}
	return ForType("image/jpeg", nil).Convert(ctx, bytes.NewReader(preview))
}

var tikaHTTPClient = &http.Client{Timeout: 30 * time.Second}

// tikaUnpack PUTs the file to a Tika unpack endpoint and returns the response zip.
func tikaUnpack(ctx context.Context, tikaURL, endpoint, filename string, data []byte) (*zip.Reader, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimRight(tikaURL, "/")+endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "application/zip")
	if filename != "" {
		// the extension routes Tika to its raw parser; quote it so spaces survive
		req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	}

	resp, err := tikaHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tika unpack request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent { // no embedded resources
		return nil, thumbnailerErrors.ErrNoEmbeddedImage
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
	return zr, nil
}

// tikaLargestPreview returns the largest renderable embedded JPEG.
func tikaLargestPreview(ctx context.Context, tikaURL, filename string, data []byte) ([]byte, error) {
	zr, err := tikaUnpack(ctx, tikaURL, "/unpack", filename, data)
	if err != nil {
		return nil, err
	}
	var best []byte
	for _, f := range zr.File {
		if f.UncompressedSize64 == 0 || f.UncompressedSize64 > uint64(maxPreviewLength) || int(f.UncompressedSize64) <= len(best) {
			continue
		}
		if jpg := readZipEntry(f); isRenderableJPEG(jpg) {
			best = jpg
		}
	}
	if best == nil {
		return nil, thumbnailerErrors.ErrNoEmbeddedImage
	}
	return best, nil
}

// tikaFrontCover returns the tagged front cover (ID3 APIC "Cover (front)") among the
// embedded images, else the first renderable one. Reads /unpack/all for the sidecars.
func tikaFrontCover(ctx context.Context, tikaURL, filename string, data []byte) ([]byte, error) {
	zr, err := tikaUnpack(ctx, tikaURL, "/unpack/all", filename, data)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		byName[f.Name] = f
	}
	var first []byte
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ".metadata.json") {
			continue
		}
		sidecar, ok := byName[f.Name+".metadata.json"]
		if !ok {
			continue
		}
		meta := parseEmbeddedMeta(readZipEntry(sidecar))
		if !strings.HasPrefix(meta.ContentType, "image/") {
			continue // skip the audio stream and other non-image parts
		}
		if f.UncompressedSize64 == 0 || f.UncompressedSize64 > uint64(maxPreviewLength) {
			continue
		}
		img := readZipEntry(f)
		if !isRenderableImage(img) {
			continue
		}
		if first == nil {
			first = img
		}
		if meta.Description == "Cover (front)" {
			return img, nil
		}
	}
	if first == nil {
		return nil, thumbnailerErrors.ErrNoEmbeddedImage
	}
	return first, nil
}

// tikaEmbeddedMeta is the subset of a Tika /unpack/all sidecar we read.
type tikaEmbeddedMeta struct {
	ContentType string `json:"Content-Type"`
	Description string `json:"dc:description"`
}

func parseEmbeddedMeta(b []byte) tikaEmbeddedMeta {
	var m tikaEmbeddedMeta
	_ = json.Unmarshal(b, &m)
	return m
}

// isRenderableImage accepts the formats audio cover art uses.
func isRenderableImage(buf []byte) bool {
	return isRenderableJPEG(buf) || isPNG(buf)
}

func isPNG(buf []byte) bool {
	return len(buf) >= 8 && bytes.Equal(buf[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
}

func readZipEntry(f *zip.File) []byte {
	rc, err := f.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()
	b, err := io.ReadAll(io.LimitReader(rc, maxPreviewLength))
	if err != nil {
		return nil
	}
	return b
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
