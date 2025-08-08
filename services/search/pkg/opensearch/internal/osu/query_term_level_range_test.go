package osu_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestRangeQuery(t *testing.T) {
	now := time.Now()
	tests := []opensearchtest.TableTest[osu.Builder, map[string]any]{
		{
			Name: "empty",
			Got:  osu.NewRangeQuery[string]("empty"),
			Want: nil,
		},
		{
			Name: "gt string",
			Got:  osu.NewRangeQuery[string]("created").Gt("2023-01-01T00:00:00Z"),
			Want: map[string]any{
				"range": map[string]any{
					"created": map[string]any{
						"gt": "2023-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			Name: "gt time",
			Got:  osu.NewRangeQuery[time.Time]("created").Gt(now),
			Want: map[string]any{
				"range": map[string]any{
					"created": map[string]any{
						"gt": now,
					},
				},
			},
		},
		{
			Name: "gte string",
			Got:  osu.NewRangeQuery[string]("created").Gte("2023-01-01T00:00:00Z"),
			Want: map[string]any{
				"range": map[string]any{
					"created": map[string]any{
						"gte": "2023-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			Name: "gte time",
			Got:  osu.NewRangeQuery[time.Time]("created").Gte(now),
			Want: map[string]any{
				"range": map[string]any{
					"created": map[string]any{
						"gte": now,
					},
				},
			},
		},
		{
			Name: "gt & gte",
			Got:  osu.NewRangeQuery[time.Time]("created").Gt(now).Gte(now),
			Want: nil,
			Err:  errors.New(""),
		},
		{
			Name: "gt string",
			Got:  osu.NewRangeQuery[string]("created").Lt("2023-01-01T00:00:00Z"),
			Want: map[string]any{
				"range": map[string]any{
					"created": map[string]any{
						"lt": "2023-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			Name: "lt time",
			Got:  osu.NewRangeQuery[time.Time]("created").Lt(now),
			Want: map[string]any{
				"range": map[string]any{
					"created": map[string]any{
						"lt": now,
					},
				},
			},
		},
		{
			Name: "lte string",
			Got:  osu.NewRangeQuery[string]("created").Lte("2023-01-01T00:00:00Z"),
			Want: map[string]any{
				"range": map[string]any{
					"created": map[string]any{
						"lte": "2023-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			Name: "lte time",
			Got:  osu.NewRangeQuery[time.Time]("created").Lte(now),
			Want: map[string]any{
				"range": map[string]any{
					"created": map[string]any{
						"lte": now,
					},
				},
			},
		},
		{
			Name: "lt & lte",
			Got:  osu.NewRangeQuery[time.Time]("created").Lt(now).Lte(now),
			Want: nil,
			Err:  errors.New(""),
		},
		{
			Name: "options",
			Got: osu.NewRangeQuery[time.Time]("created").Options(&osu.RangeQueryOptions{
				Format:   "strict_date_optional_time",
				Relation: "within",
				Boost:    1.0,
				TimeZone: "UTC",
			}).Lte(now).Gte(now),
			Want: map[string]any{
				"range": map[string]any{
					"created": map[string]any{
						"lte":       now,
						"gte":       now,
						"format":    "strict_date_optional_time",
						"relation":  "within",
						"boost":     1.0,
						"time_zone": "UTC",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			got, err := test.Got.MarshalJSON()
			switch {
			case test.Err != nil && test.Err.Error() == "": // Expecting any error
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			case test.Err != nil && test.Err.Error() != "": // Expecting a specific error
				assert.ErrorIs(t, test.Err, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, test.Want), string(got))
		})
	}
}
