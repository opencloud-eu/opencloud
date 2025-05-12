package godata

import (
	"context"
	"strings"
)

// tokenDurationRe is a regex for a token of type duration.
// The token value is set to the ISO 8601 string inside the single quotes
// For example, if the input data is duration'PT2H', then the token value is set to PT2H without quotes.
const tokenDurationRe = `^(duration)?'(?P<subtoken>-?P((([0-9]+Y([0-9]+M)?([0-9]+D)?|([0-9]+M)([0-9]+D)?|([0-9]+D))(T(([0-9]+H)([0-9]+M)?([0-9]+(\.[0-9]+)?S)?|([0-9]+M)([0-9]+(\.[0-9]+)?S)?|([0-9]+(\.[0-9]+)?S)))?)|(T(([0-9]+H)([0-9]+M)?([0-9]+(\.[0-9]+)?S)?|([0-9]+M)([0-9]+(\.[0-9]+)?S)?|([0-9]+(\.[0-9]+)?S)))))'`

// Addressing properties.
// Addressing items within a collection:
//   ABNF: entityColNavigationProperty [ collectionNavigation ]
//         collectionNavigation = [ "/" qualifiedEntityTypeName ] [ collectionNavPath ]
//   Description: OData identifier, optionally followed by collection navigation.
//
// propertyPath = entityColNavigationProperty [ collectionNavigation ]
//             / entityNavigationProperty    [ singleNavigation ]
//             / complexColProperty          [ collectionPath ]
//             / complexProperty             [ complexPath ]
//             / primitiveColProperty        [ collectionPath ]
//             / primitiveProperty           [ singlePath ]
//             / streamProperty              [ boundOperation ]

type ExpressionTokenType int

func (e ExpressionTokenType) Value() int {
	return (int)(e)
}

const (
	ExpressionTokenOpenParen        ExpressionTokenType = iota // Open parenthesis - parenthesis expression, list expression, or path segment selector.
	ExpressionTokenCloseParen                                  // Close parenthesis
	ExpressionTokenWhitespace                                  // white space token
	ExpressionTokenNav                                         // Property navigation
	ExpressionTokenColon                                       // Function arg separator for 'any(v:boolExpr)' and 'all(v:boolExpr)' lambda operators
	ExpressionTokenComma                                       // [5] List delimiter and function argument delimiter.
	ExpressionTokenLogical                                     // eq|ne|gt|ge|lt|le|and|or|not|has|in
	ExpressionTokenOp                                          // add|sub|mul|divby|div|mod
	ExpressionTokenFunc                                        // Function, e.g. contains, substring...
	ExpressionTokenLambdaNav                                   // "/" token when used in lambda expression, e.g. tags/any()
	ExpressionTokenLambda                                      // [10] any(), all() lambda functions
	ExpressionTokenCase                                        // A case() statement. See https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_case
	ExpressionTokenCasePair                                    // A case statement expression pair [ <boolean expression> : <value expression> ]
	ExpressionTokenNull                                        //
	ExpressionTokenIt                                          // The '$it' token
	ExpressionTokenRoot                                        // [15] The '$root' token
	ExpressionTokenFloat                                       // A floating point value.
	ExpressionTokenInteger                                     // An integer value
	ExpressionTokenString                                      // SQUOTE *( SQUOTE-in-string / pchar-no-SQUOTE ) SQUOTE
	ExpressionTokenDate                                        // A date value
	ExpressionTokenTime                                        // [20] A time value
	ExpressionTokenDateTime                                    // A date-time value
	ExpressionTokenBoolean                                     // A literal boolean value
	ExpressionTokenLiteral                                     // A literal non-boolean value
	ExpressionTokenDuration                                    // duration      = [ "duration" ] SQUOTE durationValue SQUOTE
	ExpressionTokenGuid                                        // [25] A 128-bit GUID
	ExpressionTokenAssignement                                 // The '=' assignement for function arguments.
	ExpressionTokenGeographyPolygon                            // A polygon with geodetic (ie spherical) coordinates. Parsed Token.Value is '<long> <lat>,<long> <lat>...'
	ExpressionTokenGeometryPolygon                             // A polygon with planar (ie cartesian) coordinates. Parsed Token.Value is '<long> <lat>,<long> <lat>...'
	ExpressionTokenGeographyPoint                              // A geodetic coordinate point. Parsed Token.Value is '<long> <lat>'
	expressionTokenLast
)

