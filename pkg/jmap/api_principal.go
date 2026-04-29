package jmap

var NS_PRINCIPALS = ns(JmapPrincipals)

func (j *Client) GetPrincipals(accountId string, ids []string, ctx Context) (Result[PrincipalGetResponse], Error) {
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

func (j *Client) QueryPrincipals(accountId string, //NOSONAR
	filter PrincipalFilterElement, sortBy []PrincipalComparator,
	position int, anchor string, anchorOffset *int, limit *uint, calculateTotal bool,
	ctx Context) (Result[*PrincipalSearchResults], Error) {
	return query(j, "QueryPrincipals", PrincipalType,
		[]PrincipalComparator{{Property: PrincipalPropertyName, IsAscending: true}},
		func(filter PrincipalFilterElement, sortBy []PrincipalComparator, position int, anchor string, anchorOffset *int, limit *uint) PrincipalQueryCommand {
			return PrincipalQueryCommand{AccountId: accountId, Filter: filter, Sort: sortBy, Position: position, Anchor: anchor, AnchorOffset: anchorOffset, Limit: limit, CalculateTotal: calculateTotal}
		},
		func(cmd Command, path string, rof string) PrincipalGetRefCommand {
			return PrincipalGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: cmd, Path: path, ResultOf: rof}}
		},
		func(query PrincipalQueryResponse, get PrincipalGetResponse) *PrincipalSearchResults {
			return &PrincipalSearchResults{
				Results:             get.List,
				CanCalculateChanges: ChangeCalculation(query.CanCalculateChanges),
				Position:            ptrIf(query.Position, anchor == ""),
				Total:               ptrIf(query.Total, calculateTotal),
				Limit:               ptrIf(query.Limit, limit != nil),
			}
		},
		filter, sortBy, position, anchor, anchorOffset, limit, ctx,
	)
}
