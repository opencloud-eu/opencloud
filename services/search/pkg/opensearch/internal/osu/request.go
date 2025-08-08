package osu

import (
	"bytes"
	"encoding/json"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

type RequestBody[O any] struct {
	query   Builder
	options O
}

func NewRequestBody[O any](q Builder, o ...O) *RequestBody[O] {
	return &RequestBody[O]{query: q, options: merge(o...)}
}

func (q RequestBody[O]) Map() (map[string]any, error) {
	base, err := newBase(q.options)
	if err != nil {
		return nil, err
	}

	if err := applyBuilder(base, "query", q.query); err != nil {
		return nil, err
	}

	return base, nil
}

func (q RequestBody[O]) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}

	return json.Marshal(data)
}

//----------------------------------------------------------------------------//

type HighlightOption struct {
	PreTags  []string                   `json:"pre_tags,omitempty"`
	PostTags []string                   `json:"post_tags,omitempty"`
	Fields   map[string]HighlightOption `json:"fields,omitempty"`
}

type ScriptOption struct {
	Source string         `json:"source,omitempty"`
	Lang   string         `json:"lang,omitempty"`
	Params map[string]any `json:"params,omitempty"`
}

//----------------------------------------------------------------------------//

func BuildSearchReq(req *opensearchgoAPI.SearchReq, q Builder, o ...SearchReqOptions) (*opensearchgoAPI.SearchReq, error) {
	body, err := json.Marshal(NewRequestBody(q, o...))
	if err != nil {
		return nil, err
	}
	req.Body = bytes.NewReader(body)
	return req, nil
}

type SearchReqOptions struct {
	Highlight *HighlightOption `json:"highlight,omitempty"`
}

//----------------------------------------------------------------------------//

func BuildDocumentDeleteByQueryReq(req opensearchgoAPI.DocumentDeleteByQueryReq, q Builder) (opensearchgoAPI.DocumentDeleteByQueryReq, error) {
	body, err := json.Marshal(NewRequestBody[any](q))
	if err != nil {
		return req, err
	}
	req.Body = bytes.NewReader(body)
	return req, nil
}

//----------------------------------------------------------------------------//

func BuildUpdateByQueryReq(req opensearchgoAPI.UpdateByQueryReq, q Builder, o ...UpdateByQueryReqOptions) (opensearchgoAPI.UpdateByQueryReq, error) {
	body, err := json.Marshal(NewRequestBody(q, o...))
	if err != nil {
		return req, err
	}
	req.Body = bytes.NewReader(body)
	return req, nil
}

type UpdateByQueryReqOptions struct {
	Script *ScriptOption `json:"script,omitempty"`
}

//----------------------------------------------------------------------------//

func BuildIndicesCountReq(req *opensearchgoAPI.IndicesCountReq, q Builder) (*opensearchgoAPI.IndicesCountReq, error) {
	body, err := json.Marshal(NewRequestBody[any](q))
	if err != nil {
		return nil, err
	}
	req.Body = bytes.NewReader(body)
	return req, nil
}
