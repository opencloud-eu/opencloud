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
	Title     string     `json:"Title"`
	Name      string     `json:"Name"`
	Content   string     `json:"Content"`
	Size      uint64     `json:"Size"`
	Mtime     *time.Time `json:"Mtime,omitempty"`
	MimeType  string     `json:"MimeType"`
	Tags      []string   `json:"Tags"`
	Favorites []string   `json:"Favorites"`
	Preview   *Preview   `json:"preview,omitempty"`

	Audio       *libregraph.Audio          `json:"audio,omitempty"`
	Image       *libregraph.Image          `json:"image,omitempty"`
	Location    *libregraph.GeoCoordinates `json:"location,omitempty"`
	Photo       *libregraph.Photo          `json:"photo,omitempty"`
	Video       *libregraph.Video          `json:"video,omitempty"`
	MotionPhoto *libregraph.MotionPhoto    `json:"motionPhoto,omitempty"`
	LivePhoto   *libregraph.LivePhoto      `json:"livePhoto,omitempty"`
}

// Preview holds the dimensions of an embedded preview (e.g. audio cover art).
// Internal signal, not a Graph facet; its presence marks that a preview exists.
type Preview struct {
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

// ToMap flows Preview through the shared facet flattening, under oc.preview.
func (p Preview) ToMap() (map[string]interface{}, error) {
	return map[string]interface{}{
		"width":  p.Width,
		"height": p.Height,
	}, nil
}

func CleanString(content, langCode string) string {
	return strings.TrimSpace(stopwords.CleanString(content, langCode, true))
}
