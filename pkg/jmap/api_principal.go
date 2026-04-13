package jmap

var NS_PRINCIPALS = ns(JmapPrincipals)

func (j *Client) GetPrincipals(accountId string, ids []string, ctx Context) (PrincipalGetResponse, SessionState, State, Language, Error) {
	return get(j, "GetPrincipals", PrincipalType,
		func(accountId string, ids []string) PrincipalGetCommand {
			return PrincipalGetCommand{AccountId: accountId, Ids: ids}
		},
		PrincipalGetResponse{},
		identity1,
		accountId, ids,
		ctx,
	)
}

type PrincipalSearchResults SearchResultsTemplate[Principal]

var _ SearchResults[Principal] = PrincipalSearchResults{}

func (r PrincipalSearchResults) GetResults() []Principal      { return r.Results }
func (r PrincipalSearchResults) GetCanCalculateChanges() bool { return r.CanCalculateChanges }
func (r PrincipalSearchResults) GetPosition() uint            { return r.Position }
func (r PrincipalSearchResults) GetLimit() uint               { return r.Limit }
func (r PrincipalSearchResults) GetTotal() *uint              { return r.Total }

func (j *Client) QueryPrincipals(accountId string,
	filter PrincipalFilterElement, sortBy []PrincipalComparator,
	position uint, limit uint, calculateTotal bool,
	ctx Context) (PrincipalSearchResults, SessionState, State, Language, Error) {
	return query(j, "QueryPrincipals", PrincipalType,
		[]PrincipalComparator{{Property: PrincipalPropertyName, IsAscending: true}},
		func(filter PrincipalFilterElement, sortBy []PrincipalComparator, position uint, limit uint) PrincipalQueryCommand {
			return PrincipalQueryCommand{AccountId: accountId, Filter: filter, Sort: sortBy, Position: position, Limit: limit, CalculateTotal: calculateTotal}
		},
		func(cmd Command, path string, rof string) PrincipalGetRefCommand {
			return PrincipalGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: cmd, Path: path, ResultOf: rof}}
		},
		func(query PrincipalQueryResponse, get PrincipalGetResponse) PrincipalSearchResults {
			return PrincipalSearchResults{
				Results:             get.List,
				CanCalculateChanges: query.CanCalculateChanges,
				Position:            query.Position,
				Total:               uintPtrIf(query.Total, calculateTotal),
				Limit:               query.Limit,
			}
		},
		filter, sortBy, limit, position, ctx,
	)
}
