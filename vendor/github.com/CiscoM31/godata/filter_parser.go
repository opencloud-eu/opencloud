package godata

import "context"

var GlobalFilterTokenizer *Tokenizer
var GlobalFilterParser *ExpressionParser

// ParseFilterString converts an input string from the $filter part of the URL into a parse
// tree that can be used by providers to create a response.
func ParseFilterString(ctx context.Context, filter string) (*GoDataFilterQuery, error) {
	tokens, err := GlobalFilterTokenizer.Tokenize(ctx, filter)
	if err != nil {
		return nil, err
	}
	// TODO: can we do this in one fell swoop?
	postfix, err := GlobalFilterParser.InfixToPostfix(ctx, tokens)
	if err != nil {
		return nil, err
	}
	tree, err := GlobalFilterParser.PostfixToTree(ctx, postfix)
	if err != nil {
		return nil, err
	}
	if tree == nil || tree.Token == nil || !GlobalFilterParser.isBooleanExpression(tree.Token) {
		return nil, BadRequestError("Value must be a boolean expression")
	}
	return &GoDataFilterQuery{tree, filter}, nil
}

func SemanticizeFilterQuery(
	filter *GoDataFilterQuery,
	service *GoDataService,
	entity *GoDataEntityType,
) error {

	if filter == nil || filter.Tree == nil {
		return nil
	}

	var semanticizeFilterNode func(node *ParseNode) error
	semanticizeFilterNode = func(node *ParseNode) error {

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
			err := semanticizeFilterNode(child)
			if err != nil {
				return err
			}
		}

		return nil
	}

	return semanticizeFilterNode(filter.Tree)
}
