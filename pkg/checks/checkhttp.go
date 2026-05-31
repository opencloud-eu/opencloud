package checks

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/handlers"
)

// NewHTTPCheck checks the reachability of a http server.
func NewHTTPCheck(url string) func(context.Context) error {
	url, err := handlers.FailSaveAddress(url)
	if err != nil {
		return func(context.Context) error {
			return fmt.Errorf("invalid url: %v", err)
		}
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	c := &http.Client{
		Timeout: 3 * time.Second,
	}
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err != nil {
			return fmt.Errorf("could not connect to http server: %v", err)
		}
		_ = resp.Body.Close()
		return nil
	}
}
