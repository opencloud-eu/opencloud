package bleve

import (
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/opencloud-eu/opencloud/pkg/ast"
	searchquery "github.com/opencloud-eu/opencloud/services/search/pkg/query"
	tAssert "github.com/stretchr/testify/assert"
)

var timeMustParse = func(t *testing.T, ts string) time.Time {
	tp, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t.Fatalf("time.Parse(...) error = %v", err)
	}

	return tp
}

// TODO(followup): make this a pure compiler test. Field resolution and
// media-type expansion live in query.Normalize, so this test could feed
// canonical ASTs (real field names, media-type already expanded) and call
// compile() directly, dropping the query.Normalize wrapper and the mediatype
// cases.
func Test_compile(t *testing.T) {
	tests := []struct {
		name    string
		args    *ast.Ast
		want    query.Query
		wantErr bool
	}{
		{
			name: `federated`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "federated"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`Name_lowercase:federated`),
			}),
			wantErr: false,
		},
		{
			// path fields expand to match the folder itself and its descendants,
			// since bleve has no path hierarchy analyzer.
			name: `path:/Foo`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "path", Value: "/Foo"},
				},
			},
			// a BooleanQuery (should: exact OR descendants), not a DisjunctionQuery,
			// so an enclosing AND does not redistribute the folder-itself clause.
			want: func() query.Query {
				bq := query.NewBooleanQuery(nil, []query.Query{
					query.NewQueryStringQuery(`Path_lowercase:\/foo`),
					query.NewQueryStringQuery(`Path_lowercase:\/foo\/*`),
				}, nil)
				bq.SetMinShould(1)
				return query.NewConjunctionQuery([]query.Query{bq})
			}(),
			wantErr: false,
		},
		{
			name: `"John Smith"`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "John Smith"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`Name_lowercase:john\ smith`),
			}),
			wantErr: false,
		},
		{
			name: `"John Smith" Jane`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "John Smith"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "name", Value: "Jane"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`Name_lowercase:john\ smith`),
				query.NewQueryStringQuery(`Name_lowercase:jane`),
			}),
			wantErr: false,
		},
		{
			name: `tag:bestseller tag:book`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "tag", Value: "bestseller"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "tag", Value: "book"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`Tags_lowercase:bestseller`),
				query.NewQueryStringQuery(`Tags_lowercase:book`),
			}),
			wantErr: false,
		},
		{
			name: `name:"moby di*" OR tag:bestseller AND tag:book`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "moby di*"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "tag", Value: "bestseller"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "tag", Value: "book"},
				},
			},
			want: query.NewDisjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`Name_lowercase:moby\ di*`),
				query.NewConjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`Tags_lowercase:bestseller`),
					query.NewQueryStringQuery(`Tags_lowercase:book`),
				}),
			}),
			wantErr: false,
		},
		{
			name: `a AND b OR c`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "a"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Value: "b"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Value: "c"},
				},
			},
			want: query.NewDisjunctionQuery([]query.Query{
				query.NewConjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`Name_lowercase:a`),
					query.NewQueryStringQuery(`Name_lowercase:b`),
				}),
				query.NewQueryStringQuery(`Name_lowercase:c`),
			}),
			wantErr: false,
		},
		{
			name: `a OR b AND c`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "a"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Value: "b"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Value: "c"},
				},
			},
			want: query.NewDisjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`Name_lowercase:a`),
				query.NewConjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`Name_lowercase:b`),
					query.NewQueryStringQuery(`Name_lowercase:c`),
				}),
			}),
			wantErr: false,
		},
		{
			name: `(a OR b OR c) AND d`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "a"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Value: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Value: "c"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Value: "d"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewDisjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`Name_lowercase:a`),
					query.NewQueryStringQuery(`Name_lowercase:b`),
					query.NewQueryStringQuery(`Name_lowercase:c`),
				}),
				query.NewQueryStringQuery(`Name_lowercase:d`),
			}),
			wantErr: false,
		},
		{
			name: `(name:"moby di*" OR tag:bestseller) AND tag:book`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "name", Value: "moby di*"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "tag", Value: "bestseller"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "tag", Value: "book"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewDisjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`Name_lowercase:moby\ di*`),
					query.NewQueryStringQuery(`Tags_lowercase:bestseller`),
				}),
				query.NewQueryStringQuery(`Tags_lowercase:book`),
			}),
			wantErr: false,
		},
		{
			name: `(name:"moby di*" OR tag:bestseller) AND tag:book AND NOT tag:read`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "name", Value: "moby di*"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "tag", Value: "bestseller"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "tag", Value: "book"},
					&ast.OperatorNode{Value: "AND"},
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "tag", Value: "read"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewDisjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`Name_lowercase:moby\ di*`),
					query.NewQueryStringQuery(`Tags_lowercase:bestseller`),
				}),
				query.NewQueryStringQuery(`Tags_lowercase:book`),
				query.NewBooleanQuery(nil, nil, []query.Query{query.NewQueryStringQuery(`Tags_lowercase:read`)}),
			}),
			wantErr: false,
		},
		{
			name: `author:("John Smith" Jane)`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Key: "author",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "John Smith"},
							&ast.OperatorNode{Value: "AND"},
							&ast.StringNode{Value: "Jane"},
						},
					},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`author:John\ Smith`),
				query.NewQueryStringQuery(`author:Jane`),
			}),
			wantErr: false,
		},
		{
			name: `author:("John Smith" Jane) AND tag:bestseller`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Key: "author",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "John Smith"},
							&ast.OperatorNode{Value: "AND"},
							&ast.StringNode{Value: "Jane"},
						},
					},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "tag", Value: "bestseller"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`author:John\ Smith`),
				query.NewQueryStringQuery(`author:Jane`),
				query.NewQueryStringQuery(`Tags_lowercase:bestseller`),
			}),
			wantErr: false,
		},
		{
			name: `id:b27d3bf1-b254-459f-92e8-bdba668d6d3f$d0648459-25fb-4ed8-8684-bc62c7dca29c!d0648459-25fb-4ed8-8684-bc62c7dca29c mtime>=2023-09-05T12:40:59.14741+02:00`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{
						Key:   "id",
						Value: "b27d3bf1-b254-459f-92e8-bdba668d6d3f$d0648459-25fb-4ed8-8684-bc62c7dca29c!d0648459-25fb-4ed8-8684-bc62c7dca29c",
					},
					&ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`ID:b27d3bf1-b254-459f-92e8-bdba668d6d3f$d0648459-25fb-4ed8-8684-bc62c7dca29c!d0648459-25fb-4ed8-8684-bc62c7dca29c`),
				func() query.Query {
					q := query.NewDateRangeInclusiveQuery(timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"), time.Time{}, &[]bool{true}[0], nil)
					q.FieldVal = "Mtime"
					return q
				}(),
			}),
			wantErr: false,
		},
		{
			name: `StringNode value lowercase`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "John Smith"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Hidden", Value: "T"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "hidden", Value: "T"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`Name_lowercase:john\ smith`),
				query.NewQueryStringQuery(`Hidden:T`),
				query.NewQueryStringQuery(`Hidden:T`),
			}),
			wantErr: false,
		},
		{
			name: `NOT tag:physik`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "tag", Value: "physik"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewBooleanQuery(nil, nil, []query.Query{query.NewQueryStringQuery(`Tags_lowercase:physik`)}),
			}),
			wantErr: false,
		},
		{
			name: `ast.DateTimeNode`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key: "mtime",
						// "=" is not supported by bleve, ignore
						Operator: &ast.OperatorNode{Value: "="},
						Value:    timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{
						Key: "mtime",
						// ":" is not supported by bleve, ignore
						Operator: &ast.OperatorNode{Value: ":"},
						Value:    timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{
						Key: "mtime",
						// no operator, skip
						Value: timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{
						Key:      "mtime",
						Operator: &ast.OperatorNode{Value: ">"},
						Value:    timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{
						Key:      "mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{
						Key:      "mtime",
						Operator: &ast.OperatorNode{Value: "<"},
						Value:    timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{
						Key:      "mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				func() query.Query {
					q := query.NewDateRangeInclusiveQuery(timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"), time.Time{}, &[]bool{false}[0], nil)
					q.FieldVal = "Mtime"
					return q
				}(),
				func() query.Query {
					q := query.NewDateRangeInclusiveQuery(timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"), time.Time{}, &[]bool{true}[0], nil)
					q.FieldVal = "Mtime"
					return q
				}(),
				func() query.Query {
					q := query.NewDateRangeInclusiveQuery(time.Time{}, timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"), nil, &[]bool{false}[0])
					q.FieldVal = "Mtime"
					return q
				}(),
				func() query.Query {
					q := query.NewDateRangeInclusiveQuery(time.Time{}, timeMustParse(t, "2023-09-05T08:42:11.23554+02:00"), nil, &[]bool{true}[0])
					q.FieldVal = "Mtime"
					return q
				}(),
			}),
			wantErr: false,
		},
		{
			name: `MimeType:document`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "mediatype", Value: "document"},
				},
			},
			want: query.NewDisjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`MimeType:application/msword`),
				query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.document`),
				query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.form`),
				query.NewQueryStringQuery(`MimeType:application/vnd.oasis.opendocument.text`),
				query.NewQueryStringQuery(`MimeType:text/plain`),
				query.NewQueryStringQuery(`MimeType:text/markdown`),
				query.NewQueryStringQuery(`MimeType:application/rtf`),
				query.NewQueryStringQuery(`MimeType:application/vnd.apple.pages`),
			}),
			wantErr: false,
		},
		{
			name: `MimeType:document AND *tdd*`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "mediatype", Value: "document"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "name", Value: "*tdd*"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewDisjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`MimeType:application/msword`),
					query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.document`),
					query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.form`),
					query.NewQueryStringQuery(`MimeType:application/vnd.oasis.opendocument.text`),
					query.NewQueryStringQuery(`MimeType:text/plain`),
					query.NewQueryStringQuery(`MimeType:text/markdown`),
					query.NewQueryStringQuery(`MimeType:application/rtf`),
					query.NewQueryStringQuery(`MimeType:application/vnd.apple.pages`),
				}),
				query.NewQueryStringQuery(`Name_lowercase:*tdd*`),
			}),
			wantErr: false,
		},
		{
			name: `MimeType:document OR MimeType:pdf AND *tdd*`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "mediatype", Value: "document"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "mediatype", Value: "pdf"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "name", Value: "*tdd*"},
				},
			},
			want: query.NewDisjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`MimeType:application/msword`),
				query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.document`),
				query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.form`),
				query.NewQueryStringQuery(`MimeType:application/vnd.oasis.opendocument.text`),
				query.NewQueryStringQuery(`MimeType:text/plain`),
				query.NewQueryStringQuery(`MimeType:text/markdown`),
				query.NewQueryStringQuery(`MimeType:application/rtf`),
				query.NewQueryStringQuery(`MimeType:application/vnd.apple.pages`),
				query.NewConjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`MimeType:application/pdf`),
					query.NewQueryStringQuery(`Name_lowercase:*tdd*`),
				}),
			}),
			wantErr: false,
		},
		{
			name: `(MimeType:document OR MimeType:pdf) AND *tdd*`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "mediatype", Value: "document"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "mediatype", Value: "pdf"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "name", Value: "*tdd*"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewDisjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`MimeType:application/msword`),
					query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.document`),
					query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.form`),
					query.NewQueryStringQuery(`MimeType:application/vnd.oasis.opendocument.text`),
					query.NewQueryStringQuery(`MimeType:text/plain`),
					query.NewQueryStringQuery(`MimeType:text/markdown`),
					query.NewQueryStringQuery(`MimeType:application/rtf`),
					query.NewQueryStringQuery(`MimeType:application/vnd.apple.pages`),
					query.NewQueryStringQuery(`MimeType:application/pdf`),
				}),
				query.NewQueryStringQuery(`Name_lowercase:*tdd*`),
			}),
			wantErr: false,
		},
		{
			name: `(MimeType:pdf OR MimeType:document) AND *tdd*`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "mediatype", Value: "pdf"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "mediatype", Value: "document"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "name", Value: "*tdd*"},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewDisjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`MimeType:application/pdf`),
					query.NewQueryStringQuery(`MimeType:application/msword`),
					query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.document`),
					query.NewQueryStringQuery(`MimeType:application/vnd.openxmlformats-officedocument.wordprocessingml.form`),
					query.NewQueryStringQuery(`MimeType:application/vnd.oasis.opendocument.text`),
					query.NewQueryStringQuery(`MimeType:text/plain`),
					query.NewQueryStringQuery(`MimeType:text/markdown`),
					query.NewQueryStringQuery(`MimeType:application/rtf`),
					query.NewQueryStringQuery(`MimeType:application/vnd.apple.pages`),
				}),
				query.NewQueryStringQuery(`Name_lowercase:*tdd*`),
			}),
			wantErr: false,
		},
		{
			name: `John Smith`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "John Smith +-=&|><!(){}[]^\"~: "},
				},
			},
			want: query.NewConjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`Name_lowercase:john\ smith\ \+\-\=\&\|\>\<\!\(\)\{\}\[\]\^\"\~\:\ `),
			}),
			wantErr: false,
		},
	}

	assert := tAssert.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compile(searchquery.Normalize(tt.args, searchquery.ResolveField))

			if (err != nil) != tt.wantErr {
				t.Errorf("compile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func Test_escape(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "all escaped",
			args: args{
				`+-=&|><!(){}[]^"~:\/ `,
			},
			want: `\+\-\=\&\|\>\<\!\(\)\{\}\[\]\^\"\~\:\\\/\ `,
		},
		{
			name: "no one escaped",
			args: args{
				`@#$%`,
			},
			want: `@#$%`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tAssert.Equalf(t, tt.want, bleveEscaper.Replace(tt.args.str), "bleveEscaper(%v)", tt.args.str)
		})
	}
}
