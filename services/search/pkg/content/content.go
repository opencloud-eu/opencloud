package content

import (
	"strings"

	"github.com/bbalet/stopwords"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func init() {
	stopwords.OverwriteWordSegmenter(`[^ ]+`)
}

// Document wraps all resource meta fields,
// it is used as a content extraction result.
type Document struct {
	Title    string
	Name     string
	Content  string
	Size     uint64
	Mtime    string
	MimeType string
	Tags     []string
	Audio    *libregraph.Audio          `json:"audio,omitempty"`
	Image    *libregraph.Image          `json:"image,omitempty"`
	Location *libregraph.GeoCoordinates `json:"location,omitempty"`
	Photo    *libregraph.Photo          `json:"photo,omitempty"`
}

func CleanString(content, langCode string) string {
	return strings.TrimSpace(stopwords.CleanString(content, langCode, true))
}
