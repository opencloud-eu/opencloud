package kql_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jinzhu/now"
	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/ast/test"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
	tAssert "github.com/stretchr/testify/assert"
)

func TestParse_Spec(t *testing.T) {
	// SPEC //////////////////////////////////////////////////////////////////////////////
	//
	// https://msopenspecs.azureedge.net/files/MS-KQL/%5bMS-KQL%5d.pdf
	// https://learn.microsoft.com/en-us/openspecs/sharepoint_protocols/ms-kql/3bbf06cd-8fc1-4277-bd92-8661ccd3c9b0
	// https://learn.microsoft.com/en-us/sharepoint/dev/general-development/keyword-query-language-kql-syntax-reference
	tests := []testCase{
		// 2.1.2 AND Operator
		// 3.1.2 AND Operator
		{
			name: `cat AND dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
		{
			name: `AND`,
			error: query.StartsWithBinaryOperatorError{
				Node: &ast.OperatorNode{Value: kql.BoolAND},
			},
		},
		{
			name: `AND cat AND dog`,
			error: query.StartsWithBinaryOperatorError{
				Node: &ast.OperatorNode{Value: kql.BoolAND},
			},
		},
		// 2.1.6 NOT Operator
		// 3.1.6 NOT Operator
		{
			name: `cat NOT dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
		{
			name: `NOT dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
		// 2.1.8 OR Operator
		// 3.1.8 OR Operator
		{
			name: `cat OR dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
		{
			name: `OR`,
			error: query.StartsWithBinaryOperatorError{
				Node: &ast.OperatorNode{Value: kql.BoolOR},
			},
		},
		{
			name: `OR cat AND dog`,
			error: query.StartsWithBinaryOperatorError{
				Node: &ast.OperatorNode{Value: kql.BoolOR},
			},
		},
		// 3.1.11 Implicit Operator
		{
			name: `cat dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
		{
			name: `cat AND (dog OR fox)`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "dog"},
						&ast.OperatorNode{Value: kql.BoolOR},
						&ast.StringNode{Value: "fox"},
					}},
				},
			},
		},
		{
			name: `cat (dog OR fox)`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "dog"},
						&ast.OperatorNode{Value: kql.BoolOR},
						&ast.StringNode{Value: "fox"},
					}},
				},
			},
		},
		// 2.1.12 Parentheses
		// 3.1.12 Parentheses
		{
			name: `(cat OR dog) AND fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "cat"},
						&ast.OperatorNode{Value: kql.BoolOR},
						&ast.StringNode{Value: "dog"},
					}},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		// 3.2.3 Implicit Operator for Property Restriction
		{
			name: `author:"John Smith" filetype:docx`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "author", Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "filetype", Value: "docx"},
				},
			},
		},
		{
			name: `author:"John Smith" AND filetype:docx`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "author", Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "filetype", Value: "docx"},
				},
			},
		},
		{
			name: `author:"John Smith" author:"Jane Smith"`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "author", Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "author", Value: "Jane Smith"},
				},
			},
		},
		{
			name: `author:"John Smith" OR author:"Jane Smith"`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "author", Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "author", Value: "Jane Smith"},
				},
			},
		},
		{
			name: `cat filetype:docx`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "filetype", Value: "docx"},
				},
			},
		},
		{
			name: `cat AND filetype:docx`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "filetype", Value: "docx"},
				},
			},
		},
		// 3.3.1.1.1 Implicit AND Operator
		{
			name: `cat +dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
		{
			name: `cat AND dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
		{
			name: `cat -dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
		{
			name: `cat AND NOT dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
		{
			name: `cat +dog -fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		{
			name: `cat AND dog AND NOT fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		{
			name: `cat dog +fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		{
			name: `fox OR (fox AND (cat OR dog))`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "fox"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "fox"},
						&ast.OperatorNode{Value: kql.BoolAND},
						&ast.GroupNode{Nodes: []ast.Node{
							&ast.StringNode{Value: "cat"},
							&ast.OperatorNode{Value: kql.BoolOR},
							&ast.StringNode{Value: "dog"},
						}},
					}},
				},
			},
		},
		{
			name: `cat dog -fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		{
			name: `(NOT fox) AND (cat OR dog)`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.OperatorNode{Value: kql.BoolNOT},
						&ast.StringNode{Value: "fox"},
					}},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "cat"},
						&ast.OperatorNode{Value: kql.BoolOR},
						&ast.StringNode{Value: "dog"},
					}},
				},
			},
		},
		{
			name: `cat +dog -fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		{
			name: `(NOT fox) AND (dog OR (dog AND cat))`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.OperatorNode{Value: kql.BoolNOT},
						&ast.StringNode{Value: "fox"},
					}},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "dog"},
						&ast.OperatorNode{Value: kql.BoolOR},
						&ast.GroupNode{Nodes: []ast.Node{
							&ast.StringNode{Value: "dog"},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{Value: "cat"},
						}},
					}},
				},
			},
		},
		// 2.3.5 Date Tokens
		// 3.3.5 Date Tokens
		{
			name: `Modified:2023-09-05`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Modified",
						Operator: &ast.OperatorNode{Value: ":"},
						Value:    mustParseTime(t, "2023-09-05"),
					},
				},
			},
		},
		{
			name: `Modified:"2008-01-29"`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Modified",
						Operator: &ast.OperatorNode{Value: ":"},
						Value:    mustParseTime(t, "2008-01-29"),
					},
				},
			},
		},
		{
			name:  `Modified:today`,
			patch: patchNow(t, "2023-09-10"),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Modified",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-09-10"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Modified",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-10 23:59:59.999999999"),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testKQL(t, tc)
		})
	}
}

func TestParse_DateTimeRestrictionNode(t *testing.T) {
	tests := []testCase{
		{
			name: "format",
			query: join([]string{
				`Mtime:"2023-09-05T08:42:11.23554+02:00"`,
				`Mtime:2023-09-05T08:42:11.23554+02:00`,
				`Mtime="2023-09-05T08:42:11.23554+02:00"`,
				`Mtime=2023-09-05T08:42:11.23554+02:00`,
				`Mtime<"2023-09-05T08:42:11.23554+02:00"`,
				`Mtime<2023-09-05T08:42:11.23554+02:00`,
				`Mtime<="2023-09-05T08:42:11.23554+02:00"`,
				`Mtime<=2023-09-05T08:42:11.23554+02:00`,
				`Mtime>"2023-09-05T08:42:11.23554+02:00"`,
				`Mtime>2023-09-05T08:42:11.23554+02:00`,
				`Mtime>="2023-09-05T08:42:11.23554+02:00"`,
				`Mtime>=2023-09-05T08:42:11.23554+02:00`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ":"},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ":"},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "="},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "="},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<"},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<"},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">"},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">"},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-09-05T08:42:11.23554+02:00"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - today",
			patch: setWorldClock(t, "2023-09-10"),
			query: join([]string{
				`Mtime:today`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-09-10"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-10 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - yesterday",
			patch: setWorldClock(t, "2023-09-10"),
			query: join([]string{
				`Mtime:yesterday`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-09-09"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-09 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - yesterday - the beginning of the month",
			patch: setWorldClock(t, "2023-09-01"),
			query: join([]string{
				`Mtime:yesterday`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-08-31"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-08-31 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - this week",
			patch: setWorldClock(t, "2023-09-06"),
			query: join([]string{
				`Mtime:"this week"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-09-04"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-10 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - last week",
			patch: setWorldClock(t, "2023-09-06"),
			query: join([]string{
				`Mtime:"last week"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-08-28"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-03 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - last 7 days",
			patch: setWorldClock(t, "2023-09-06"),
			query: join([]string{
				`Mtime:"last 7 days"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-08-31"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-06 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - this month",
			patch: setWorldClock(t, "2023-09-02"),
			query: join([]string{
				`Mtime:"this month"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-09-01"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-30 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - last month",
			patch: setWorldClock(t, "2023-09-02"),
			query: join([]string{
				`Mtime:"last month"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-08-01"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-08-31 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - last month - edge case when last day of the month",
			patch: setWorldClock(t, "2023-10-31"),
			query: join([]string{
				`Mtime:"last month"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-09-01"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-30 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - last month - edge case when last day of the month",
			patch: setWorldClock(t, "2023-03-31"),
			query: join([]string{
				`Mtime:"last month"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-02-01"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-02-28 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - last month - edge case when last day of the month",
			patch: setWorldClock(t, "2024-03-31"),
			query: join([]string{
				`Mtime:"last month"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2024-02-01"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2024-02-29 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - last month - the beginning of the year",
			patch: setWorldClock(t, "2023-01-01"),
			query: join([]string{
				`Mtime:"last month"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2022-12-01"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2022-12-31 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - last 30 days",
			patch: setWorldClock(t, "2023-09-06"),
			query: join([]string{
				`Mtime:"last 30 days"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-08-08"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-09-06 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - this year",
			patch: setWorldClock(t, "2023-06-18"),
			query: join([]string{
				`Mtime:"this year"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2023-01-01"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2023-12-31 23:59:59.999999999"),
					},
				},
			},
		},
		{
			name:  "NaturalLanguage DateTimeNode - last year",
			patch: setWorldClock(t, "2023-01-01"),
			query: join([]string{
				`Mtime:"last year"`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: ">="},
						Value:    mustParseTime(t, "2022-01-01"),
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.DateTimeNode{
						Key:      "Mtime",
						Operator: &ast.OperatorNode{Value: "<="},
						Value:    mustParseTime(t, "2022-12-31 23:59:59.999999999"),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testKQL(t, tc)
		})
	}
}

func TestParse_Errors(t *testing.T) {
	tests := []testCase{
		{
			query: "animal:(mammal:cat mammal:dog reptile:turtle)",
			error: query.NamedGroupInvalidNodesError{
				Node: &ast.StringNode{Key: "mammal", Value: "cat"},
			},
		},
		{
			query: "animal:(cat mammal:dog turtle)",
			error: query.NamedGroupInvalidNodesError{
				Node: &ast.StringNode{Key: "mammal", Value: "dog"},
			},
		},
		{
			query: "animal:(AND cat)",
			error: query.StartsWithBinaryOperatorError{
				Node: &ast.OperatorNode{Value: kql.BoolAND},
			},
		},
		{
			query: "animal:(OR cat)",
			error: query.StartsWithBinaryOperatorError{
				Node: &ast.OperatorNode{Value: kql.BoolOR},
			},
		},
		{
			query: "(AND cat)",
			error: query.StartsWithBinaryOperatorError{
				Node: &ast.OperatorNode{Value: kql.BoolAND},
			},
		},
		{
			query: "(OR cat)",
			error: query.StartsWithBinaryOperatorError{
				Node: &ast.OperatorNode{Value: kql.BoolOR},
			},
		},
		{
			query: `cat dog`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testKQL(t, tc)
		})
	}
}

func TestParse_DottedKey(t *testing.T) {
	tests := []testCase{
		// TextPropertyRestrictionNode with a dotted key.
		{
			name: `audio.artist:Motörhead`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "audio.artist", Value: "Motörhead"},
				},
			},
		},
		// Multi-level key exercises the ("." Char+)* Kleene in the Key rule.
		{
			name: `foo.bar.baz:qux`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "foo.bar.baz", Value: "qux"},
				},
			},
		},
		// GroupNode also uses the Key rule via k:Key?.
		{
			name: `audio.artist:("Motörhead" OR Beatles)`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Key: "audio.artist",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "Motörhead"},
							&ast.OperatorNode{Value: kql.BoolOR},
							&ast.StringNode{Value: "Beatles"},
						},
					},
				},
			},
		},
		// A dot on the value side is still a literal character.
		{
			name: `name:foo.bar`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Key: "name", Value: "foo.bar"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testKQL(t, tc)
		})
	}
}

