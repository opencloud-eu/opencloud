package jmap

var NS_PRINCIPALS = ns(JmapPrincipals)

func (j *Client) GetPrincipals(accountId string, ids []string, ctx Context) (Result[PrincipalGetResponse], error) {
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

var _ SearchResults[Principal] = &PrincipalSearchResults{}

func (r *PrincipalSearchResults) GetResults() []Principal { return r.Results }
func (r *PrincipalSearchResults) GetCanCalculateChanges() ChangeCalculation {
	return r.CanCalculateChanges
}
func (r *PrincipalSearchResults) GetPosition() *uint         { return r.Position }
func (r *PrincipalSearchResults) GetLimit() *uint            { return r.Limit }
func (r *PrincipalSearchResults) GetTotal() *uint            { return r.Total }
func (r *PrincipalSearchResults) RemoveResults()             { r.Results = nil }
func (r *PrincipalSearchResults) SetLimit(limit *uint)       { r.Limit = limit }
func (r *PrincipalSearchResults) SetPosition(position *uint) { r.Position = position }

func (j *Client) QueryPrincipals(accountIds map[string]QueryParams, limit *uint, //NOSONAR
	filter PrincipalFilterElement, sortBy []PrincipalComparator, calculateTotal bool,
	ctx Context) (Result[map[string]*PrincipalSearchResults], error) {
	return queryN(j, "QueryPrincipals", PrincipalType,
		[]PrincipalComparator{{Property: PrincipalPropertyName, IsAscending: true}},
		func(accountId string, p QueryParams, limit *uint, filter PrincipalFilterElement, sortBy []PrincipalComparator) PrincipalQueryCommand {
			return PrincipalQueryCommand{AccountId: accountId, Filter: filter, Sort: sortBy, Position: p.Position, Anchor: p.Anchor, AnchorOffset: p.AnchorOffset, Limit: limit, CalculateTotal: calculateTotal}
		},
		func(accountId string, cmd Command, path, rof string) PrincipalGetRefCommand {
			return PrincipalGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: cmd, Path: path, ResultOf: rof}}
		},
		func(query PrincipalQueryResponse, queryParams QueryParams, limit *uint) *PrincipalSearchResults {
			return &PrincipalSearchResults{
				Results:             []Principal{},
				CanCalculateChanges: ChangeCalculation(query.CanCalculateChanges),
				Position:            valueIf(query.Position, queryParams.Anchor == ""),
				Total:               ptrIf(query.Total, calculateTotal),
				Limit:               valueIf(query.Limit, limit != nil),
			}
		},
		func(query PrincipalQueryResponse, get PrincipalGetResponse, queryParams QueryParams, limit *uint) *PrincipalSearchResults {
			return &PrincipalSearchResults{
				Results:             get.List,
				CanCalculateChanges: ChangeCalculation(query.CanCalculateChanges),
				Position:            valueIf(query.Position, queryParams.Anchor == ""),
				Total:               ptrIf(query.Total, calculateTotal),
				Limit:               valueIf(query.Limit, limit != nil),
			}
		},
		accountIds, limit, filter, sortBy,
		ctx,
	)
}
