package bleve

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve/v2"
	bleveQuery "github.com/blevesearch/bleve/v2/search/query"
	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// lowercaseFields holds the fields whose query-side value is pre-lowercased so
// it matches the index-time lowercasing analyzer. Shared with the OpenSearch
// backend via search.LowercaseValueFields; every other field keeps its casing.
var lowercaseFields = search.LowercaseValueFields()

var _fields = map[string]string{
	"rootid":    "RootID",
	"path":      "Path",
	"id":        "ID",
	"name":      "Name",
	"size":      "Size",
	"mtime":     "Mtime",
	"mediatype": "MimeType",
	"type":      "Type",
	"tag":       "Tags",
	"tags":      "Tags",
	"content":   "Content",
	"hidden":    "Hidden",
	"favorite":  "Favorites",
}

// The following quoted string enumerates the characters which may be escaped: "+-=&|><!(){}[]^\"~*?:\\/ "
// based on bleve docs https://blevesearch.com/docs/Query-String-Query/
// Wildcards * and ? are excluded
var bleveEscaper = strings.NewReplacer(
	`+`, `\+`,
	`-`, `\-`,
	`=`, `\=`,
	`&`, `\&`,
	`|`, `\|`,
	`>`, `\>`,
	`<`, `\<`,
	`!`, `\!`,
	`(`, `\(`,
	`)`, `\)`,
	`{`, `\{`,
	`}`, `\}`,
	`{`, `\}`,
	`[`, `\[`,
	`]`, `\]`,
	`^`, `\^`,
	`"`, `\"`,
	`~`, `\~`,
	`:`, `\:`,
	`\`, `\\`,
	`/`, `\/`,
	` `, `\ `,
)

// Compiler represents a KQL query search string to the bleve query formatter.
type Compiler struct{}

// Compile implements the query formatter which converts the KQL query search string to the bleve query.
func (c Compiler) Compile(givenAst *ast.Ast) (bleveQuery.Query, error) {
	q, err := compile(givenAst)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func compile(a *ast.Ast) (bleveQuery.Query, error) {
	q, _, err := walk(0, a.Nodes)
	if err != nil {
		return nil, err
	}
	switch q.(type) {
	case *bleveQuery.ConjunctionQuery, *bleveQuery.DisjunctionQuery:
		return q, nil
	}
	return bleve.NewConjunctionQuery(q), nil
}

func walk(offset int, nodes []ast.Node) (bleveQuery.Query, int, error) {
	var prev, next bleveQuery.Query
	var operator *ast.OperatorNode
	var isGroup bool
	for i := offset; i < len(nodes); i++ {
		switch n := nodes[i].(type) {
		case *ast.StringNode:
			k := getField(n.Key)
			v := n.Value
			if k != "ID" && k != "Size" {
				v = bleveEscaper.Replace(n.Value)
			}

			if _, ok := lowercaseFields[k]; ok {
				v = strings.ToLower(v)
			}

			var q bleveQuery.Query
			var group bool
			switch k {
			case "MimeType":
				q, group = mimeType(k, v)
				if prev == nil {
					isGroup = group
				}
			default:
				q = bleveQuery.NewQueryStringQuery(k + ":" + v)
			}

			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.DateTimeNode:
			q := &bleveQuery.DateRangeQuery{
				Start:          bleveQuery.BleveQueryTime{},
				End:            bleveQuery.BleveQueryTime{},
				InclusiveStart: nil,
				InclusiveEnd:   nil,
				FieldVal:       getField(n.Key),
			}

			if n.Operator == nil {
				continue
			}

			switch n.Operator.Value {
			case ">":
				q.Start.Time = n.Value
				q.InclusiveStart = &[]bool{false}[0]
			case ">=":
				q.Start.Time = n.Value
				q.InclusiveStart = &[]bool{true}[0]
			case "<":
				q.End.Time = n.Value
				q.InclusiveEnd = &[]bool{false}[0]
			case "<=":
				q.End.Time = n.Value
				q.InclusiveEnd = &[]bool{true}[0]
			default:
				continue
			}

			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.BooleanNode:
			q := bleveQuery.NewQueryStringQuery(getField(n.Key) + fmt.Sprintf(":%v", n.Value))
			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.GroupNode:
			if n.Key != "" {
				n = normalizeGroupingProperty(n)
			}
			q, _, err := walk(0, n.Nodes)
			if err != nil {
				return nil, 0, err
			}
			if prev == nil {
				prev = q
				isGroup = true
			} else {
				next = q
			}
		case *ast.OperatorNode:
			if n.Value == kql.BoolAND || n.Value == kql.BoolOR {
				operator = n
			} else if n.Value == kql.BoolNOT {
				var err error
				next, offset, err = nextNode(i+1, nodes)
				if err != nil {
					return nil, 0, err
				}
				q := bleve.NewBooleanQuery()
				q.AddMustNot(next)
				if prev == nil {
					// unary in the beginning
					prev = q
				} else {
					next = q
				}
			}
		}
		if prev != nil && next != nil && operator != nil {
			prev = mapBinary(operator, prev, next, isGroup)
			isGroup = false
			operator = nil
			next = nil
		}
		if i < offset {
			i = offset
		}
	}
	if prev == nil {
		return nil, 0, fmt.Errorf("can not compile the query")
	}
	return prev, offset, nil
}

func nextNode(offset int, nodes []ast.Node) (bleveQuery.Query, int, error) {
	if n, ok := nodes[offset].(*ast.GroupNode); ok {
		gq, _, err := walk(0, n.Nodes)
		if err != nil {
			return nil, 0, err
		}
		return gq, offset + 1, nil
	}
	if n, ok := nodes[offset].(*ast.OperatorNode); ok {
		if n.Value == kql.BoolNOT {
			return walk(offset, nodes)
		}
	}
	one := nodes[:offset+1]
	return walk(offset, one)
}

func mapBinary(operator *ast.OperatorNode, ln, rn bleveQuery.Query, leftIsGroup bool) bleveQuery.Query {
	if operator.Value == kql.BoolOR {
		right, ok := rn.(*bleveQuery.DisjunctionQuery)
		switch left := ln.(type) {
		case *bleveQuery.DisjunctionQuery:
			if ok {
				left.AddQuery(right.Disjuncts...)
			} else {
				left.AddQuery(rn)
			}
			return left
		case *bleveQuery.ConjunctionQuery:
			return bleveQuery.NewDisjunctionQuery([]bleveQuery.Query{ln, rn})
		default:
			if ok {
				left := bleveQuery.NewDisjunctionQuery([]bleveQuery.Query{ln})
				left.AddQuery(right.Disjuncts...)
				return left
			}
			return bleveQuery.NewDisjunctionQuery([]bleveQuery.Query{ln, rn})
		}
	}
	if operator.Value == kql.BoolAND {
		switch left := ln.(type) {
		case *bleveQuery.ConjunctionQuery:
			left.AddQuery(rn)
			return left
		case *bleveQuery.DisjunctionQuery:
			if !leftIsGroup {
				last := left.Disjuncts[len(left.Disjuncts)-1]
				rn = bleveQuery.NewConjunctionQuery([]bleveQuery.Query{
					last,
					rn,
				})
				dj := bleveQuery.NewDisjunctionQuery(left.Disjuncts[:len(left.Disjuncts)-1])
				dj.AddQuery(rn)
				return dj
			}
		}
	}
	return bleveQuery.NewConjunctionQuery([]bleveQuery.Query{
		ln,
		rn,
	})
}

func getField(name string) string {
	if name == "" {
		return "Name"
	}
	if _, ok := _fields[strings.ToLower(name)]; ok {
		return _fields[strings.ToLower(name)]
	}
	return name
}

func normalizeGroupingProperty(group *ast.GroupNode) *ast.GroupNode {
	for _, n := range group.Nodes {
		if onode, ok := n.(*ast.StringNode); ok {
			onode.Key = group.Key
		}
	}
	return group
}

func mimeType(k, v string) (bleveQuery.Query, bool) {
	switch v {
	case "file":
		q := bleve.NewBooleanQuery()
		q.AddMustNot(bleveQuery.NewQueryStringQuery(k + ":httpd/unix-directory"))
		return q, false
	case "folder":
		return bleveQuery.NewQueryStringQuery(k + ":httpd/unix-directory"), false
	case "document":
		return bleveQuery.NewDisjunctionQuery(newQueryStringQueryList(k,
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.form",
			"application/vnd.oasis.opendocument.text",
			"text/plain",
			"text/markdown",
			"application/rtf",
			"application/vnd.apple.pages",
		)), true
	case "spreadsheet":
		return bleveQuery.NewDisjunctionQuery(newQueryStringQueryList(k,
			"application/vnd.ms-excel",
			"application/vnd.oasis.opendocument.spreadsheet",
			"text/csv",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"application/vnd.oasis.opendocument.spreadsheet",
			"application/vnd.apple.numbers",
		)), true
	case "presentation":
		return bleveQuery.NewDisjunctionQuery(newQueryStringQueryList(k,
			"application/vnd.openxmlformats-officedocument.presentationml.presentation",
			"application/vnd.oasis.opendocument.presentation",
			"application/vnd.ms-powerpoint",
			"application/vnd.apple.keynote",
		)), true
	case "pdf":
		return bleveQuery.NewQueryStringQuery(k + ":application/pdf"), false
	case "image":
		return bleveQuery.NewQueryStringQuery(k + ":image/*"), false
	case "video":
		return bleveQuery.NewQueryStringQuery(k + ":video/*"), false
	case "audio":
		return bleveQuery.NewQueryStringQuery(k + ":audio/*"), false
	case "archive":
		return bleveQuery.NewDisjunctionQuery(newQueryStringQueryList(k,
			"application/zip",
			"application/gzip",
			"application/x-gzip",
			"application/x-7z-compressed",
			"application/x-rar-compressed",
			"application/x-tar",
			"application/x-bzip2",
			"application/x-bzip",
			"application/x-tgz",
		)), true
	default:
		return bleveQuery.NewQueryStringQuery(k + ":" + v), false
	}
}

func newQueryStringQueryList(k string, v ...string) []bleveQuery.Query {
	list := make([]bleveQuery.Query, len(v))
	for i := 0; i < len(v); i++ {
		list[i] = bleveQuery.NewQueryStringQuery(k + ":" + v[i])
	}
	return list
}
