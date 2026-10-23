package svc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalPNG is a valid 1×1 PNG used for testing image downloads.
var minimalPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // signature
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
	0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
	0x00, 0x00, 0x03, 0x00, 0x01, 0x5B, 0x42, 0xD3,
	0xBF, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
	0x44, 0xAE, 0x42, 0x60, 0x82,
}

// testPhotoService creates a UsersUserProfilePhotoService configured for
// profile picture sync with the given allowlist and issuer.
func testPhotoService(httpClient HTTPClient, urlAllowlist []string, oidcIssuer string) UsersUserProfilePhotoService {
	return UsersUserProfilePhotoService{
		storage:      nil, // not needed for sync tests
		httpClient:   httpClient,
		urlAllowlist: urlAllowlist,
		oidcIssuer:   oidcIssuer,
	}
}

func TestURLPatternMatches(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		target  string
		want    bool
	}{
		// Host glob patterns
		{"glob subdomain", "https://*.example.com", "https://cdn.example.com", true},
		{"glob exact host", "https://example.com", "https://example.com", true},
		{"glob no match host", "https://*.example.com", "https://evil.com", false},
		{"glob all", "*", "https://anything.com/path", true},
		{"glob all http", "*", "http://anything.com", true},

		// Scheme matching
		{"scheme mismatch", "https://example.com", "http://example.com", false},
		{"scheme match http", "http://example.com", "http://example.com", true},

		// Path matching
		{"path prefix wildcard", "https://example.com/img/*", "https://example.com/img/avatar.png", true},
		{"path exact", "https://example.com/avatar.png", "https://example.com/avatar.png", true},
		{"path no match", "https://example.com/avatar.png", "https://example.com/other.png", false},
		{"root path matches any path", "https://example.com", "https://example.com/any/path", true},
		{"root path slash matches any path", "https://example.com/", "https://example.com/any/path", true},

		// Only complete HTTP(S) URL patterns and the explicit wildcard are accepted.
		{"bare host rejected", "example.com", "https://example.com/avatar.png", false},
		{"bare host glob rejected", "*.example.com", "https://cdn.example.com/avatar.png", false},
		{"relative URL rejected", "//example.com", "https://example.com/avatar.png", false},
		{"invalid URL pattern", "https://example.com/%zz", "https://example.com/avatar.png", false},
		{"invalid host glob", "https://[*.example.com", "https://example.com/avatar.png", false},
		{"unsupported pattern scheme", "ftp://example.com", "https://example.com/avatar.png", false},
		{"unsupported target scheme", "*", "ftp://example.com/avatar.png", false},
		{"relative target rejected", "*", "/avatar.png", false},
		{"target credentials rejected", "*", "https://user:password@example.com/avatar.png", false},
		{"pattern credentials rejected", "https://user@example.com", "https://example.com/avatar.png", false},
		{"query in pattern rejected", "https://example.com?token=secret", "https://example.com/avatar.png", false},
		{"empty query in pattern rejected", "https://example.com?", "https://example.com/avatar.png", false},
		{"fragment in pattern rejected", "https://example.com#avatar", "https://example.com/avatar.png", false},
		{"empty fragment in pattern rejected", "https://example.com#", "https://example.com/avatar.png", false},
		{"target query ignored", "https://example.com/avatar.png", "https://example.com/avatar.png?size=128#photo", true},

		// Ports are part of the host pattern; paths retain their literal encoding.
		{"port match", "https://example.com:8443", "https://example.com:8443/avatar.png", true},
		{"port mismatch", "https://example.com:8443", "https://example.com/avatar.png", false},
		{"explicit default port differs", "https://example.com", "https://example.com:443/avatar.png", false},
		{"path prefix boundary", "https://example.com/img/*", "https://example.com/images/avatar.png", false},
		{"path case sensitive", "https://example.com/Avatar.png", "https://example.com/avatar.png", false},
		{"path not normalized", "https://example.com/avatar.png", "https://example.com/other/../avatar.png", false},
		{"encoded separator differs", "https://example.com/img/*", "https://example.com/img%2Favatar.png", false},
		{"encoded path exact", "https://example.com/my%20avatar.png", "https://example.com/my%20avatar.png", true},

		// IPv6 literals must not be interpreted as glob character classes.
		{"IPv6 literal", "https://[::1]", "https://[::1]/avatar.png", true},
		{"IPv6 port match", "https://[::1]:9200", "https://[::1]:9200/avatar.png", true},
		{"IPv6 port mismatch", "https://[::1]:9200", "https://[::1]:9201/avatar.png", false},
		{"IPv6 path match", "https://[2001:db8::1]:9200/img/*", "https://[2001:db8::1]:9200/img/avatar.png", true},
		{"IPv6 path mismatch", "https://[2001:db8::1]:9200/img/*", "https://[2001:db8::1]:9200/other/avatar.png", false},
		{"IPv6 host mismatch", "https://[::1]", "https://[::2]/avatar.png", false},
		{"IPv6 brackets do not match DNS host", "https://[::1]", "https://1/avatar.png", false},
		{"IPv6 case insensitive", "https://[2001:DB8::1]", "https://[2001:db8::1]/avatar.png", true},

		// Case insensitivity
		{"case insensitive host", "https://*.Example.COM", "https://cdn.example.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(tt.target)
			require.NoError(t, err)
			got := urlPatternMatches(tt.pattern, parsed)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestURLPatternMatchesNilTarget(t *testing.T) {
	assert.False(t, urlPatternMatches("*", nil))
}

func TestIsProfilePictureURLAllowed(t *testing.T) {
	tests := []struct {
		name       string
		allowlist  []string
		oidcIssuer string
		targetURL  string
		want       bool
	}{
		{
			name:       "default to issuer host",
			allowlist:  nil,
			oidcIssuer: "https://idp.example.com",
			targetURL:  "https://idp.example.com/avatar.png",
			want:       true,
		},
		{
			name:       "default to issuer host rejects other host",
			allowlist:  nil,
			oidcIssuer: "https://idp.example.com",
			targetURL:  "https://evil.com/avatar.png",
			want:       false,
		},
		{
			name:       "explicit allowlist with glob",
			allowlist:  []string{"https://*.cdn.example.com"},
			oidcIssuer: "https://idp.example.com",
			targetURL:  "https://img.cdn.example.com/avatar.png",
			want:       true,
		},
		{
			name:       "wildcard allows all",
			allowlist:  []string{"*"},
			oidcIssuer: "",
			targetURL:  "https://any.host.com/path",
			want:       true,
		},
		{
			name:       "rejects non-http scheme",
			allowlist:  []string{"*"},
			oidcIssuer: "",
			targetURL:  "ftp://example.com/file",
			want:       false,
		},
		{
			name:       "rejects invalid url",
			allowlist:  []string{"*"},
			oidcIssuer: "",
			targetURL:  "://not-a-url",
			want:       false,
		},
		{
			name:      "wildcard rejects credentials",
			allowlist: []string{"*"},
			targetURL: "https://user:password@example.com/avatar.png",
			want:      false,
		},
		{
			name:       "issuer path does not restrict photo paths",
			oidcIssuer: "https://idp.example.com/realms/opencloud",
			targetURL:  "https://idp.example.com/avatar.png",
			want:       true,
		},
		{
			name:       "issuer scheme is enforced",
			oidcIssuer: "https://idp.example.com",
			targetURL:  "http://idp.example.com/avatar.png",
			want:       false,
		},
		{
			name:       "no issuer and no allowlist rejects",
			allowlist:  nil,
			oidcIssuer: "",
			targetURL:  "https://example.com/avatar.png",
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testPhotoService(http.DefaultClient, tt.allowlist, tt.oidcIssuer)
			got := s.isProfilePictureURLAllowed(tt.targetURL)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSyncPhotoFromURL(t *testing.T) {
	t.Run("rejects disallowed URL", func(t *testing.T) {
		s := testPhotoService(http.DefaultClient, nil, "https://idp.example.com")
		err := s.SyncPhotoFromURL(context.Background(), "user123", "https://evil.com/img.png")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("returns error for empty user ID", func(t *testing.T) {
		s := testPhotoService(http.DefaultClient, []string{"*"}, "")
		err := s.SyncPhotoFromURL(context.Background(), "", "https://example.com/img.png")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required argument is missing")
	})

	t.Run("rejects non-image content type", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html>not an image</html>"))
		}))
		defer srv.Close()

		s := testPhotoService(srv.Client(), []string{"*"}, "")
		err := s.SyncPhotoFromURL(context.Background(), "user123", srv.URL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content type")
	})

	t.Run("rejects empty response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte{})
		}))
		defer srv.Close()

		s := testPhotoService(srv.Client(), []string{"*"}, "")
		err := s.SyncPhotoFromURL(context.Background(), "user123", srv.URL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("rejects error status code", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer srv.Close()

		s := testPhotoService(srv.Client(), []string{"*"}, "")
		err := s.SyncPhotoFromURL(context.Background(), "user123", srv.URL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

func TestFetchProfilePictureSizeLimit(t *testing.T) {
	oversized := bytes.Repeat(minimalPNG, (maxProfilePhotoBytes/len(minimalPNG))+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(oversized)
	}))
	defer srv.Close()

	s := testPhotoService(srv.Client(), []string{"*"}, "")
	_, err := s.fetchProfilePicture(context.Background(), srv.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

// fakePhotoService is a test double for UsersUserProfilePhotoProvider.
type fakePhotoService struct {
	updatePhotoFn func(ctx context.Context, id string, r io.Reader) error
}

func (f *fakePhotoService) GetPhoto(_ context.Context, _ string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (f *fakePhotoService) UpdatePhoto(ctx context.Context, id string, r io.Reader) error {
	if f.updatePhotoFn != nil {
		return f.updatePhotoFn(ctx, id, r)
	}
	return nil
}

func (f *fakePhotoService) DeletePhoto(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

func (f *fakePhotoService) SyncPhotoFromURL(_ context.Context, _, _ string) error {
	return errors.New("not implemented")
}
