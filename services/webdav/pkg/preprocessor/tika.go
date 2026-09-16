package preprocessor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// maxTikaResponse bounds what is read back from Tika
const maxTikaResponse = 100 * 1024 * 1024

var tikaHTTPClient = &http.Client{Timeout: 60 * time.Second}

// TikaThumbnail asks a Tika server for the document's thumbnail and decodes it.
type TikaThumbnail struct {
	tikaURL string
	// detection hints
	filename    string
	contentType string
}

// Convert reads the file and returns its thumbnail as an image.
func (t TikaThumbnail) Convert(r io.Reader) (any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	contentType, img, err := tikaThumbnail(t.tikaURL, t.filename, t.contentType, data)
	if err != nil {
		return nil, err
	}

	return ForType(contentType, nil).Convert(bytes.NewReader(img))
}

// tikaThumbnail returns the thumbnail Tika found in the document and its
// content type. The thumbnail catalog preset (Tika 4.1, TIKA-4856) unpacks
// exactly the raster image a client would show for the document — a stored
// thumbnail such as a raw photo's embedded preview or an audio file's cover,
// the rendering of an office document's metafile thumbnail, or the rendered
// first page of a PDF — so the zip carries that one image and nothing else.
func tikaThumbnail(tikaURL, filename, contentType string, data []byte) (string, []byte, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req, err := http.NewRequest(http.MethodPut, strings.TrimRight(tikaURL, "/")+"/unpack/preset/thumbnail", bytes.NewReader(data))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/zip")
	// the extension is what routes a raw image to its parser
	if filename != "" {
		req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	}

	resp, err := tikaHTTPClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("tika request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNoContent:
		return "", nil, ErrNoThumbnail
	case http.StatusNotFound:
		// the route exists once the catalog preset is active
		return "", nil, fmt.Errorf(`tika unpack returned %s: the "thumbnail" preset needs Tika >= 4.1 and "presets": {"thumbnail": true} in its config`, resp.Status)
	default:
		return "", nil, fmt.Errorf("tika unpack returned %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTikaResponse))
	if err != nil {
		return "", nil, fmt.Errorf("tika unpack response: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", nil, fmt.Errorf("tika unpack response: %w", err)
	}
	return thumbnailFromZip(zr)
}

// thumbnailFromZip takes the one image the preset unpacked. Lenient about the
// shape: the first regular file that is not a metadata sidecar wins.
func thumbnailFromZip(zr *zip.Reader) (string, []byte, error) {
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || strings.HasSuffix(f.Name, ".metadata.json") {
			continue
		}
		img, err := readZipEntry(zr, f.Name)
		if err != nil || len(img) == 0 {
			return "", nil, ErrNoThumbnail
		}
		return entryContentType(f.Name, img), img, nil
	}
	return "", nil, ErrNoThumbnail
}

// entryContentType names the image's type. The preset zip carries no metadata
// entries, so the extension decides and the bytes break the tie.
func entryContentType(name string, data []byte) string {
	if byExt := mime.TypeByExtension(filepath.Ext(name)); byExt != "" {
		return byExt
	}
	return http.DetectContentType(data)
}

func readZipEntry(zr *zip.Reader, name string) ([]byte, error) {
	rc, err := zr.Open(name)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, maxTikaResponse))
}
