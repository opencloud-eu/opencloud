package svc

import (
	"context"
	"strings"

	"github.com/CiscoM31/godata"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

const (
	appRoleID          = "appRoleId"
	appRoleAssignments = "appRoleAssignments"
)

func invalidFilterError() error {
	return godata.BadRequestError("invalid filter")
}

func unsupportedFilterError() error {
	return godata.NotImplementedError("unsupported filter")
}

func (g Graph) applyUserFilter(ctx context.Context, req *godata.GoDataRequest, root *godata.ParseNode) (users []*libregraph.User, err error) {
	logger := g.logger.SubloggerWithRequestID(ctx)

	if root == nil {
		root = req.Query.Filter.Tree
	}

	switch root.Token.Type {
	case godata.ExpressionTokenLambdaNav:
		return g.applyFilterLambda(ctx, req, root.Children)
	case godata.ExpressionTokenLogical:
		return g.applyFilterLogical(ctx, req, root)
	case godata.ExpressionTokenFunc:
		return g.applyFilterFunction(ctx, req, root)
	}
	logger.Debug().Str("filter", req.Query.Filter.RawValue).Msg("filter is not supported")
	return users, unsupportedFilterError()
}

func (g Graph) applyFilterFunction(ctx context.Context, req *godata.GoDataRequest, root *godata.ParseNode) (users []*libregraph.User, err error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if root.Token.Type != godata.ExpressionTokenFunc {
		return users, invalidFilterError()
	}

	switch root.Token.Value {
	case "startswith":
		// 'startswith' needs 2 operands
		if len(root.Children) != 2 {
			return users, invalidFilterError()
		}
		return g.applyFilterFunctionStartsWith(ctx, req, root.Children[0], root.Children[1]), nil
	case "contains":
		// 'contains' needs 2 operands
		if len(root.Children) != 2 {
			return users, invalidFilterError()
		}
		return g.applyFilterFunctionContains(ctx, req, root.Children[0], root.Children[1]), nil
	}
	logger.Debug().Str("Token", root.Token.Value).Msg("unsupported function filter")
	return users, unsupportedFilterError()
}

func (g Graph) applyFilterFunctionStartsWith(ctx context.Context, req *godata.GoDataRequest, operand1 *godata.ParseNode, operand2 *godata.ParseNode) (users []*libregraph.User) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if operand1.Token.Type != godata.ExpressionTokenLiteral {
		logger.Debug().Str("Token", operand1.Token.Value).Msg("unsupported function filter")
		return users
	}

	switch operand1.Token.Value {
	case "displayName":
		var retUsers []*libregraph.User
		filterValue := strings.Trim(operand2.Token.Value, "'")
		logger.Debug().Str("property", operand2.Token.Value).Str("value", filterValue).Msg("Filtering displayName by startsWith")
		if users, err := g.identityBackend.GetUsers(ctx, req); err == nil {
			for _, user := range users {
				if strings.HasPrefix(strings.ToLower(user.GetDisplayName()), strings.ToLower(filterValue)) {
					retUsers = append(retUsers, user)
				}
			}
		}
		return retUsers
	default:
		logger.Warn().Str("Token", operand1.Token.Value).Msg("unsupported function filter")
		return nil
	}
}

func (g Graph) applyFilterFunctionContains(ctx context.Context, req *godata.GoDataRequest, operand1 *godata.ParseNode, operand2 *godata.ParseNode) (users []*libregraph.User) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if operand1.Token.Type != godata.ExpressionTokenLiteral {
		logger.Debug().Str("Token", operand1.Token.Value).Msg("unsupported function filter")
		return users
	}

	switch operand1.Token.Value {
	case "displayName":
		var retUsers []*libregraph.User
		filterValue := strings.Trim(operand2.Token.Value, "'")
		logger.Debug().Str("property", operand2.Token.Value).Str("value", filterValue).Msg("Filtering displayName by contains")
		if users, err := g.identityBackend.GetUsers(ctx, req); err == nil {
			for _, user := range users {
				if strings.Contains(strings.ToLower(user.GetDisplayName()), strings.ToLower(filterValue)) {
					retUsers = append(retUsers, user)
				}
			}
		}
		return retUsers
	default:
		logger.Warn().Str("Token", operand1.Token.Value).Msg("unsupported function filter")
		return nil
	}
}