func (e ExpressionTokenType) String() string {
	return [...]string{
		"ExpressionTokenOpenParen",
		"ExpressionTokenCloseParen",
		"ExpressionTokenWhitespace",
		"ExpressionTokenNav",
		"ExpressionTokenColon",
		"ExpressionTokenComma",
		"ExpressionTokenLogical",
		"ExpressionTokenOp",
		"ExpressionTokenFunc",
		"ExpressionTokenLambdaNav",
		"ExpressionTokenLambda",
		"ExpressionTokenCase",
		"ExpressionTokenCasePair",
		"ExpressionTokenNull",
		"ExpressionTokenIt",
		"ExpressionTokenRoot",
		"ExpressionTokenFloat",
		"ExpressionTokenInteger",
		"ExpressionTokenString",
		"ExpressionTokenDate",
		"ExpressionTokenTime",
		"ExpressionTokenDateTime",
		"ExpressionTokenBoolean",
		"ExpressionTokenLiteral",
		"ExpressionTokenDuration",
		"ExpressionTokenGuid",
		"ExpressionTokenAssignement",
		"ExpressionTokenGeographyPolygon",
		"ExpressionTokenGeometryPolygon",
		"ExpressionTokenGeographyPoint",
		"expressionTokenLast",
	}[e]
}

// ExpressionParser is a ODATA expression parser.
type ExpressionParser struct {
	*Parser
	ExpectBoolExpr bool       // Request expression to validate it is a boolean expression.
	tokenizer      *Tokenizer // The expression tokenizer.
}

// ParseExpressionString converts a ODATA expression input string into a parse
// tree that can be used by providers to create a response.
// Expressions can be used within $filter and $orderby query options.
func (p *ExpressionParser) ParseExpressionString(ctx context.Context, expression string) (*GoDataExpression, error) {
	tokens, err := p.tokenizer.Tokenize(ctx, expression)
	if err != nil {
		return nil, err
	}
	// TODO: can we do this in one fell swoop?
	postfix, err := p.InfixToPostfix(ctx, tokens)
	if err != nil {
		return nil, err
	}
	tree, err := p.PostfixToTree(ctx, postfix)
	if err != nil {
		return nil, err
	}
	if tree == nil || tree.Token == nil {
		return nil, BadRequestError("Expression cannot be nil")
	}
	if p.ExpectBoolExpr && !p.isBooleanExpression(tree.Token) {
		return nil, BadRequestError("Expression does not return a boolean value")
	}
	return &GoDataExpression{tree, expression}, nil
}

var GlobalExpressionTokenizer *Tokenizer
var GlobalExpressionParser *ExpressionParser

// init constructs single instances of Tokenizer and ExpressionParser and initializes their
// respective packages variables.
func init() {
	p := NewExpressionParser()
	t := p.tokenizer // use the Tokenizer instance created by

	GlobalExpressionTokenizer = t
	GlobalExpressionParser = p

	GlobalFilterTokenizer = t
	GlobalFilterParser = p
}

