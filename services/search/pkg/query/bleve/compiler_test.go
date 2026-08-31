package bleve

import (
	"strings"
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/opencloud-eu/opencloud/pkg/ast"
	tAssert "github.com/stretchr/testify/assert"
)

var timeMustParse = func(t *testing.T, ts string) time.Time {
	tp, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t.Fatalf("time.Parse(...) error = %v", err)
	}

	return tp
}

func wildcardQuery(field, value string) query.Query {
	patterns := []query.Query{query.NewQueryStringQuery(field + ".wildcard:" + value)}
	if !strings.HasSuffix(value, "*") {
		patterns = append(patterns, query.NewQueryStringQuery(field+".wildcard:"+value+".*"))
	}

	return query.NewConjunctionQuery([]query.Query{query.NewDisjunctionQuery(patterns)})
}

func phraseQuery(field, value string) query.Query {
	q := query.NewMatchPhraseQuery(value)
	q.SetField(field)

	return q
}

func boolFieldQuery(field string, value bool) query.Query {
	q := query.NewBoolFieldQuery(value)
	q.SetField(field)

	return q
}

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
				phraseQuery("Name", "federated"),
			}),
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
				phraseQuery("Name", "John Smith"),
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
				phraseQuery("Name", "John Smith"),
				phraseQuery("Name", "Jane"),
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
				query.NewQueryStringQuery(`Tags:bestseller`),
				query.NewQueryStringQuery(`Tags:book`),
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
				wildcardQuery("Name", `moby\ di*`),
				query.NewConjunctionQuery([]query.Query{
					query.NewQueryStringQuery(`Tags:bestseller`),
					query.NewQueryStringQuery(`Tags:book`),
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
					phraseQuery("Name", "a"),
					phraseQuery("Name", "b"),
				}),
				phraseQuery("Name", "c"),
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
				phraseQuery("Name", "a"),
				query.NewConjunctionQuery([]query.Query{
					phraseQuery("Name", "b"),
					phraseQuery("Name", "c"),
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
					phraseQuery("Name", "a"),
					phraseQuery("Name", "b"),
					phraseQuery("Name", "c"),
				}),
				phraseQuery("Name", "d"),
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
					wildcardQuery("Name", `moby\ di*`),
					query.NewQueryStringQuery(`Tags:bestseller`),
				}),
				query.NewQueryStringQuery(`Tags:book`),
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
					wildcardQuery("Name", `moby\ di*`),
					query.NewQueryStringQuery(`Tags:bestseller`),
				}),
				query.NewQueryStringQuery(`Tags:book`),
				query.NewBooleanQuery(nil, nil, []query.Query{query.NewQueryStringQuery(`Tags:read`)}),
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
				phraseQuery("author", "John Smith"),
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
				phraseQuery("author", "John Smith"),
				query.NewQueryStringQuery(`author:Jane`),
				query.NewQueryStringQuery(`Tags:bestseller`),
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
				phraseQuery("Name", "John Smith"),
				boolFieldQuery("Hidden", true),
				boolFieldQuery("Hidden", true),
			}),
			wantErr: false,
		},
		{
			name: `hidden:banana`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "hidden", Value: "banana"},
				},
			},
			want:    query.NewConjunctionQuery([]query.Query{query.NewMatchNoneQuery()}),
			wantErr: false,
		},
		{
			name: `name="Report.txt"`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "Report.txt", Exact: true},
				},
			},
			want:    query.NewConjunctionQuery([]query.Query{query.NewQueryStringQuery(`Name.wildcard:report.txt`)}),
			wantErr: false,
		},
		{
			name: `type:File`,
			args: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "type", Value: "File"},
					&ast.OperatorNode{Value: "OR"},
					&ast.StringNode{Key: "type", Value: "FOLDER"},
				},
			},
			want: query.NewDisjunctionQuery([]query.Query{
				query.NewQueryStringQuery(`Type:1`),
				query.NewQueryStringQuery(`Type:2`),
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
				query.NewBooleanQuery(nil, nil, []query.Query{query.NewQueryStringQuery(`Tags:physik`)}),
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
				wildcardQuery("Name", `*tdd*`),
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
					wildcardQuery("Name", `*tdd*`),
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
				wildcardQuery("Name", `*tdd*`),
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
				wildcardQuery("Name", `*tdd*`),
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
				phraseQuery("Name", "John Smith +-=&|><!(){}[]^\"~: "),
			}),
			wantErr: false,
		},
	}

	assert := tAssert.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compile(tt.args)

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
