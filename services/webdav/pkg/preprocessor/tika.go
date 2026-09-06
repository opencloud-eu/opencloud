package preprocessor

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// tikaThumbnail returns the thumbnail Tika found in the document and its content
// type. Tika 4.1 marks it as a THUMBNAIL embedded document, so /unpack/all is
// enough: no preset, no endpoint of our own.
func tikaThumbnail(tikaURL, filename, contentType string, data []byte) (string, []byte, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req, err := http.NewRequest(http.MethodPut, strings.TrimRight(tikaURL, "/")+"/unpack/all", bytes.NewReader(data))
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
	return thumbnailFromUnpack(zr)
}

// vectorThumbnailTypes are thumbnail types nothing here can decode. Office
// documents carry their preview as a metafile, so the THUMBNAIL entry is a
// vector and the usable image is its rendering.
var vectorThumbnailTypes = map[string]struct{}{
	"image/emf":     {},
	"image/wmf":     {},
	"image/x-emf":   {},
	"image/x-wmf":   {},
	"image/svg+xml": {},
}

// unpacked is one entry of an /unpack/all response: the bytes and the metadata
// are separate zip entries, "1.jpg" alongside "1.jpg.metadata.json".
type unpacked struct {
	name     string
	kind     string
	mimeType string
	idPath   string
}

// thumbnailFromUnpack picks the image to use out of an unpack response. Tika
// marks the document's preview as THUMBNAIL. For office documents that preview
// is a metafile, and the raster we want is the RENDERING nested under it, which
// tika only emits when its emf/wmf parser has renderImage turned on.
func thumbnailFromUnpack(zr *zip.Reader) (string, []byte, error) {
	var entries []unpacked
	for _, f := range zr.File {
		if !strings.HasSuffix(f.Name, ".metadata.json") {
			continue
		}
		meta, err := readZipJSON(f)
		if err != nil {
			continue
		}
		entries = append(entries, unpacked{
			name:     strings.TrimSuffix(f.Name, ".metadata.json"),
			kind:     metadataString(meta, "tk:embedded-resource-type"),
			mimeType: metadataString(meta, "Content-Type"),
			idPath:   metadataString(meta, "tk:embedded-id-path"),
		})
	}

	for _, e := range entries {
		if e.kind != "THUMBNAIL" {
			continue
		}
		if _, vector := vectorThumbnailTypes[e.mimeType]; !vector {
			return readThumbnail(zr, e)
		}
		if r, ok := renderingOf(entries, e); ok {
			return readThumbnail(zr, r)
		}
		return "", nil, ErrNoThumbnail
	}

	// a document without a preview of its own can still have been rendered,
	// a pdf page for instance
	for _, e := range entries {
		if e.kind == "RENDERING" {
			return readThumbnail(zr, e)
		}
	}
	return "", nil, ErrNoThumbnail
}

// renderingOf finds the rendering tika nested under a thumbnail. The rendering
// is a child of the image it renders, so its id path extends the thumbnail's.
func renderingOf(entries []unpacked, thumbnail unpacked) (unpacked, bool) {
	for _, e := range entries {
		if e.kind == "RENDERING" && thumbnail.idPath != "" && strings.HasPrefix(e.idPath, thumbnail.idPath+"/") {
			return e, true
		}
	}
	return unpacked{}, false
}

func readThumbnail(zr *zip.Reader, e unpacked) (string, []byte, error) {
	img, err := readZipEntry(zr, e.name)
	if err != nil || len(img) == 0 {
		return "", nil, ErrNoThumbnail
	}
	return e.mimeType, img, nil
}

// readZipJSON decodes a metadata entry. Tika writes either an object or a
// single element list.
func readZipJSON(f *zip.File) (map[string]any, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var raw json.RawMessage
	if err := json.NewDecoder(io.LimitReader(rc, maxTikaResponse)).Decode(&raw); err != nil {
		return nil, err
	}
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err == nil {
		return meta, nil
	}
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil || len(list) == 0 {
		return nil, fmt.Errorf("unexpected metadata shape")
	}
	return list[0], nil
}

func readZipEntry(zr *zip.Reader, name string) ([]byte, error) {
	rc, err := zr.Open(name)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, maxTikaResponse))
}

// metadataString reads a key; values are strings or lists.
func metadataString(meta map[string]any, key string) string {
	switch v := meta[key].(type) {
	case string:
		return v
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	}
	return ""
}
