package osu

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
)

type RequestBody[O any] struct {
	query   Builder
	options O
}

func NewRequestBody[O any](q Builder, o ...O) *RequestBody[O] {
	return &RequestBody[O]{query: q, options: merge(o...)}
}

func (q RequestBody[O]) Map() (map[string]any, error) {
	data, err := conversions.To[map[string]any](q.options)
	if err != nil {
		return nil, err
	}

	if err := applyBuilder(data, "query", q.query); err != nil {
		return nil, err
	}

	if isEmpty(data) {
		return nil, nil
	}

	return data, nil
}

func (q RequestBody[O]) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}

	return json.Marshal(data)
}

func (q RequestBody[O]) String() string {
	b, _ := q.MarshalJSON()
	return string(b)
}

func (q RequestBody[O]) Reader() io.Reader {
	return strings.NewReader(q.String())
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
	body := NewRequestBody(q, o...)
	data, err := body.MarshalJSON()
	if err != nil {
		return nil, err
	}
	req.Body = bytes.NewReader(data)
	return req, nil
}

type SearchReqOptions struct {
	Highlight *HighlightOption `json:"highlight,omitempty"`
}

//----------------------------------------------------------------------------//

func BuildDocumentDeleteByQueryReq(req opensearchgoAPI.DocumentDeleteByQueryReq, q Builder) (opensearchgoAPI.DocumentDeleteByQueryReq, error) {
	body := NewRequestBody[any](q)
	data, err := body.MarshalJSON()
	if err != nil {
		return req, err
	}
	req.Body = bytes.NewReader(data)
	return req, nil
}

//----------------------------------------------------------------------------//

func BuildUpdateByQueryReq(req opensearchgoAPI.UpdateByQueryReq, q Builder, o ...UpdateByQueryReqOptions) (opensearchgoAPI.UpdateByQueryReq, error) {
	body := NewRequestBody(q, o...)
	data, err := body.MarshalJSON()
	if err != nil {
		return req, err
	}
	req.Body = bytes.NewReader(data)
	return req, nil
}

type UpdateByQueryReqOptions struct {
	Script *ScriptOption `json:"script,omitempty"`
}

//----------------------------------------------------------------------------//

func BuildIndicesCountReq(req *opensearchgoAPI.IndicesCountReq, q Builder) (*opensearchgoAPI.IndicesCountReq, error) {
	body := NewRequestBody[any](q)
	data, err := body.MarshalJSON()
	if err != nil {
		return nil, err
	}
	req.Body = bytes.NewReader(data)
	return req, nil
}