// ExpressionTokenizer creates a tokenizer capable of tokenizing ODATA expressions.
// 4.01 Services MUST support case-insensitive operator names.
// See https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#_Toc31360955
func NewExpressionTokenizer() *Tokenizer {
	t := Tokenizer{}
	// guidValue = 8HEXDIG "-" 4HEXDIG "-" 4HEXDIG "-" 4HEXDIG "-" 12HEXDIG
	t.Add(`^[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12}`, ExpressionTokenGuid)
	// duration      = [ "duration" ] SQUOTE durationValue SQUOTE
	// durationValue = [ SIGN ] "P" [ 1*DIGIT "D" ] [ "T" [ 1*DIGIT "H" ] [ 1*DIGIT "M" ] [ 1*DIGIT [ "." 1*DIGIT ] "S" ] ]
	// Duration literals in OData 4.0 required prefixing with “duration”.
	// In OData 4.01, services MUST support duration and enumeration literals with or without the type prefix.
	// OData clients that want to operate across OData 4.0 and OData 4.01 services should always include the prefix for duration and enumeration types.
	t.Add(tokenDurationRe, ExpressionTokenDuration)
	t.Add("^[0-9]{4,4}-[0-9]{2,2}-[0-9]{2,2}T[0-9]{2,2}:[0-9]{2,2}(:[0-9]{2,2}(.[0-9]+)?)?(Z|[+-][0-9]{2,2}:[0-9]{2,2})", ExpressionTokenDateTime)
	t.Add("^-?[0-9]{4,4}-[0-9]{2,2}-[0-9]{2,2}", ExpressionTokenDate)
	t.Add("^[0-9]{2,2}:[0-9]{2,2}(:[0-9]{2,2}(.[0-9]+)?)?", ExpressionTokenTime)
	t.Add("^\\(", ExpressionTokenOpenParen)
	t.Add("^\\)", ExpressionTokenCloseParen)
	t.Add("^(?P<token>/)(?i)(any|all)", ExpressionTokenLambdaNav)                              // '/' as a token between a collection expression and a lambda function any() or all()
	t.Add("^/", ExpressionTokenNav)                                                            // '/' as a token for property navigation.
	t.Add("^=", ExpressionTokenAssignement)                                                    // '=' as a token for function argument assignment.
	t.AddWithSubstituteFunc("^:", ExpressionTokenColon, func(in string) string { return "," }) // Function arg separator for lambda functions (any, all)
	t.Add("^,", ExpressionTokenComma)                                                          // Default arg separator for functions
	// Per ODATA ABNF grammar, functions must be followed by a open parenthesis.
	// This implementation is a bit more lenient and allows space character between
	// the function name and the open parenthesis.
	// TODO: If we remove the optional space character, the function token will be
	// mistakenly interpreted as a literal.
	// E.g. ABNF for 'geo.distance':
	// distanceMethodCallExpr   = "geo.distance"   OPEN BWS commonExpr BWS COMMA BWS commonExpr BWS CLOSE
	t.Add("(?i)^(?P<token>(geo.distance|geo.intersects|geo.length))[\\s(]", ExpressionTokenFunc)
	// Example: geography'POLYGON((-122.031577 47.578581, -122.031577 47.678581, -122.131577 47.678581))'
	t.Add(`(?i)^geography'(?:SRID=(\d{1,5});)?POLYGON\s*\(\(\s*(?P<subtoken>-?\d+(\.\d+)?\s+-?\d+(\.\d+)?(?:\s*,\s*-?\d+(\.\d+)?\s+-?\d+(\.\d+)?)*?)\s*\)\)'`, ExpressionTokenGeographyPolygon)
	t.Add(`(?i)^geometry'(?:SRID=(\d{1,5});)?POLYGON\s*\(\(\s*(?P<subtoken>-?\d+(\.\d+)?\s+-?\d+(\.\d+)?(?:\s*,\s*-?\d+(\.\d+)?\s+-?\d+(\.\d+)?)*?)\s*\)\)'`, ExpressionTokenGeometryPolygon)
	// Example: geography'POINT(-122.131577 47.678581)'
	t.Add(`(?i)^geography'POINT\s*\(\s*(?P<subtoken>-?\d+(\.\d+)?\s+-?\d+(\.\d+)?)\s*\)'`, ExpressionTokenGeographyPoint)
	// According to ODATA ABNF notation, functions must be followed by a open parenthesis with no space
	// between the function name and the open parenthesis.
	// However, we are leniently allowing space characters between the function and the open parenthesis.
	// TODO make leniency configurable.
	// E.g. ABNF for 'indexof':
	// indexOfMethodCallExpr    = "indexof"    OPEN BWS commonExpr BWS COMMA BWS commonExpr BWS CLOSE
	t.Add("(?i)^(?P<token>(substringof|substring|length|indexof|exists|"+
		"contains|endswith|startswith|tolower|toupper|trim|concat|year|month|day|"+
		"hour|minute|second|fractionalseconds|date|time|totaloffsetminutes|now|"+
		"maxdatetime|mindatetime|totalseconds|round|floor|ceiling|isof|cast))[\\s(]", ExpressionTokenFunc)
	// Logical operators must be followed by a space character.
	// However, in practice user have written requests such as not(City eq 'Seattle')
	// We are leniently allowing space characters between the operator name and the open parenthesis.
	// TODO make leniency configurable.
	// Example:
	// notExpr = "not" RWS boolCommonExpr
	t.Add("(?i)^(?P<token>(eq|ne|gt|ge|lt|le|and|or|not|has|in))[\\s(]", ExpressionTokenLogical)
	// Arithmetic operators must be followed by a space character.
	t.Add("(?i)^(?P<token>(add|sub|mul|divby|div|mod))\\s", ExpressionTokenOp)
	// anyExpr = "any" OPEN BWS [ lambdaVariableExpr BWS COLON BWS lambdaPredicateExpr ] BWS CLOSE
	// allExpr = "all" OPEN BWS   lambdaVariableExpr BWS COLON BWS lambdaPredicateExpr   BWS CLOSE
	t.Add("(?i)^(?P<token>(any|all))[\\s(]", ExpressionTokenLambda)
	t.Add("(?i)^(?P<token>(case))[\\s(]", ExpressionTokenCase)
	t.Add("^null", ExpressionTokenNull)
	t.Add("^\\$it", ExpressionTokenIt)
	t.Add("^\\$root", ExpressionTokenRoot)
	t.Add("^-?[0-9]+\\.[0-9]+", ExpressionTokenFloat)
	t.Add("^-?[0-9]+", ExpressionTokenInteger)
	t.AddWithSubstituteFunc("^'(''|[^'])*'", ExpressionTokenString, unescapeTokenString)
	t.Add("^(true|false)", ExpressionTokenBoolean)
	t.AddWithSubstituteFunc("^@*[a-zA-Z][a-zA-Z0-9_.]*",
		ExpressionTokenLiteral, unescapeUtfEncoding) // The optional '@' character is used to identify parameter aliases
	t.Ignore("^ ", ExpressionTokenWhitespace)

	return &t
}

