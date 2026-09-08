package kql

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
)

func base(text []byte, pos position) (*ast.Base, error) {
	source, err := toString(text)
	if err != nil {
		return nil, err
	}

	return &ast.Base{
		Loc: &ast.Location{
			Start: ast.Position{
				Line:   pos.line,
				Column: pos.col,
			},
			End: ast.Position{
				Line:   pos.line,
				Column: pos.col + len(text),
			},
			Source: &source,
		},
	}, nil
}

func buildAST(n any, text []byte, pos position) (*ast.Ast, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}

	nodes, err := toNodes[ast.Node](n)
	if err != nil {
		return nil, err
	}

	a := &ast.Ast{
		Base:  b,
		Nodes: connectNodes(DefaultConnector{sameKeyOPValue: BoolOR}, nodes...),
	}

	if err := validateAst(a); err != nil {
		return nil, err
	}

	return a, nil
}

func buildStringNode(k, v any, exact bool, text []byte, pos position) (*ast.StringNode, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}

	key, err := toString(k)
	if err != nil {
		return nil, err
	}

	value, err := toString(v)
	if err != nil {
		return nil, err
	}

	return &ast.StringNode{
		Base:  b,
		Key:   key,
		Value: value,
		Exact: exact,
	}, nil
}

func buildDateTimeNode(k, o, v any, text []byte, pos position) (*ast.DateTimeNode, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}

	operator, err := toNode[*ast.OperatorNode](o)
	if err != nil {
		return nil, err
	}

	key, err := toString(k)
	if err != nil {
		return nil, err
	}

	value, err := toTime(v)
	if err != nil {
		return nil, err
	}

	return &ast.DateTimeNode{
		Base:     b,
		Key:      key,
		Operator: operator,
		Value:    value,
	}, nil
}

func buildNumberNode(k, o, v any, text []byte, pos position) (*ast.NumberNode, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}

	operator, err := toNode[*ast.OperatorNode](o)
	if err != nil {
		return nil, err
	}

	key, err := toString(k)
	if err != nil {
		return nil, err
	}

	value, err := toFloat(v)
	if err != nil {
		return nil, err
	}

	return &ast.NumberNode{
		Base:     b,
		Key:      key,
		Operator: operator,
		Value:    value,
	}, nil
}

func buildNaturalLanguageDateTimeNodes(k, v any, text []byte, pos position) ([]ast.Node, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}

	key, err := toString(k)
	if err != nil {
		return nil, err
	}

	from, to, err := toTimeRange(v)
	if err != nil {
		return nil, err
	}

	return []ast.Node{
		&ast.DateTimeNode{
			Base:     b,
			Value:    *from,
			Key:      key,
			Operator: &ast.OperatorNode{Value: ">="},
		},
		&ast.OperatorNode{Value: BoolAND},
		&ast.DateTimeNode{
			Base:     b,
			Value:    *to,
			Key:      key,
			Operator: &ast.OperatorNode{Value: "<="},
		},
	}, nil

}

func buildBooleanNode(k, v any, text []byte, pos position) (*ast.BooleanNode, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}

	key, err := toString(k)
	if err != nil {
		return nil, err
	}

	value, err := toString(v)
	if err != nil {
		return nil, err
	}

	return &ast.BooleanNode{
		Base:  b,
		Key:   key,
		Value: strings.ToLower(value) == "true",
	}, nil
}

func buildOperatorNode(text []byte, pos position) (*ast.OperatorNode, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}

	value, err := toString(text)
	if err != nil {
		return nil, err
	}

	switch value {
	case "+":
		value = BoolAND
	case "-":
		value = BoolNOT
	}

	return &ast.OperatorNode{
		Base:  b,
		Value: value,
	}, nil
}

func buildGroupNode(k, n any, text []byte, pos position) (*ast.GroupNode, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}

	key, _ := toString(k)

	nodes, err := toNodes[ast.Node](n)
	if err != nil {
		return nil, err
	}

	gn := &ast.GroupNode{
		Base:  b,
		Key:   key,
		Nodes: connectNodes(DefaultConnector{sameKeyOPValue: BoolOR}, nodes...),
	}

	if err := validateGroupNode(gn); err != nil {
		return nil, err
	}

	return gn, nil
}

func buildGeoDistanceNode(k, a any, text []byte, pos position) (*ast.GeoDistanceNode, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}
	key, err := toString(k)
	if err != nil {
		return nil, err
	}
	args, err := toString(a)
	if err != nil {
		return nil, err
	}
	lat, lon, radius, err := parseGeoDistanceArgs(args)
	if err != nil {
		return nil, err
	}
	return &ast.GeoDistanceNode{Base: b, Key: key, Lat: lat, Lon: lon, Radius: radius}, nil
}

func buildGeoBoundingBoxNode(k, a any, text []byte, pos position) (*ast.GeoBoundingBoxNode, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}
	key, err := toString(k)
	if err != nil {
		return nil, err
	}
	args, err := toString(a)
	if err != nil {
		return nil, err
	}
	minLat, minLon, maxLat, maxLon, err := parseGeoBoundingBoxArgs(args)
	if err != nil {
		return nil, err
	}
	// Latitude does not wrap, so the two values just delimit the box. Only bleve
	// strictly requires MinLat <= MaxLat (OpenSearch tolerates the inverted
	// order), but we normalise centrally here for consistency and easier
	// debugging. Longitude order is kept: minLon > maxLon denotes a box crossing
	// the antimeridian, which both backends interpret the same way.
	if minLat > maxLat {
		minLat, maxLat = maxLat, minLat
	}
	return &ast.GeoBoundingBoxNode{Base: b, Key: key, MinLat: minLat, MinLon: minLon, MaxLat: maxLat, MaxLon: maxLon}, nil
}

func buildGeoPolygonNode(k, a any, text []byte, pos position) (*ast.GeoPolygonNode, error) {
	b, err := base(text, pos)
	if err != nil {
		return nil, err
	}
	key, err := toString(k)
	if err != nil {
		return nil, err
	}
	args, err := toString(a)
	if err != nil {
		return nil, err
	}
	points, err := parseGeoPolygonArgs(args)
	if err != nil {
		return nil, err
	}
	return &ast.GeoPolygonNode{Base: b, Key: key, Points: points}, nil
}
