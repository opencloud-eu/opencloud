package osu

import (
	"bytes"
	"encoding/json"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

type QueryReqBody[P any] struct {
	query  Builder
	params P
}

func NewQueryReqBody[P any](q Builder, p ...P) *QueryReqBody[P] {
	return &QueryReqBody[P]{query: q, params: merge(p...)}
}

func (q QueryReqBody[O]) Map() (map[string]any, error) {
	base, err := newBase(q.params)
	if err != nil {
		return nil, err
	}

	if err := applyBuilder(base, "query", q.query); err != nil {
		return nil, err
	}

	return base, nil
}

func (q QueryReqBody[O]) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}

	return json.Marshal(data)
}

//----------------------------------------------------------------------------//

type BodyParamHighlight struct {
	HighlightOptions
	Fields map[string]HighlightOptions `json:"fields,omitempty"`
}

type HighlightType string

const (
	HighlightTypeUnified  HighlightType = "unified"
	HighlightTypeFvh      HighlightType = "fvh"
	HighlightTypePlain    HighlightType = "plain"
	HighlightTypeSemantic HighlightType = "semantic"
)

type HighlightOptions struct {
	Type                  HighlightType `json:"type,omitempty"`
	FragmentSize          int           `json:"fragment_size,omitempty"`
	NumberOfFragments     int           `json:"number_of_fragments,omitempty"`
	FragmentOffset        int           `json:"fragment_offset,omitempty"`
	BoundaryChars         string        `json:"boundary_chars,omitempty"`
	BoundaryMaxScan       int           `json:"boundary_max_scan,omitempty"`
	BoundaryScanner       string        `json:"boundary_scanner,omitempty"`
	BoundaryScannerLocale string        `json:"boundary_scanner_locale,omitempty"`
	Encoder               string        `json:"encoder,omitempty"`
	ForceSource           bool          `json:"force_source,omitempty"`
	Fragmenter            string        `json:"fragmenter,omitempty"`
	HighlightQuery        Builder       `json:"highlight_query,omitempty"`
	Order                 string        `json:"order,omitempty"`
	NoMatchSize           int           `json:"no_match_size,omitempty"`
	RequireFieldMatch     bool          `json:"require_field_match,omitempty"`
	MatchedFields         []string      `json:"matched_fields,omitempty"`
	PhraseLimit           int           `json:"phrase_limit,omitempty"`
	PreTags               []string      `json:"pre_tags,omitempty"`
	PostTags              []string      `json:"post_tags,omitempty"`
}

type BodyParamScript struct {
	Source string         `json:"source,omitempty"`
	Lang   string         `json:"lang,omitempty"`
	Params map[string]any `json:"params,omitempty"`
}

//----------------------------------------------------------------------------//

func BuildSearchReq(req *opensearchgoAPI.SearchReq, q Builder, p ...SearchBodyParams) (*opensearchgoAPI.SearchReq, error) {
	body, err := json.Marshal(NewQueryReqBody(q, p...))
	if err != nil {
		return nil, err
	}
	req.Body = bytes.NewReader(body)
	return req, nil
}

type SearchBodyParams struct {
	Highlight *BodyParamHighlight `json:"highlight,omitempty"`
	Aggs      map[string]any      `json:"aggs,omitempty"`
}

//----------------------------------------------------------------------------//

func BuildDocumentDeleteByQueryReq(req opensearchgoAPI.DocumentDeleteByQueryReq, q Builder) (opensearchgoAPI.DocumentDeleteByQueryReq, error) {
	body, err := json.Marshal(NewQueryReqBody[any](q))
	if err != nil {
		return req, err
	}
	req.Body = bytes.NewReader(body)
	return req, nil
}

//----------------------------------------------------------------------------//

func BuildUpdateByQueryReq(req opensearchgoAPI.UpdateByQueryReq, q Builder, o ...UpdateByQueryBodyParams) (opensearchgoAPI.UpdateByQueryReq, error) {
	body, err := json.Marshal(NewQueryReqBody(q, o...))
	if err != nil {
		return req, err
	}
	req.Body = bytes.NewReader(body)
	return req, nil
}

type UpdateByQueryBodyParams struct {
	Script *BodyParamScript `json:"script,omitempty"`
}

//----------------------------------------------------------------------------//

func BuildIndicesCountReq(req *opensearchgoAPI.IndicesCountReq, q Builder) (*opensearchgoAPI.IndicesCountReq, error) {
	body, err := json.Marshal(NewQueryReqBody[any](q))
	if err != nil {
		return nil, err
	}
	req.Body = bytes.NewReader(body)
	return req, nil
}