// unescapeTokenString unescapes the input string according to the ODATA ABNF rules
// and returns the unescaped string.
// In ODATA ABNF, strings are encoded according to the following rules:
// string           = SQUOTE *( SQUOTE-in-string / pchar-no-SQUOTE ) SQUOTE
// SQUOTE-in-string = SQUOTE SQUOTE ; two consecutive single quotes represent one within a string literal
// pchar-no-SQUOTE       = unreserved / pct-encoded-no-SQUOTE / other-delims / "$" / "&" / "=" / ":" / "@"
// pct-encoded-no-SQUOTE = "%" ( "0" / "1" /   "3" / "4" / "5" / "6" / "8" / "9" / A-to-F ) HEXDIG
// / "%" "2" ( "0" / "1" / "2" / "3" / "4" / "5" / "6" /   "8" / "9" / A-to-F )
// unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
//
// See http://docs.oasis-open.org/odata/odata/v4.01/csprd03/abnf/odata-abnf-construction-rules.txt
func unescapeTokenString(in string) string {
	// The call to ReplaceAll() implements
	// SQUOTE-in-string = SQUOTE SQUOTE ; two consecutive single quotes represent one within a string literal
	if in == "''" {
		return in
	}
	return strings.ReplaceAll(in, "''", "'")
}

// TODO: should we make this configurable?
func unescapeUtfEncoding(in string) string {
	return strings.ReplaceAll(in, "_x0020_", " ")
}

