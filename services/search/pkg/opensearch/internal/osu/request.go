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
	PreTags  []string                      `json:"pre_tags,omitempty"`
	PostTags []string                      `json:"post_tags,omitempty"`
	Fields   map[string]BodyParamHighlight `json:"fields,omitempty"`
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
