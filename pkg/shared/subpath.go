package shared

import (
	"net/url"
	"path"
	"strings"
)

// SubPath returns the path component of rawURL, cleaned and without a
// trailing slash ("" if the URL has no path, or is just "/"). It is used to
// derive the various per-service root/prefix defaults from OC_URL, so that a
// single env var is enough to deploy OpenCloud under a subpath.
func SubPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	p := strings.TrimRight(u.Path, "/")
	if p == "" {
		return ""
	}

	return path.Clean(p)
}
