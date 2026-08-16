package osu

import (
	"encoding/json"
	"fmt"
)

// KnnQuery builds an approximate k-NN query on a knn_vector field, optionally
// pre-filtered (the engine restricts the neighbor search to the filter
// matches).
type KnnQuery struct {
	field  string
	vector []float32
	k      int
	filter Builder
}

func NewKnnQuery(field string) *KnnQuery {
	return &KnnQuery{field: field}
}

func (q *KnnQuery) Vector(v []float32) *KnnQuery {
	q.vector = v
	return q
}

func (q *KnnQuery) K(k int) *KnnQuery {
	q.k = k
	return q
}

func (q *KnnQuery) Filter(f Builder) *KnnQuery {
	q.filter = f
	return q
}

func (q *KnnQuery) Map() (map[string]any, error) {
	if q.field == "" || len(q.vector) == 0 || q.k <= 0 {
		return nil, fmt.Errorf("knn query needs a field, a vector and k > 0")
	}
	inner := map[string]any{
		"vector": q.vector,
		"k":      q.k,
	}
	if q.filter != nil {
		f, err := q.filter.Map()
		if err != nil {
			return nil, err
		}
		inner["filter"] = f
	}
	return map[string]any{
		"knn": map[string]any{
			q.field: inner,
		},
	}, nil
}

func (q *KnnQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}
