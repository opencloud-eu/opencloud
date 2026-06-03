package jmap

var NS_CALENDAR_EVENTS = ns(JmapCalendars)

type CalendarEventSearchResults SearchResultsTemplate[CalendarEvent]

var _ SearchResults[CalendarEvent] = &CalendarEventSearchResults{}

func (r *CalendarEventSearchResults) GetResults() []CalendarEvent { return r.Results }
func (r *CalendarEventSearchResults) GetCanCalculateChanges() ChangeCalculation {
	return r.CanCalculateChanges
}
func (r *CalendarEventSearchResults) GetPosition() *uint         { return r.Position }
func (r *CalendarEventSearchResults) GetLimit() *uint            { return r.Limit }
func (r *CalendarEventSearchResults) GetTotal() *uint            { return r.Total }
func (r *CalendarEventSearchResults) RemoveResults()             { r.Results = nil }
func (r *CalendarEventSearchResults) SetLimit(limit *uint)       { r.Limit = limit }
func (r *CalendarEventSearchResults) SetPosition(position *uint) { r.Position = position }

func (j *Client) GetCalendarEvents(accountId string, eventIds []string, ctx Context) (Result[CalendarEventGetResponse], error) {
	return get(j, "GetCalendarEvents", CalendarEventType,
		func(accountId string, ids []string) CalendarEventGetCommand {
			return CalendarEventGetCommand{AccountId: accountId, Ids: eventIds}
		},
		CalendarEventGetResponse{},
		identity1,
		accountId, eventIds,
		ctx,
	)
}

func (j *Client) QueryCalendarEvents(accountIds map[string]QueryParams, limit *uint, //NOSONAR
	filter CalendarEventFilterElement, sortBy []CalendarEventComparator, calculateTotal bool,
	ctx Context) (Result[map[string]*CalendarEventSearchResults], error) {
	return queryN(j, "QueryCalendarEvents", CalendarEventType,
		[]CalendarEventComparator{{Property: CalendarEventPropertyStart, IsAscending: false}},
		func(accountId string, queryParams QueryParams, limit *uint, filter CalendarEventFilterElement, sortBy []CalendarEventComparator) CalendarEventQueryCommand {
			return CalendarEventQueryCommand{AccountId: accountId, Filter: filter, Sort: sortBy, Position: queryParams.Position, Anchor: queryParams.Anchor, AnchorOffset: queryParams.AnchorOffset, Limit: limit, CalculateTotal: calculateTotal}
		},
		func(accountId string, cmd Command, path string, rof string) CalendarEventGetRefCommand {
			return CalendarEventGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: cmd, Path: path, ResultOf: rof}}
		},
		func(query CalendarEventQueryResponse, queryParams QueryParams, limit *uint) *CalendarEventSearchResults {
			return &CalendarEventSearchResults{
				Results:             []CalendarEvent{},
				CanCalculateChanges: ChangeCalculation(query.CanCalculateChanges),
				Position:            valueIf(query.Position, queryParams.Anchor == ""),
				Total:               valueIf(query.Total, calculateTotal),
				Limit:               valueIf(query.Limit, limit != nil),
			}
		},
		func(query CalendarEventQueryResponse, get CalendarEventGetResponse, queryParams QueryParams, limit *uint) *CalendarEventSearchResults {
			return &CalendarEventSearchResults{
				Results:             get.List,
				CanCalculateChanges: ChangeCalculation(query.CanCalculateChanges),
				Position:            valueIf(query.Position, queryParams.Anchor == ""),
				Total:               valueIf(query.Total, calculateTotal),
				Limit:               valueIf(query.Limit, limit != nil),
			}
		},
		accountIds, limit,
		filter, sortBy, ctx,
	)
}

type CalendarEventChanges ChangesTemplate[CalendarEvent]

var _ Changes[CalendarEvent] = CalendarEventChanges{}

func (c CalendarEventChanges) GetHasMoreChanges() bool     { return c.HasMoreChanges }
func (c CalendarEventChanges) GetOldState() State          { return c.OldState }
func (c CalendarEventChanges) GetNewState() State          { return c.NewState }
func (c CalendarEventChanges) GetCreated() []CalendarEvent { return c.Created }
func (c CalendarEventChanges) GetUpdated() []CalendarEvent { return c.Updated }
func (c CalendarEventChanges) GetDestroyed() []string      { return c.Destroyed }

// Retrieve the changes in Calendar Events since a given State.
// @api:tags event,changes
func (j *Client) GetCalendarEventChanges(accountId string, sinceState State, maxChanges uint,
	ctx Context) (Result[CalendarEventChanges], error) {
	return changes(j, "GetCalendarEventChanges", CalendarEventType,
		func() CalendarEventChangesCommand {
			return CalendarEventChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: uintPtr(maxChanges)}
		},
		CalendarEventChangesResponse{},
		func(path string, rof string) CalendarEventGetRefCommand {
			return CalendarEventGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandCalendarEventChanges,
					Path:     path,
					ResultOf: rof,
				},
			}
		},
		func(resp CalendarEventGetResponse) []CalendarEvent { return resp.List },
		func(oldState, newState State, hasMoreChanges bool, created, updated []CalendarEvent, destroyed []string) CalendarEventChanges {
			return CalendarEventChanges{
				OldState:       oldState,
				NewState:       newState,
				HasMoreChanges: hasMoreChanges,
				Created:        created,
				Updated:        updated,
				Destroyed:      destroyed,
			}
		},
		ctx,
	)
}

func (j *Client) CreateCalendarEvent(accountId string, event CalendarEventChange, ctx Context) (Result[*CalendarEvent], error) {
	return create(j, "CreateCalendarEvent", CalendarEventType,
		func(accountId string, create map[string]CalendarEventChange) CalendarEventSetCommand {
			return CalendarEventSetCommand{AccountId: accountId, Create: create}
		},
		func(accountId string, ref string) CalendarEventGetCommand {
			return CalendarEventGetCommand{AccountId: accountId, Ids: []string{ref}}
		},
		func(resp CalendarEventSetResponse) map[string]*CalendarEvent {
			return resp.Created
		},
		func(resp CalendarEventGetResponse) []CalendarEvent {
			return resp.List
		},
		accountId, event,
		ctx,
	)
}

func (j *Client) DeleteCalendarEvent(accountId string, destroyIds []string, ctx Context) (Result[map[string]SetError], error) {
	return destroy(j, "DeleteCalendarEvent", CalendarEventType,
		func(accountId string, destroy []string) CalendarEventSetCommand {
			return CalendarEventSetCommand{AccountId: accountId, Destroy: destroy}
		},
		CalendarEventSetResponse{},
		accountId, destroyIds,
		ctx,
	)
}

func (j *Client) UpdateCalendarEvent(accountId string, id string, changes CalendarEventChange, ctx Context) (Result[CalendarEvent], error) {
	return update(j, "UpdateCalendarEvent", CalendarEventType,
		func(update map[string]PatchObject) CalendarEventSetCommand {
			return CalendarEventSetCommand{AccountId: accountId, Update: update}
		},
		func(id string) CalendarEventGetCommand {
			return CalendarEventGetCommand{AccountId: accountId, Ids: []string{id}}
		},
		func(resp CalendarEventSetResponse) map[string]SetError { return resp.NotUpdated },
		func(resp CalendarEventGetResponse) CalendarEvent { return resp.List[0] },
		id, changes,
		ctx,
	)
}
