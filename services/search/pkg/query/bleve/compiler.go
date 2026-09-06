package bleve

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/geo"
	bleveQuery "github.com/blevesearch/bleve/v2/search/query"
	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	searchQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

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

// geoQueryField resolves a KQL geo key to its indexed geopoint field, erroring
// when the key is not a geopoint field.
func geoQueryField(key string) (string, error) {
	field, ok := searchQuery.ResolveGeoField(key)
	if !ok {
		return "", &kql.GeoFieldError{Key: key}
	}
	return field, nil
}

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
			// hidden takes bool words only; anything else matches nothing
			if n.Key == "Hidden" {
				var q bleveQuery.Query
				if b, err := strconv.ParseBool(n.Value); err == nil {
					bq := bleveQuery.NewBoolFieldQuery(b)
					bq.SetField(n.Key)
					q = bq
				} else {
					q = bleveQuery.NewMatchNoneQuery()
				}
				if prev == nil {
					prev = q
				} else {
					next = q
				}
				break
			}

			// keys are resolved and media-type expanded by normalize. MimeType
			// skips the escaper so the category wildcards (image/*) keep their `*`;
			// bleve treats `/` and `+` as literals mid-term, so a literal MIME like
			// image/svg+xml still matches exactly.
			val := n.Value
			if searchQuery.FieldIsPath(n.Key) {
				val = strings.TrimSuffix(val, "/")
			}
			k := n.Key
			v := val
			if k != "ID" && k != "Size" && k != "MimeType" {
				v = bleveEscaper.Replace(val)
			}
			if n.CaseInsensitive {
				k += mapping.LowercaseSuffix
				v = strings.ToLower(v)
				val = strings.ToLower(val)
			}

			isWildcard := strings.ContainsAny(val, "*?")

			// a word-broken field matches the value as a phrase of its words on the
			// _words sibling (a quoted query string term is a match phrase query
			// run through the field's analyzer); wildcards stay on _lowercase.
			// A fulltext field is its own words field, the phrase runs on it.
			if searchQuery.FieldIsWordBroken(n.Key) && !isWildcard && !n.Exact {
				k, v = n.Key+mapping.WordsSuffix, `"`+strings.ReplaceAll(val, `"`, `\"`)+`"`
			} else if searchQuery.FieldIsFulltext(n.Key) && !isWildcard && !n.Exact {
				v = `"` + strings.ReplaceAll(val, `"`, `\"`) + `"`
			}

			var q bleveQuery.Query = bleveQuery.NewQueryStringQuery(k + ":" + v)
			switch {
			case n.Exact && !isWildcard:
				// = matches the whole value, on the lowercased sibling for
				// case-insensitive fields
				tq := bleveQuery.NewTermQuery(val)
				tq.SetField(k)
				q = tq
			case isWildcard && searchQuery.FieldIsWordBroken(n.Key) && !strings.HasSuffix(val, "*"):
				// a wildcard on a word-broken field forgives a missing extension:
				// *report also matches Report.txt
				bq := bleve.NewBooleanQuery()
				bq.AddShould(
					bleveQuery.NewQueryStringQuery(k+":"+v),
					bleveQuery.NewQueryStringQuery(k+":"+v+".*"),
				)
				bq.SetMinShould(1)
				q = bq
			}
			if searchQuery.FieldIsPath(n.Key) {
				// bleve has no path hierarchy analyzer, unlike OpenSearch: match the
				// folder itself and its descendants (`\/*`). A BooleanQuery keeps
				// this atomic; a DisjunctionQuery would be redistributed by an
				// enclosing AND (mapBinary treats a left disjunction as an OR-chain).
				bq := bleve.NewBooleanQuery()
				bq.AddShould(q, bleveQuery.NewQueryStringQuery(k+":"+v+`\/*`))
				bq.SetMinShould(1)
				q = bq
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
				FieldVal:       n.Key,
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
		case *ast.NumberNode:
			var q bleveQuery.Query
			if field := n.Key; slices.Contains([]string{"Size", "Type"}, field) {
				q = numberRange(field, n.Operator, n.Value)
			} else {
				// same answer as the OpenSearch backend: unknown numeric keys
				// match nothing instead of querying an arbitrary field
				q = bleveQuery.NewMatchNoneQuery()
			}
			if q == nil {
				continue
			}

			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.BooleanNode:
			q := bleveQuery.NewBoolFieldQuery(n.Value)
			q.SetField(n.Key)
			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.GroupNode:
			// keys resolved and grouping property propagated in normalize
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
		case *ast.GeoDistanceNode:
			field, err := geoQueryField(n.Key)
			if err != nil {
				return nil, 0, err
			}
			q := bleveQuery.NewGeoDistanceQuery(n.Lon, n.Lat, strconv.FormatFloat(n.Radius, 'f', -1, 64)+"m")
			q.SetField(field)
			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.GeoBoundingBoxNode:
			field, err := geoQueryField(n.Key)
			if err != nil {
				return nil, 0, err
			}
			// bleve takes the top-left and bottom-right corners.
			q := bleveQuery.NewGeoBoundingBoxQuery(n.MinLon, n.MaxLat, n.MaxLon, n.MinLat)
			q.SetField(field)
			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.GeoPolygonNode:
			field, err := geoQueryField(n.Key)
			if err != nil {
				return nil, 0, err
			}
			points := make([]geo.Point, 0, len(n.Points))
			for _, p := range n.Points {
				points = append(points, geo.Point{Lon: p.Lon, Lat: p.Lat})
			}
			q := bleveQuery.NewGeoBoundingPolygonQuery(points)
			q.SetField(field)
			if prev == nil {
				prev = q
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
					// unary at the beginning: the term was consumed into the
					// MustNot via nextNode, so clear next, otherwise a following
					// operator would bind the stale term (NOT x AND y drops y).
					prev = q
					next = nil
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
		// keys are resolved and group keys propagated by normalize
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

func numberRange(field string, operator *ast.OperatorNode, value float64) bleveQuery.Query {
	if operator == nil {
		return nil
	}

	inclusive, exclusive := true, false

	var q *bleveQuery.NumericRangeQuery
	switch operator.Value {
	case ">":
		q = bleveQuery.NewNumericRangeInclusiveQuery(&value, nil, &exclusive, nil)
	case ">=":
		q = bleveQuery.NewNumericRangeInclusiveQuery(&value, nil, &inclusive, nil)
	case "<":
		q = bleveQuery.NewNumericRangeInclusiveQuery(nil, &value, nil, &exclusive)
	case "<=":
		q = bleveQuery.NewNumericRangeInclusiveQuery(nil, &value, nil, &inclusive)
	default:
		return nil
	}

	q.SetField(field)

	return q
}