func (g Graph) applyFilterLogical(ctx context.Context, req *godata.GoDataRequest, root *godata.ParseNode) (users []*libregraph.User, err error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if root.Token.Type != godata.ExpressionTokenLogical {
		return users, invalidFilterError()
	}

	// As we currently don't suppport the 'has' or 'in' operator, all our
	// currently supported user filters of the ExpressionTokenLogical type
	// require exactly two operands.
	if len(root.Children) != 2 {
		return users, invalidFilterError()
	}
	switch root.Token.Value {
	case "and":
		return g.applyFilterLogicalAnd(ctx, req, root.Children[0], root.Children[1])
	case "or":
		return g.applyFilterLogicalOr(ctx, req, root.Children[0], root.Children[1])
	case "eq":
		return g.applyFilterEq(ctx, req, root.Children[0], root.Children[1])
	case "le":
		return g.applyFilterLessOrEqual(ctx, req, root)
	}
	logger.Debug().Str("Token", root.Token.Value).Msg("unsupported logical filter")
	return users, unsupportedFilterError()
}

func (g Graph) applyFilterLogicalAnd(ctx context.Context, req *godata.GoDataRequest, operand1 *godata.ParseNode, operand2 *godata.ParseNode) (users []*libregraph.User, err error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	var res1, res2 []*libregraph.User

	// The appRoleAssignmentFilter requires a full list of user, get try to avoid a full user listing and
	// process the other part of the filter first, if that is an appRoleAssignmentFilter as well, we have
	// to bite the bullet and get a full user listing anyway
	if matched, property, value := g.isAppRoleAssignmentFilter(ctx, operand1); matched {
		logger.Debug().Str("property", property).Str("value", value).Msg("delay processing of approleAssignments filter")
		if property != appRoleID {
			return users, unsupportedFilterError()
		}
		res2, err = g.applyUserFilter(ctx, req, operand2)
		if err != nil {
			return []*libregraph.User{}, err
		}
		logger.Debug().Str("property", property).Str("value", value).Msg("applying approleAssignments filter on result set")
		return g.filterUsersByAppRoleID(ctx, req, value, res2)
	}

	// 1st part is no appRoleAssignmentFilter, run the filter query
	res1, err = g.applyUserFilter(ctx, req, operand1)
	if err != nil {
		return []*libregraph.User{}, err
	}

	// Now check 2nd part for appRoleAssignmentFilter and apply that using the result return from the first
	// filter
	if matched, property, value := g.isAppRoleAssignmentFilter(ctx, operand2); matched {
		if property != appRoleID {
			return users, unsupportedFilterError()
		}
		logger.Debug().Str("property", property).Str("value", value).Msg("applying approleAssignments filter on result set")
		return g.filterUsersByAppRoleID(ctx, req, value, res1)
	}

	// 2nd part is no appRoleAssignmentFilter either
	res2, err = g.applyUserFilter(ctx, req, operand2)
	if err != nil {
		return []*libregraph.User{}, err
	}

	// We now have two slice with results of the subfilters. Now turn one of them
	// into a map for efficiently getting the intersection of both slices
	userSet := userSliceToMap(res1)
	var filteredUsers []*libregraph.User
	for _, user := range res2 {
		if _, found := userSet[user.GetId()]; found {
			filteredUsers = append(filteredUsers, user)
		}
	}
	return filteredUsers, nil
}

func (g Graph) applyFilterLogicalOr(ctx context.Context, req *godata.GoDataRequest, operand1 *godata.ParseNode, operand2 *godata.ParseNode) (users []*libregraph.User, err error) {
	//	logger := g.logger.SubloggerWithRequestID(ctx)
	var res1, res2 []*libregraph.User

	res1, err = g.applyUserFilter(ctx, req, operand1)
	if err != nil {
		return []*libregraph.User{}, err
	}

	res2, err = g.applyUserFilter(ctx, req, operand2)
	if err != nil {
		return []*libregraph.User{}, err
	}

	// We now have two slices with results of the subfilters. Now turn one of them
	// into a map for efficiently getting the union of both slices
	userSet := userSliceToMap(res1)
	filteredUsers := make([]*libregraph.User, 0, len(res1)+len(res2))
	filteredUsers = append(filteredUsers, res1...)
	for _, user := range res2 {
		if _, found := userSet[user.GetId()]; !found {
			filteredUsers = append(filteredUsers, user)
		}
	}
	return filteredUsers, nil
}

func (g Graph) applyFilterEq(ctx context.Context, req *godata.GoDataRequest, operand1 *godata.ParseNode, operand2 *godata.ParseNode) (users []*libregraph.User, err error) {
	// We only support the 'eq' on 'userType' for now
	switch {
	case operand1.Token.Type != godata.ExpressionTokenLiteral:
		fallthrough
	case operand1.Token.Value != "userType":
		fallthrough
	case operand2.Token.Type != godata.ExpressionTokenString:
		return users, unsupportedFilterError()
	}

	// unquote
	value := strings.Trim(operand2.Token.Value, "'")
	switch value {
	case "Member", "Guest":
		return g.identityBackend.GetUsers(ctx, req)
	case "Federated":
		return g.searchOCMAcceptedUsers(ctx, req)
	}
	return users, unsupportedFilterError()
}

