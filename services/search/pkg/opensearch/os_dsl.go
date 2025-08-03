package opensearch

import (
	"encoding/json"
	"fmt"
)

type Rewrite string

const (
	ConstantScore         Rewrite = "constant_score"
	ScoringBoolean        Rewrite = "scoring_boolean"
	ConstantScoreBoolean  Rewrite = "constant_score_boolean"
	TopTermsN             Rewrite = "top_terms_N"
	TopTermsBoostN        Rewrite = "top_terms_boost_N"
	TopTermsBlendedFreqsN Rewrite = "top_terms_blended_freqs_N"
)

type Analyzer string

type Builder interface {
	json.Marshaler
	fmt.Stringer
	Map() (map[string]any, error)
}

type BuilderFunc func() (map[string]any, error)

func (f BuilderFunc) Map() (map[string]any, error) {
	return f()
}

func (f BuilderFunc) MarshalJSON() ([]byte, error) {
	data, err := f.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func (f BuilderFunc) String() string {
	b, _ := f.MarshalJSON()
	return string(b)
}

func applyValue[T any](target map[string]any, key string, v T) {
	if target == nil || isEmpty(key) || isEmpty(v) {
		return
	}

	target[key] = v
}

func applyValues[T any](target map[string]any, values map[string]T) {
	if target == nil || isEmpty(values) {
		return
	}

	for k, v := range values {
		applyValue[T](target, k, v)
	}
}

func applyBuilder(target map[string]any, key string, builder Builder) error {
	if target == nil || isEmpty(key) || isEmpty(builder) {
		return nil
	}

	data, err := builder.Map()
	if err != nil {
		return fmt.Errorf("failed to map builder %s: %w", key, err)
	}

	if isEmpty(data) {
		return nil
	}

	target[key] = data
	return nil
}

func applyBuilders(target map[string]any, key string, bs ...Builder) error {
	if target == nil || isEmpty(key) || isEmpty(bs) {
		return nil
	}

	builders := make([]map[string]any, len(bs))
	for i, builder := range bs {
		data, err := builder.Map()
		switch {
		case err != nil:
			return fmt.Errorf("failed to map builder %s: %w", key, err)
		case isEmpty(data):
			continue
		default:
			builders[i] = data
		}
	}

	if len(builders) > 0 {
		target[key] = builders
	}

	return nil
}

func builderToBoolQuery(b Builder) *BoolQuery {
	var bq *BoolQuery

	if q, ok := b.(*RootQuery); ok {
		b = q.query
	}

	if q, ok := b.(*BoolQuery); !ok {
		bq = NewBoolQuery().Must(b)
	} else {
		bq = q
	}

	return bq
}
