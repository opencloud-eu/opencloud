// Package qdrant provides a lightweight client for the Qdrant vector database REST API.
// Used by the search service to store document embeddings from open_taki v2.
package qdrant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Client is a minimal Qdrant REST API client.
type Client struct {
	url        string
	collection string
	httpClient *http.Client
	logger     log.Logger
}

// Point represents a vector point to upsert into Qdrant.
type Point struct {
	ID      string                 `json:"id"`
	Vector  []float64              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

type upsertRequest struct {
	Points []Point `json:"points"`
}

type collectionConfig struct {
	Vectors struct {
		Size     int    `json:"size"`
		Distance string `json:"distance"`
	} `json:"vectors"`
}

// New creates a new Qdrant client.
func New(url, collection string, logger log.Logger) *Client {
	return &Client{
		url:        url,
		collection: collection,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

// EnsureCollection creates the collection if it doesn't exist.
func (c *Client) EnsureCollection(vectorSize int) error {
	// Check if collection exists
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/collections/%s", c.url, c.collection))
	if err != nil {
		return fmt.Errorf("qdrant: checking collection: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil // exists
	}

	// Create collection
	cfg := collectionConfig{}
	cfg.Vectors.Size = vectorSize
	cfg.Vectors.Distance = "Cosine"

	body, _ := json.Marshal(cfg)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/collections/%s", c.url, c.collection), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant: creating collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant: create collection failed (%d): %s", resp.StatusCode, string(respBody))
	}

	c.logger.Info().Str("collection", c.collection).Int("vector_size", vectorSize).Msg("qdrant: collection created")
	return nil
}

// Upsert inserts or updates points in the collection.
func (c *Client) Upsert(points []Point) error {
	if len(points) == 0 {
		return nil
	}

	body, _ := json.Marshal(upsertRequest{Points: points})
	req, _ := http.NewRequest("PUT",
		fmt.Sprintf("%s/collections/%s/points", c.url, c.collection),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant: upsert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant: upsert failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Search performs a vector similarity search.
func (c *Client) Search(vector []float64, limit int) ([]SearchResult, error) {
	payload := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/collections/%s/points/search", c.url, c.collection),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant: search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Result []SearchResult `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Result, nil
}

// SearchResult represents a single search hit.
type SearchResult struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}
