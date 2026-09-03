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

// Document wraps all resource meta fields,
// it is used as a content extraction result.
type Document struct {
	Title       string                     `json:"Title"`
	Name        string                     `json:"Name"`
	Content     string                     `json:"Content"`
	Size        uint64                     `json:"Size"`
	Mtime       *time.Time                 `json:"Mtime,omitempty"`
	MimeType    string                     `json:"MimeType"`
	Tags        []string                   `json:"Tags"`
	Favorites   []string                   `json:"Favorites"`
	Audio       *libregraph.Audio          `json:"audio,omitempty"`
	Image       *libregraph.Image          `json:"image,omitempty"`
	Location    *libregraph.GeoCoordinates `json:"location,omitempty"`
	Photo       *libregraph.Photo          `json:"photo,omitempty"`
	Video       *libregraph.Video          `json:"video,omitempty"`
	MotionPhoto *libregraph.MotionPhoto    `json:"motionPhoto,omitempty"`
}

func CleanString(content, langCode string) string {
	return strings.TrimSpace(stopwords.CleanString(content, langCode, true))
}
