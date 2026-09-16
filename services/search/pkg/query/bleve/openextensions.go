package bleve

import (
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	bleveQuery "github.com/blevesearch/bleve/v2/search/query"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	searchQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

var time0 time.Time // open end of a date range

func openExtensionStringQuery(n *ast.StringNode) bleveQuery.Query {
	if strings.ContainsAny(n.Value, "*?") {
		sibling, value := mapping.SiblingLower, strings.ToLower(n.Value)
		if n.Exact {
			sibling, value = mapping.SiblingKeyword, n.Value
		}
		wq := bleveQuery.NewWildcardQuery(value)
		wq.SetField(searchQuery.OpenExtensionField(n.Key, sibling))
		return wq
	}

	inclusive := true
	plan := searchQuery.OpenExtensionEquality(n.Key, n.Value, n.Exact)
	alternatives := make([]bleveQuery.Query, 0, len(plan))
	for _, term := range plan {
		var q bleveQuery.FieldableQuery
		switch term.Sibling {
		case mapping.SiblingNumber:
			f := term.Number
			q = bleveQuery.NewNumericRangeInclusiveQuery(&f, &f, &inclusive, &inclusive)
		case mapping.SiblingBool:
			q = bleveQuery.NewBoolFieldQuery(term.Bool)
		case mapping.SiblingDate:
			q = bleveQuery.NewDateRangeInclusiveQuery(term.Time, term.Time, &inclusive, &inclusive)
		default:
			q = bleveQuery.NewTermQuery(term.String)
		}
		q.SetField(term.Field)
		alternatives = append(alternatives, q)
	}
	if len(alternatives) == 1 {
		return alternatives[0]
	}
	return bleve.NewDisjunctionQuery(alternatives...)
}

// openExtensionDateQuery: an equality is the one-instant range.
func openExtensionDateQuery(n *ast.DateTimeNode) bleveQuery.Query {
	inclusive, exclusive := true, false
	field := searchQuery.OpenExtensionField(n.Key, mapping.SiblingDate)
	var q *bleveQuery.DateRangeQuery
	switch n.Operator.Value {
	case ">":
		q = bleveQuery.NewDateRangeInclusiveQuery(n.Value, time0, &exclusive, nil)
	case ">=":
		q = bleveQuery.NewDateRangeInclusiveQuery(n.Value, time0, &inclusive, nil)
	case "<":
		q = bleveQuery.NewDateRangeInclusiveQuery(time0, n.Value, nil, &exclusive)
	case "<=":
		q = bleveQuery.NewDateRangeInclusiveQuery(time0, n.Value, nil, &inclusive)
	default:
		q = bleveQuery.NewDateRangeInclusiveQuery(n.Value, n.Value, &inclusive, &inclusive)
	}
	q.SetField(field)
	return q
}