func TestParse_Stress(t *testing.T) {
	tests := []testCase{
		{
			name:  "FullDictionary",
			query: join(FullDictionary),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "federated"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "search"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "federat*"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "search"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "search"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "fed*"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "author", Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "filetype", Value: "docx"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "filename", Value: "budget.xlsx"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "author"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "author"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "author"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "author"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "author"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "author", Value: "Shakespear"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "author", Value: "Paul"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "author", Value: "Shakesp*"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "title", Value: "Advanced Search"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "title", Value: "Advanced Sear*"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "title", Value: "Advan* Search"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "title", Value: "*anced Search"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "author", Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "author", Value: "Jane Smith"},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "author", Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "filetype", Value: "docx"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.GroupNode{
						Key: "author",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "John Smith"},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{Value: "Jane Smith"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.GroupNode{
						Key: "author",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "John Smith"},
							&ast.OperatorNode{Value: kql.BoolOR},
							&ast.StringNode{Value: "Jane Smith"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.GroupNode{
						Nodes: []ast.Node{
							&ast.StringNode{Key: "DepartmentId", Value: "*"},
							&ast.OperatorNode{Value: kql.BoolOR},
							&ast.StringNode{Key: "RelatedHubSites", Value: "*"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "contentclass", Value: "sts_site"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.BooleanNode{Key: "IsHubSite", Value: false},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "author", Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.GroupNode{
						Nodes: []ast.Node{
							&ast.StringNode{Key: "filetype", Value: "docx"},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{Key: "title", Value: "Advanced Search"},
						},
					},
				},
			},
		},
		{
			name: "complex",
			query: join([]string{
				`(name:"moby di*" OR tag:bestseller) AND tag:book NOT tag:read`,
				`author:("John Smith" Jane)`,
				`author:("John Smith" OR Jane)`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Nodes: []ast.Node{
							&ast.StringNode{Key: "name", Value: "moby di*"},
							&ast.OperatorNode{Value: kql.BoolOR},
							&ast.StringNode{Key: "tag", Value: "bestseller"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "tag", Value: "book"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Key: "tag", Value: "read"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.GroupNode{
						Key: "author",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "John Smith"},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{Value: "Jane"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.GroupNode{
						Key: "author",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "John Smith"},
							&ast.OperatorNode{Value: kql.BoolOR},
							&ast.StringNode{Value: "Jane"},
						},
					},
				},
			},
		},
		{
			name: `author:("John Smith" Jane) author:"Jack" AND author:"Oggy"`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Key: "author",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "John Smith"},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{Value: "Jane"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "author", Value: "Jack"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Key: "author", Value: "Oggy"},
				},
			},
		},
		{
			name: `author:("John Smith" OR Jane)`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Key: "author",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "John Smith"},
							&ast.OperatorNode{Value: kql.BoolOR},
							&ast.StringNode{Value: "Jane"},
						},
					},
				},
			},
		},
		{
			name: `NOT "John Smith" NOT Jane`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Value: "Jane"},
				},
			},
		},
		{
			name: `NOT author:"John Smith" NOT author:"Jane Smith" NOT tag:sifi`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Key: "author", Value: "John Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Key: "author", Value: "Jane Smith"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.OperatorNode{Value: kql.BoolNOT},
					&ast.StringNode{Key: "tag", Value: "sifi"},
				},
			},
		},
		{
			name: `scope:"<uuid>/new folder/subfolder" file`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{
						Key:   "scope",
						Value: "<uuid>/new folder/subfolder",
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{
						Value: "file",
					},
				},
			},
		},
		{
			name: `	😂 "*😀 😁*" name:😂💁👌🎍😍 name:😂💁👌 😍`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{
						Value: "😂",
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{
						Value: "*😀 😁*",
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{
						Key:   "name",
						Value: "😂💁👌🎍😍",
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{
						Key:   "name",
						Value: "😂💁👌",
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{
						Value: "😍",
					},
				},
			},
		},
		{
			name: "animal:(cat dog turtle)",
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Key: "animal",
						Nodes: []ast.Node{
							&ast.StringNode{
								Value: "cat",
							},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{
								Value: "dog",
							},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{
								Value: "turtle",
							},
						},
					},
				},
			},
		},
		{
			name: "(cat dog turtle)",
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Nodes: []ast.Node{
							&ast.StringNode{
								Value: "cat",
							},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{
								Value: "dog",
							},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{
								Value: "turtle",
							},
						},
					},
				},
			},
		},
		{
			name: `cat dog fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{Value: "cat"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "dog"},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		{
			name: `(cat dog) fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Nodes: []ast.Node{
							&ast.StringNode{Value: "cat"},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{Value: "dog"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		{
			name: `(mammal:cat mammal:dog) fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Nodes: []ast.Node{
							&ast.StringNode{Key: "mammal", Value: "cat"},
							&ast.OperatorNode{Value: kql.BoolOR},
							&ast.StringNode{Key: "mammal", Value: "dog"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		{
			name: `mammal:(cat dog) fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Key: "mammal",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "cat"},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{Value: "dog"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{Value: "fox"},
				},
			},
		},
		{
			name: `mammal:(cat dog) mammal:fox`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Key: "mammal",
						Nodes: []ast.Node{
							&ast.StringNode{Value: "cat"},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.StringNode{Value: "dog"},
						},
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{Key: "mammal", Value: "fox"},
				},
			},
		},
		{
			name: `title:((Advanced OR Search OR Query) -"Advanced Search Query")`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.GroupNode{
						Key: "title",
						Nodes: []ast.Node{
							&ast.GroupNode{
								Nodes: []ast.Node{
									&ast.StringNode{Value: "Advanced"},
									&ast.OperatorNode{Value: kql.BoolOR},
									&ast.StringNode{Value: "Search"},
									&ast.OperatorNode{Value: kql.BoolOR},
									&ast.StringNode{Value: "Query"},
								},
							},
							&ast.OperatorNode{Value: kql.BoolAND},
							&ast.OperatorNode{Value: kql.BoolNOT},
							&ast.StringNode{Value: "Advanced Search Query"},
						},
					},
				},
			},
		},
		{
			name: "ids",
			query: join([]string{
				`id:b27d3bf1-b254-459f-92e8-bdba668d6d3f$d0648459-25fb-4ed8-8684-bc62c7dca29c!d0648459-25fb-4ed8-8684-bc62c7dca29c`,
				`ID:b27d3bf1-b254-459f-92e8-bdba668d6d3f$d0648459-25fb-4ed8-8684-bc62c7dca29c!d0648459-25fb-4ed8-8684-bc62c7dca29c`,
			}),
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{
						Key:   "id",
						Value: "b27d3bf1-b254-459f-92e8-bdba668d6d3f$d0648459-25fb-4ed8-8684-bc62c7dca29c!d0648459-25fb-4ed8-8684-bc62c7dca29c",
					},
					&ast.OperatorNode{Value: kql.BoolOR},
					&ast.StringNode{
						Key:   "ID",
						Value: "b27d3bf1-b254-459f-92e8-bdba668d6d3f$d0648459-25fb-4ed8-8684-bc62c7dca29c!d0648459-25fb-4ed8-8684-bc62c7dca29c",
					},
				},
			},
		},
		{
			name: `"test:test" test:"test:test" "more:*+#!/°^§$%&&/()=?<><<more" more:"more:*+#!/°^§$%&&/()=?<><<more"`,
			ast: &ast.Ast{
				Nodes: []ast.Node{
					&ast.StringNode{
						Value: "test:test",
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{
						Key:   "test",
						Value: "test:test",
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{
						Value: "more:*+#!/°^§$%&&/()=?<><<more",
					},
					&ast.OperatorNode{Value: kql.BoolAND},
					&ast.StringNode{
						Key:   "more",
						Value: "more:*+#!/°^§$%&&/()=?<><<more",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testKQL(t, tc)
		})
	}
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		if _, err := kql.Parse("", []byte(strings.Join(FullDictionary, " "))); err != nil {
			b.Fatal(err)
		}
	}
}

type testCase struct {
	name  string
	query string
	ast   *ast.Ast
	error error
	skip  bool
	patch func() func()
}

var mustParseTime = func(t *testing.T, ts string) time.Time {
	tp, err := now.Parse(ts)
	if err != nil {
		t.Fatalf("time.Parse(...) error = %v", err)
	}

	return tp
}

var setWorldClock = func(t *testing.T, ts string) func() func() {
	return func() func() {
		kql.PatchTimeNow(func() time.Time {
			return mustParseTime(t, ts)
		})

		return func() {
			kql.PatchTimeNow(time.Now)
		}
	}
}

var patchNow = func(t *testing.T, ts string) func() func() {
	return func() func() {
		kql.PatchTimeNow(func() time.Time {
			return mustParseTime(t, ts)
		})

		return func() {
			kql.PatchTimeNow(time.Now)
		}
	}
}

var join = func(v []string) string {
	return strings.Join(v, " ")
}

func testKQL(t *testing.T, tc testCase) {
	if tc.skip {
		t.Skip()
	}

	if tc.patch != nil {
		revert := tc.patch()
		defer revert()
	}

	query := tc.name
	if tc.query != "" {
		query = tc.query
	}

	astResult, err := kql.Builder{}.Build(query)
	assert := tAssert.New(t)

	if tc.error != nil {
		if expectedError := tc.error.Error(); expectedError != "" {
			assert.Equal(err.Error(), expectedError)
		} else {
			assert.NotNil(err)
		}

		return
	}

	if diff := test.DiffAst(tc.ast, astResult); diff != "" {
		t.Fatalf("AST mismatch \nquery: '%s' \n(-expected +got): %s", query, diff)
	}
}

var FullDictionary = []string{
	`federated search`,
	`federat* search`,
	`search fed*`,
	`author:"John Smith"`,
	`filetype:docx`,
	`filename:budget.xlsx`,
	`author: "John Smith"`,
	`author :"John Smith"`,
	`author : "John Smith"`,
	`author "John Smith"`,
	`author "John Smith"`,
	`author:Shakespear`,
	`author:Paul`,
	`author:Shakesp*`,
	`title:"Advanced Search"`,
	`title:"Advanced Sear*"`,
	`title:"Advan* Search"`,
	`title:"*anced Search"`,
	`author:"John Smith" OR author:"Jane Smith"`,
	`author:"John Smith" AND filetype:docx`,
	`author:("John Smith" "Jane Smith")`,
	`author:("John Smith" OR "Jane Smith")`,
	`(DepartmentId:* OR RelatedHubSites:*) AND contentclass:sts_site NOT IsHubSite:false`,
	`author:"John Smith" (filetype:docx title:"Advanced Search")`,
}
