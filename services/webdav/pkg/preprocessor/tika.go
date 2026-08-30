package preprocessor

import (
	"bytes"
	"encoding/base64"
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

// tikaThumbnail returns the thumbnail Tika picks (/unpack/thumbnail, Tika >= 4.1) and its content type.
func tikaThumbnail(tikaURL, filename, contentType string, data []byte) (string, []byte, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req, err := http.NewRequest(http.MethodPut, strings.TrimRight(tikaURL, "/")+"/unpack/thumbnail?renderThumbnails=true", bytes.NewReader(data))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
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
		return "", nil, fmt.Errorf("tika thumbnail returned %s", resp.Status)
	}

	var body struct {
		Metadata map[string]any `json:"metadata"`
		Image    string         `json:"image"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxTikaResponse)).Decode(&body); err != nil {
		return "", nil, fmt.Errorf("tika thumbnail response: %w", err)
	}
	img, err := base64.StdEncoding.DecodeString(body.Image)
	if err != nil || len(img) == 0 {
		return "", nil, fmt.Errorf("tika thumbnail response: no image")
	}
	return metadataString(body.Metadata, "Content-Type"), img, nil
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
