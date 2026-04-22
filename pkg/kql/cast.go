package kql

import (
	"fmt"
	"time"

	"github.com/jinzhu/now"
	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

func toNode[T ast.Node](in any) (T, error) {
	var t T
	out, ok := in.(T)
	if !ok {
		return t, fmt.Errorf("can't convert '%T' to '%T'", in, t)
	}

	return out, nil
}

func toNodes[T ast.Node](in any) ([]T, error) {
	switch v := in.(type) {
	case T:
		return []T{v}, nil
	case []T:
		return v, nil
	case []*ast.OperatorNode, []*ast.DateTimeNode:
		return toNodes[T](v)
	case []any:
		var nodes []T
		for _, el := range v {
			node, err := toNodes[T](el)
			if err != nil {
				return nil, err
			}

			nodes = append(nodes, node...)
		}
		return nodes, nil
	case nil:
		return nil, nil
	default:
		var t T
		return nil, fmt.Errorf("can't convert '%T' to '%T'", in, t)
	}
}

func toString(in any) (string, error) {
	switch v := in.(type) {
	case []byte:
		return string(v), nil
	case []any:
		var str string

		for i := range v {
			sv, err := toString(v[i])
			if err != nil {
				return "", err
			}

			str += sv
		}

		return str, nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("can't convert '%T' to string", v)
	}
}

func toTime(in any) (time.Time, error) {
	ts, err := toString(in)
	if err != nil {
		return time.Time{}, err
	}

	return now.Parse(ts)
}

func toTimeRange(in any) (*time.Time, *time.Time, error) {
	var from, to time.Time

	value, err := toString(in)
	if err != nil {
		return &from, &to, &query.UnsupportedTimeRangeError{}
	}

	c := &now.Config{
		WeekStartDay: time.Monday,
	}

	n := c.With(timeNow())

	switch value {
	case "today":
		from = n.BeginningOfDay()
		to = n.EndOfDay()
	case "yesterday":
		yesterday := n.With(n.AddDate(0, 0, -1))
		from = yesterday.BeginningOfDay()
		to = yesterday.EndOfDay()
	case "this week":
		from = n.BeginningOfWeek()
		to = n.EndOfWeek()
	case "last week":
		lastWeek := n.With(n.AddDate(0, 0, -7))
		from = lastWeek.BeginningOfWeek()
		to = lastWeek.EndOfWeek()
	case "last 7 days":
		from = n.With(n.AddDate(0, 0, -6)).BeginningOfDay()
		to = n.EndOfDay()
	case "this month":
		from = n.BeginningOfMonth()
		to = n.EndOfMonth()
	case "last month":
		lastMonth := n.With(n.BeginningOfMonth().AddDate(0, 0, -1))
		from = lastMonth.BeginningOfMonth()
		to = lastMonth.EndOfMonth()
	case "last 30 days":
		from = n.With(n.AddDate(0, 0, -29)).BeginningOfDay()
		to = n.EndOfDay()
	case "this year":
		from = n.BeginningOfYear()
		to = n.EndOfYear()
	case "last year":
		lastYear := n.With(n.AddDate(-1, 0, 0))
		from = lastYear.BeginningOfYear()
		to = lastYear.EndOfYear()
	}

	if from.IsZero() || to.IsZero() {
		return nil, nil, &query.UnsupportedTimeRangeError{}
	}

	return &from, &to, nil
}