func NewExpressionParser() *ExpressionParser {
	parser := &ExpressionParser{
		Parser:         EmptyParser().WithLiteralToken(ExpressionTokenLiteral),
		ExpectBoolExpr: false,
		tokenizer:      NewExpressionTokenizer(),
	}
	parser.DefineOperator("/", 2, OpAssociationLeft, 8) // Note: '/' is used as a property navigator and between a collExpr and lambda function.
	parser.DefineOperator("has", 2, OpAssociationLeft, 8)
	// 'in' operator takes a literal list.
	// City in ('Seattle') needs to be interpreted as a list expression, not a paren expression.
	parser.DefineOperator("in", 2, OpAssociationLeft, 8).WithListExprPreference(true)
	parser.DefineOperator("-", 1, OpAssociationNone, 7)
	parser.DefineOperator("not", 1, OpAssociationRight, 7)
	parser.DefineOperator("cast", 2, OpAssociationNone, 7)
	parser.DefineOperator("mul", 2, OpAssociationNone, 6)
	parser.DefineOperator("div", 2, OpAssociationNone, 6)   // Division
	parser.DefineOperator("divby", 2, OpAssociationNone, 6) // Decimal Division
	parser.DefineOperator("mod", 2, OpAssociationNone, 6)
	parser.DefineOperator("add", 2, OpAssociationNone, 5)
	parser.DefineOperator("sub", 2, OpAssociationNone, 5)
	parser.DefineOperator("gt", 2, OpAssociationLeft, 4)
	parser.DefineOperator("ge", 2, OpAssociationLeft, 4)
	parser.DefineOperator("lt", 2, OpAssociationLeft, 4)
	parser.DefineOperator("le", 2, OpAssociationLeft, 4)
	parser.DefineOperator("eq", 2, OpAssociationLeft, 3)
	parser.DefineOperator("ne", 2, OpAssociationLeft, 3)
	parser.DefineOperator("and", 2, OpAssociationLeft, 2)
	parser.DefineOperator("or", 2, OpAssociationLeft, 1)
	parser.DefineOperator("=", 2, OpAssociationRight, 0) // Function argument assignment. E.g. MyFunc(Arg1='abc')
	parser.DefineFunction("contains", []int{2}, true)
	parser.DefineFunction("endswith", []int{2}, true)
	parser.DefineFunction("startswith", []int{2}, true)
	parser.DefineFunction("exists", []int{2}, true)
	parser.DefineFunction("length", []int{1}, false)
	parser.DefineFunction("indexof", []int{2}, false)
	parser.DefineFunction("substring", []int{2, 3}, false)
	parser.DefineFunction("substringof", []int{2}, false)
	parser.DefineFunction("tolower", []int{1}, false)
	parser.DefineFunction("toupper", []int{1}, false)
	parser.DefineFunction("trim", []int{1}, false)
	parser.DefineFunction("concat", []int{2}, false)
	parser.DefineFunction("year", []int{1}, false)
	parser.DefineFunction("month", []int{1}, false)
	parser.DefineFunction("day", []int{1}, false)
	parser.DefineFunction("hour", []int{1}, false)
	parser.DefineFunction("minute", []int{1}, false)
	parser.DefineFunction("second", []int{1}, false)
	parser.DefineFunction("fractionalseconds", []int{1}, false)
	parser.DefineFunction("date", []int{1}, false)
	parser.DefineFunction("time", []int{1}, false)
	parser.DefineFunction("totaloffsetminutes", []int{1}, false)
	parser.DefineFunction("now", []int{0}, false)
	parser.DefineFunction("maxdatetime", []int{0}, false)
	parser.DefineFunction("mindatetime", []int{0}, false)
	parser.DefineFunction("totalseconds", []int{1}, false)
	parser.DefineFunction("round", []int{1}, false)
	parser.DefineFunction("floor", []int{1}, false)
	parser.DefineFunction("ceiling", []int{1}, false)
	parser.DefineFunction("isof", []int{1, 2}, true) // isof function can take one or two arguments.
	parser.DefineFunction("cast", []int{2}, false)
	parser.DefineFunction("geo.distance", []int{2}, false)
	// The geo.intersects function has the following signatures:
	//   Edm.Boolean geo.intersects(Edm.GeographyPoint,Edm.GeographyPolygon)
	//   Edm.Boolean geo.intersects(Edm.GeometryPoint,Edm.GeometryPolygon)
	// The geo.intersects function returns true if the specified point lies within the interior
	// or on the boundary of the specified polygon, otherwise it returns false.
	parser.DefineFunction("geo.intersects", []int{2}, true)
	// The geo.length function has the following signatures:
	//   Edm.Double geo.length(Edm.GeographyLineString)
	//   Edm.Double geo.length(Edm.GeometryLineString)
	// The geo.length function returns the total length of its line string parameter
	// in the coordinate reference system signified by its SRID.
	parser.DefineFunction("geo.length", []int{1}, false)
	// 'any' can take either zero or two arguments with the later having the form any(d:d/Prop eq 1).
	// Godata interprets the colon as an argument delimiter and considers the function to have two arguments.
	parser.DefineFunction("any", []int{0, 2}, true)
	// 'all' requires two arguments of a form similar to 'any'.
	parser.DefineFunction("all", []int{2}, true)
	// Define 'case' as a function accepting 1-10 arguments. Each argument is a pair of expressions separated by a colon.
	// See https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_case
	parser.DefineFunction("case", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, true)

	return parser
}

func (p *ExpressionParser) SemanticizeExpression(
	expression *GoDataExpression,
	service *GoDataService,
	entity *GoDataEntityType,
) error {

	if expression == nil || expression.Tree == nil {
		return nil
	}

	var semanticizeExpressionNode func(node *ParseNode) error
	semanticizeExpressionNode = func(node *ParseNode) error {

		if node.Token.Type == ExpressionTokenLiteral {
			prop, ok := service.PropertyLookup[entity][node.Token.Value]
			if !ok {
				return BadRequestError("No property found " + node.Token.Value + " on entity " + entity.Name)
			}
			node.Token.SemanticType = SemanticTypeProperty
			node.Token.SemanticReference = prop
		} else {
			node.Token.SemanticType = SemanticTypePropertyValue
			node.Token.SemanticReference = &node.Token.Value
		}

		for _, child := range node.Children {
			err := semanticizeExpressionNode(child)
			if err != nil {
				return err
			}
		}

		return nil
	}

	return semanticizeExpressionNode(expression.Tree)
}