func (g Graph) applyFilterLessOrEqual(ctx context.Context, req *godata.GoDataRequest, filterRoot *godata.ParseNode) (users []*libregraph.User, err error) {
	// Assume that our identity backend is able to apply this filter
	return g.identityBackend.FilterUsers(ctx, req, filterRoot)
}

func (g Graph) applyFilterLambda(ctx context.Context, req *godata.GoDataRequest, nodes []*godata.ParseNode) (users []*libregraph.User, err error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if len(nodes) != 2 {
		return users, invalidFilterError()
	}
	// We only support the "any" operator for lambda queries for now
	if nodes[1].Token.Type != godata.ExpressionTokenLambda || nodes[1].Token.Value != "any" {
		logger.Debug().Str("Token", nodes[1].Token.Value).Msg("unsupported lambda filter")
		return users, unsupportedFilterError()
	}
	if nodes[0].Token.Type != godata.ExpressionTokenLiteral {
		return users, unsupportedFilterError()
	}
	switch nodes[0].Token.Value {
	case "memberOf":
		return g.applyLambdaMemberOfAny(ctx, req, nodes[1].Children)
	case appRoleAssignments:
		return g.applyLambdaAppRoleAssignmentAny(ctx, req, nodes[1].Children)
	}
	logger.Debug().Str("Token", nodes[0].Token.Value).Msg("unsupported relation for lambda filter")
	return users, unsupportedFilterError()
}

func (g Graph) applyLambdaMemberOfAny(ctx context.Context, req *godata.GoDataRequest, nodes []*godata.ParseNode) (users []*libregraph.User, err error) {
	if len(nodes) != 2 {
		return users, invalidFilterError()
	}

	// First element is the "name" of the lambda function's parameter
	if nodes[0].Token.Type != godata.ExpressionTokenLiteral {
		return users, invalidFilterError()
	}

	// We only support the 'eq' expression for now
	if nodes[1].Token.Type != godata.ExpressionTokenLogical || nodes[1].Token.Value != "eq" {
		return users, unsupportedFilterError()
	}
	return g.applyMemberOfEq(ctx, req, nodes[1].Children)
}

func (g Graph) applyMemberOfEq(ctx context.Context, req *godata.GoDataRequest, nodes []*godata.ParseNode) (users []*libregraph.User, err error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if len(nodes) != 2 {
		return users, invalidFilterError()
	}

	if nodes[0].Token.Type != godata.ExpressionTokenNav {
		return users, invalidFilterError()
	}

	if len(nodes[0].Children) != 2 {
		return users, invalidFilterError()
	}

	switch nodes[0].Children[1].Token.Value {
	case "id":
		filterValue, err := g.getUUIDTokenValue(ctx, nodes[1])
		if err != nil {
			return users, unsupportedFilterError()
		}

		logger.Debug().Str("property", nodes[0].Children[1].Token.Value).Str("value", filterValue).Msg("Filtering memberOf by group id")
		return g.identityBackend.GetGroupMembers(ctx, filterValue, req)
	default:
		return users, unsupportedFilterError()
	}
}

func (g Graph) applyLambdaAppRoleAssignmentAny(ctx context.Context, req *godata.GoDataRequest, nodes []*godata.ParseNode) (users []*libregraph.User, err error) {
	if len(nodes) != 2 {
		return users, invalidFilterError()
	}

	// First element is the "name" of the lambda function's parameter
	if nodes[0].Token.Type != godata.ExpressionTokenLiteral {
		return users, invalidFilterError()
	}

	// We only support the 'eq' expression for now
	if nodes[1].Token.Type != godata.ExpressionTokenLogical || nodes[1].Token.Value != "eq" {
		return users, unsupportedFilterError()
	}
	return g.applyAppRoleAssignmentEq(ctx, req, nodes[1].Children)
}

