package convert_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/convert"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
)

func TestTranspileKQLToOpenSearch(t *testing.T) {
	tests := []opensearchtest.TableTest[*ast.Ast, osu.Builder]{
		// kql to os dsl - type tests
		{
			Name: "word-broken field matches the value as a phrase on its words sibling",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
				},
			},
			Want: osu.NewMatchPhraseQuery("Name_words").Query("openCloud"),
		},
		{
			Name: "case-insensitive term routes to the lowercased sibling",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Tags", Value: "openCloud", CaseInsensitive: true},
				},
			},
			Want: osu.NewTermQuery[string]("Tags_lowercase").Value("opencloud"),
		},
		{
			Name: "case-insensitive wildcard routes to the lowercased sibling",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "Open*", CaseInsensitive: true},
				},
			},
			Want: osu.NewWildcardQuery("Name_lowercase").Value("open*"),
		},
		{
			Name: "full-text field uses an analyzed match query, not an unanalyzed term",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Content", Value: "Running"},
				},
			},
			Want: osu.NewMatchPhraseQuery("Content").Query("Running"),
		},
		{
			Name: "term query - boolean node - true",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: true},
				},
			},
			Want: osu.NewTermQuery[bool]("Deleted").Value(true),
		},
		{
			Name: "term query - boolean node - false",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
				},
			},
			Want: osu.NewTermQuery[bool]("Deleted").Value(false),
		},
		{
			Name: "match-phrase query - string node",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "open cloud"},
				},
			},
			Want: osu.NewMatchPhraseQuery("Name_words").Query(`open cloud`),
		},
		{
			Name: "wildcard query - string node",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "open*"},
				},
			},
			Want: osu.NewWildcardQuery("Name").Value("open*"),
		},
		{
			Name: "wildcard query - fulltext field",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Content", Value: "open*"},
				},
			},
			Want: osu.NewWildcardQuery("Content").Value("open*"),
		},
		{
			// a phrase match would analyze the query with path_hierarchy and match
			// everything under the root
			Name: "path with spaces stays an unanalyzed term query",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Path", Value: "./parent d!r/child.pdf"},
				},
			},
			Want: osu.NewTermQuery[string]("Path").Value("./parent d!r/child.pdf"),
		},
		{
			Name: "case-insensitive path with spaces routes to the lowercased sibling as a term query",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Path", Value: "./Parent Dir", CaseInsensitive: true},
				},
			},
			Want: osu.NewTermQuery[string]("Path_lowercase").Value("./parent dir"),
		},
		{
			Name: "bool query",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "Name", Value: "a"},
						&ast.StringNode{Key: "Name", Value: "b"},
					}},
				},
			},
			Want: osu.NewBoolQuery().Must(
				osu.NewMatchPhraseQuery("Name_words").Query("a"),
				osu.NewMatchPhraseQuery("Name_words").Query("b"),
			),
		},
		{
			Name: "no bool query for single term",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "Name", Value: "any"},
					}},
				},
			},
			Want: osu.NewMatchPhraseQuery("Name_words").Query("any"),
		},
		{
			Name: "range query >",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">"},
						Value:    opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			Want: osu.NewRangeQuery[time.Time]("Mtime").Gt(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
		},
		{
			Name: "range query >=",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			Want: osu.NewRangeQuery[time.Time]("Mtime").Gte(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
		},
		{
			Name: "range query <",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<"},
						Value:    opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			Want: osu.NewRangeQuery[time.Time]("Mtime").Lt(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
		},
		{
			Name: "range query <=",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			Want: osu.NewRangeQuery[time.Time]("Mtime").Lte(opensearchtest.TimeMustParse(t, "2023-09-05T08:42:11.23554+02:00")),
		},
		// kql to os dsl - structure tests
		{
			Name: "[*]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
				},
			},
			Want: osu.NewMatchPhraseQuery("Name_words").Query("openCloud"),
		},
		{
			Name: "[* *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewMatchPhraseQuery("Name_words").Query("openCloud"),
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewMatchPhraseQuery("Name_words").Query("openCloud"),
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* OR *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Should(
					osu.NewMatchPhraseQuery("Name_words").Query("openCloud"),
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[NOT *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				MustNot(
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			Name: "[* NOT *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "age", Value: "32"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewMatchPhraseQuery("Name_words").Query("openCloud"),
				).
				MustNot(
					osu.NewTermQuery[string]("age").Value("32"),
				),
		},
		{
			// NOT binds to the node directly after it, not to whatever operator
			// follows that node: NOT x AND y is (NOT x) AND y.
			Name: "[NOT * AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "age", Value: "32"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: "openCloud"},
				},
			},
			Want: osu.NewBoolQuery().
				MustNot(
					osu.NewTermQuery[string]("age").Value("32"),
				).
				Must(
					osu.NewMatchPhraseQuery("Name_words").Query("openCloud"),
				),
		},
		{
			Name: "[* OR * OR *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "Name", Value: "openCloud"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "32"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "age", Value: "44"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Should(
					osu.NewMatchPhraseQuery("Name_words").Query("openCloud"),
					osu.NewTermQuery[string]("age").Value("32"),
					osu.NewTermQuery[string]("age").Value("44"),
				),
		},
		{
			Name: "[* AND * OR *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "a", Value: "a"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "b", Value: "b"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "c", Value: "c"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Must(
					osu.NewTermQuery[string]("a").Value("a"),
				).
				Should(
					osu.NewTermQuery[string]("b").Value("b"),
					osu.NewTermQuery[string]("c").Value("c"),
				),
		},
		{
			Name: "[* OR * AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "a", Value: "a"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "b", Value: "b"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "c", Value: "c"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Must(
					osu.NewTermQuery[string]("b").Value("b"),
					osu.NewTermQuery[string]("c").Value("c"),
				).
				Should(
					osu.NewTermQuery[string]("a").Value("a"),
				),
		},
		{
			Name: "[* OR * AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "a", Value: "a"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "b", Value: "b"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "c", Value: "c"},
				},
			},
			Want: osu.NewBoolQuery().
				Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
				Should(
					osu.NewTermQuery[string]("a").Value("a"),
				).
				Must(
					osu.NewTermQuery[string]("b").Value("b"),
					osu.NewTermQuery[string]("c").Value("c"),
				),
		},
		{
			Name: "[[* OR * OR *] AND *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "a", Value: "a"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "b", Value: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "c"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "d", Value: "d"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewBoolQuery().
						Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
						Should(
							osu.NewTermQuery[string]("a").Value("a"),
							osu.NewTermQuery[string]("b").Value("b"),
							osu.NewTermQuery[string]("c").Value("c"),
						),
					osu.NewTermQuery[string]("d").Value("d"),
				),
		},
		{
			Name: "[[* OR * OR *] AND NOT *]",
			Got: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "a", Value: "a"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "b", Value: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "c"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "d", Value: "d"},
				},
			},
			Want: osu.NewBoolQuery().
				Must(
					osu.NewBoolQuery().
						Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
						Should(
							osu.NewTermQuery[string]("a").Value("a"),
							osu.NewTermQuery[string]("b").Value("b"),
							osu.NewTermQuery[string]("c").Value("c"),
						),
				).MustNot(
				osu.NewTermQuery[string]("d").Value("d"),
			),
		},
		{
			Name: "geo_distance - geo distance node",
			Got: &ast.Ast{Nodes: []ast.Node{
				&ast.GeoDistanceNode{Key: "location", Lat: 48.2, Lon: 16.3, Radius: 5000},
			}},
			Want: osu.NewGeoDistanceQuery("location_geopoint").Distance("5000m").Point(48.2, 16.3),
		},
		{
			Name: "geo_bounding_box - geo bbox node",
			Got: &ast.Ast{Nodes: []ast.Node{
				&ast.GeoBoundingBoxNode{Key: "location", MinLat: 47.9, MinLon: 16.1, MaxLat: 48.3, MaxLon: 16.5},
			}},
			Want: osu.NewGeoBoundingBoxQuery("location_geopoint").Box(47.9, 16.1, 48.3, 16.5),
		},
		{
			Name: "geo_polygon - geo polygon node",
			Got: &ast.Ast{Nodes: []ast.Node{
				&ast.GeoPolygonNode{Key: "location", Points: []ast.GeoPoint{{Lat: 48.3, Lon: 16.1}, {Lat: 48.3, Lon: 16.5}, {Lat: 47.9, Lon: 16.5}}},
			}},
			Want: osu.NewGeoPolygonQuery("location_geopoint").Point(48.3, 16.1).Point(48.3, 16.5).Point(47.9, 16.5),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if test.Skip {
				t.Skip("skipping test: " + test.Name)
			}

			dsl, err := convert.TranspileKQLToOpenSearch(test.Got.Nodes)
			assert.NoError(t, err)

			assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, test.Want), opensearchtest.JSONMustMarshal(t, dsl))
		})
	}
}
