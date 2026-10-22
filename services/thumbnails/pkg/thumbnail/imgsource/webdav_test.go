package imgsource

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config"
	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
	"github.com/opencloud-eu/reva/v2/pkg/bytesize"
)

func gzipHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		_, _ = io.WriteString(gz, body)
	}
}

func TestWebDavGet(t *testing.T) {
	const body = "not really an image, but bytes all the same"

	t.Run("gzipped response without Content-Length is downloaded", func(t *testing.T) {
		srv := httptest.NewServer(gzipHandler(body))
		defer srv.Close()

		r, err := NewWebDavSource(config.Thumbnail{}, bytesize.MB).Get(context.Background(), srv.URL)
		require.NoError(t, err)
		defer r.Close()

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, body, string(got))
	})

	t.Run("Content-Length above the limit is rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, strings.Repeat("x", 32))
		}))
		defer srv.Close()

		_, err := NewWebDavSource(config.Thumbnail{}, bytesize.ByteSize(16)).Get(context.Background(), srv.URL)
		assert.ErrorIs(t, err, thumbnailerErrors.ErrImageTooLarge)
	})

	t.Run("length-less response above the limit fails instead of truncating", func(t *testing.T) {
		srv := httptest.NewServer(gzipHandler(strings.Repeat("x", 32)))
		defer srv.Close()

		r, err := NewWebDavSource(config.Thumbnail{}, bytesize.ByteSize(16)).Get(context.Background(), srv.URL)
		require.NoError(t, err)
		defer r.Close()

		_, err = io.ReadAll(r)
		assert.Error(t, err)
	})
}
