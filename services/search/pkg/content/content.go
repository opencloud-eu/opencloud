package content

import (
	"strings"
	"time"

	"github.com/bbalet/stopwords"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func init() {
	stopwords.OverwriteWordSegmenter(`[^ ]+`)
}

// ImageVectorDims is the dimensionality of the ImageVector field. It is part
// of the index schema (baked into the index mapping on creation), not a
// configuration value: vectors of any other length cannot be indexed, and a
// model of another size requires a new index. The CLIP extractor verifies at
// startup that the configured model produces vectors of this length.
const ImageVectorDims = 512

// Document wraps all resource meta fields,
// it is used as a content extraction result.
type Document struct {
	Title       string                     `json:"Title"`
	Name        string                     `json:"Name"`
	Content     string                     `json:"Content"`
	Size        uint64                     `json:"Size"`
	Mtime       *time.Time                 `json:"Mtime"`
	MimeType    string                     `json:"MimeType"`
	Tags        []string                   `json:"Tags"`
	Favorites   []string                   `json:"Favorites"`
	Audio       *libregraph.Audio          `json:"audio,omitempty"`
	Image       *libregraph.Image          `json:"image,omitempty"`
	Location    *libregraph.GeoCoordinates `json:"location,omitempty"`
	Photo       *libregraph.Photo          `json:"photo,omitempty"`
	ImageVector []float32                  `json:"imageVector,omitempty"`
}

func CleanString(content, langCode string) string {
	return strings.TrimSpace(stopwords.CleanString(content, langCode, true))
}
