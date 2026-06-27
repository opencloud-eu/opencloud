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
	Title     string
	Name      string
	Content   string
	Size      uint64
	Mtime     string
	MimeType  string
	Tags      []string
	Favorites []string
	Audio     *libregraph.Audio          `json:"audio,omitempty"`
	Image     *libregraph.Image          `json:"image,omitempty"`
	Location  *libregraph.GeoCoordinates `json:"location,omitempty"`
	Photo     *libregraph.Photo          `json:"photo,omitempty"`

	// Taki v2 extended fields (populated when extractor is open_taki with v2 protocol)
	Taki *TakiExtraction `json:"taki,omitempty"`
}

// TakiExtraction holds extended extraction results from open_taki v2 protocol.
type TakiExtraction struct {
	Method   string     `json:"method,omitempty"`
	Summary  string     `json:"summary,omitempty"`
	Entities []Entity   `json:"entities,omitempty"`
	Embed    []float64  `json:"embedding,omitempty"`
	Routing  *Routing   `json:"routing,omitempty"`
}

// Entity represents a named entity extracted by the LLM.
type Entity struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Routing tells the search service where to store content.
type Routing struct {
	ContentTarget string `json:"content_target"` // bleve, vector, both, none
	MetaTarget    string `json:"meta_target"`    // always bleve
	VectorTarget  string `json:"vector_target"`  // always vector
}

func CleanString(content, langCode string) string {
	return strings.TrimSpace(stopwords.CleanString(content, langCode, true))
}