func (g Graph) applyAppRoleAssignmentEq(ctx context.Context, req *godata.GoDataRequest, nodes []*godata.ParseNode) (users []*libregraph.User, err error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if len(nodes) != 2 {
		return users, invalidFilterError()
	}

	if nodes[0].Token.Type != godata.ExpressionTokenNav {
		return users, invalidFilterError()
	}

	if len(nodes[0].Children) != 2 {
		return users, invalidFilterError()
	}

	if nodes[0].Children[1].Token.Value == appRoleID {
		filterValue, err := g.getUUIDTokenValue(ctx, nodes[1])
		if err != nil {
			return users, unsupportedFilterError()
		}

		logger.Debug().Str("property", nodes[0].Children[1].Token.Value).Str("value", filterValue).Msg("Filtering appRoleAssignments by appRoleId")
		if users, err = g.identityBackend.GetUsers(ctx, req); err != nil {
			return users, err
		}

		return g.filterUsersByAppRoleID(ctx, req, filterValue, users)
	}
	return users, unsupportedFilterError()
}

func (g Graph) filterUsersByAppRoleID(ctx context.Context, req *godata.GoDataRequest, id string, users []*libregraph.User) ([]*libregraph.User, error) {
	// We're using a map for the results here, in order to avoid returning
	// a user twice. The settings API, still has an issue that causes it to
	// duplicate some assignments on restart:
	// https://github.com/owncloud/ocis/issues/3432
	usersByIdMap := make(map[string]*libregraph.User, len(users))
	for _, user := range users {
		usersByIdMap[user.GetId()] = user
	}

	var expand bool
	if exp := req.Query.GetExpand(); exp != nil {
		for _, item := range exp.ExpandItems {
			if item.Path[0].Value == appRoleAssignments {
				expand = true
				break
			}
		}
	}

	assignmentsForRole, err := g.roleService.ListRoleAssignmentsFiltered(
		ctx,
		&settingssvc.ListRoleAssignmentsFilteredRequest{
			Filters: []*settingsmsg.UserRoleAssignmentFilter{
				{
					Type: settingsmsg.UserRoleAssignmentFilter_TYPE_ROLE,
					Term: &settingsmsg.UserRoleAssignmentFilter_RoleId{RoleId: id},
				},
			},
		},
	)
	if err != nil {
		return users, err
	}
	resultUsers := make([]*libregraph.User, 0, len(assignmentsForRole.GetAssignments()))
	for _, assignment := range assignmentsForRole.GetAssignments() {
		if user, ok := usersByIdMap[assignment.GetAccountUuid()]; ok {
			if expand {
				user.AppRoleAssignments = []libregraph.AppRoleAssignment{g.assignmentToAppRoleAssignment(assignment)}
			}
			resultUsers = append(resultUsers, user)
		}
	}
	return resultUsers, nil
}

func (g Graph) getUUIDTokenValue(ctx context.Context, node *godata.ParseNode) (string, error) {
	var value string
	switch node.Token.Type {
	case godata.ExpressionTokenGuid:
		value = node.Token.Value
	case godata.ExpressionTokenString:
		// unquote
		value = strings.Trim(node.Token.Value, "'")
	default:
		return "", unsupportedFilterError()
	}
	return value, nil
}

func (g Graph) isAppRoleAssignmentFilter(ctx context.Context, node *godata.ParseNode) (match bool, property string, filter string) {
	if node.Token.Type != godata.ExpressionTokenLambdaNav {
		return false, "", ""
	}

	if len(node.Children) != 2 {
		return false, "", ""
	}

	if node.Children[0].Token.Type != godata.ExpressionTokenLiteral || node.Children[0].Token.Value != appRoleAssignments {
		return false, "", ""
	}

	if node.Children[1].Token.Type != godata.ExpressionTokenLambda || node.Children[1].Token.Value != "any" {
		return false, "", ""
	}

	if len(node.Children[1].Children) != 2 {
		return false, "", ""
	}
	lambdaParam := node.Children[1].Children
	// We only support the 'eq' expression for now
	if lambdaParam[1].Token.Type != godata.ExpressionTokenLogical || lambdaParam[1].Token.Value != "eq" {
		return false, "", ""
	}

	if len(lambdaParam[1].Children) != 2 {
		return false, "", ""
	}
	expression := lambdaParam[1].Children
	if expression[0].Token.Type != godata.ExpressionTokenNav || expression[0].Token.Value != "/" {
		return false, "", ""
	}

	if len(expression[0].Children) != 2 {
		return false, "", ""
	}
	if expression[0].Children[1].Token.Value != appRoleID {
		return false, "", ""
	}
	filterValue, err := g.getUUIDTokenValue(ctx, expression[1])
	if err != nil {
		return false, "", ""
	}
	return true, appRoleID, filterValue
}

func userSliceToMap(users []*libregraph.User) map[string]*libregraph.User {
	resMap := make(map[string]*libregraph.User, len(users))
	for _, user := range users {
		resMap[user.GetId()] = user
	}
	return resMap
}
