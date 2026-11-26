// Package clip talks to a CLIP inference service (immich machine-learning API).
// Images and texts are embedded into the same vector space, so a text query
// vector can rank image vectors by cosine similarity.
package clip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// DefaultModel is a multilingual CLIP model: German (and other non-English)
// queries match image content without a translation step.
const DefaultModel = "XLM-Roberta-Large-Vit-B-32"

// Client is a minimal client for the immich machine-learning /predict API.
type Client struct {
	url   string
	model string
	hc    *http.Client
}

// NewClient creates a Client for the inference service at url. A zero timeout
// falls back to 30 seconds, an empty model to DefaultModel.
func NewClient(url, model string, timeout time.Duration) *Client {
	if model == "" {
		model = DefaultModel
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		url:   url,
		model: model,
		hc:    &http.Client{Timeout: timeout},
	}
}

// VectorizeText embeds a text query.
func (c *Client) VectorizeText(ctx context.Context, text string) ([]float32, error) {
	return c.predict(ctx, "textual", func(w *multipart.Writer) error {
		return w.WriteField("text", text)
	})
}

// VectorizeImage embeds an image from r.
func (c *Client) VectorizeImage(ctx context.Context, r io.Reader) ([]float32, error) {
	return c.predict(ctx, "visual", func(w *multipart.Writer) error {
		fw, err := w.CreateFormFile("image", "image")
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, r)
		return err
	})
}

// predict posts a multipart request to /predict. The entries field declares
// which model runs; the response carries the embedding as a JSON-encoded array
// inside a JSON string field.
func (c *Client) predict(ctx context.Context, task string, addPayload func(*multipart.Writer) error) ([]float32, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	entries, err := json.Marshal(map[string]any{
		"clip": map[string]any{task: map[string]any{"modelName": c.model}},
	})
	if err != nil {
		return nil, err
	}
	if err := w.WriteField("entries", string(entries)); err != nil {
		return nil, err
	}
	if err := addPayload(w); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/predict", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return nil, fmt.Errorf("clip inference failed: %s: %s", res.Status, msg)
	}

	var payload struct {
		Clip string `json:"clip"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode clip inference response: %w", err)
	}
	var vector []float32
	if err := json.Unmarshal([]byte(payload.Clip), &vector); err != nil {
		return nil, fmt.Errorf("failed to decode clip embedding: %w", err)
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("clip inference returned an empty embedding")
	}
	return vector, nil
}
