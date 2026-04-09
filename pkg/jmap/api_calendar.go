package jmap

var NS_CALENDARS = ns(JmapCalendars)

func (j *Client) ParseICalendarBlob(accountId string, blobIds []string, ctx Context) (CalendarEventParseResponse, SessionState, State, Language, Error) {
	logger := j.logger("ParseICalendarBlob", ctx)

	parse := CalendarEventParseCommand{AccountId: accountId, BlobIds: blobIds}
	cmd, err := j.request(ctx.WithLogger(logger), NS_CALENDARS,
		invocation(parse, "0"),
	)
	if err != nil {
		return CalendarEventParseResponse{}, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (CalendarEventParseResponse, State, Error) {
		var response CalendarEventParseResponse
		err = retrieveParse(ctx, body, parse, "0", &response)
		if err != nil {
			return CalendarEventParseResponse{}, "", err
		}
		return response, "", nil
	})
}

func (j *Client) GetCalendars(accountId string, ids []string, ctx Context) (CalendarGetResponse, SessionState, State, Language, Error) {
	return get(j, "GetCalendars", NS_CALENDARS,
		func(accountId string, ids []string) CalendarGetCommand {
			return CalendarGetCommand{AccountId: accountId, Ids: ids}
		},
		CalendarGetResponse{},
		identity1,
		accountId, ids,
		ctx,
	)
}

type CalendarChanges = ChangesTemplate[Calendar]

// Retrieve Calendar changes since a given state.
// @apidoc calendar,changes
func (j *Client) GetCalendarChanges(accountId string, sinceState State, maxChanges uint, ctx Context) (CalendarChanges, SessionState, State, Language, Error) {
	return changes(j, "GetCalendarChanges", NS_CALENDARS,
		func() CalendarChangesCommand {
			return CalendarChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: uintPtr(maxChanges)}
		},
		CalendarChangesResponse{},
		func(path string, rof string) CalendarGetRefCommand {
			return CalendarGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandCalendarChanges,
					Path:     path,
					ResultOf: rof,
				},
			}
		},
		func(resp CalendarGetResponse) []Calendar { return resp.List },
		func(oldState, newState State, hasMoreChanges bool, created, updated []Calendar, destroyed []string) CalendarChanges {
			return CalendarChanges{
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

type CalendarEventSearchResults SearchResultsTemplate[CalendarEvent]

var _ SearchResults[CalendarEvent] = CalendarEventSearchResults{}

func (r CalendarEventSearchResults) GetResults() []CalendarEvent  { return r.Results }
func (r CalendarEventSearchResults) GetCanCalculateChanges() bool { return r.CanCalculateChanges }
func (r CalendarEventSearchResults) GetPosition() uint            { return r.Position }
func (r CalendarEventSearchResults) GetLimit() uint               { return r.Limit }
func (r CalendarEventSearchResults) GetTotal() *uint              { return r.Total }

func (j *Client) QueryCalendarEvents(accountIds []string, //NOSONAR
	filter CalendarEventFilterElement, sortBy []CalendarEventComparator,
	position int, limit uint, calculateTotal bool,
	ctx Context) (map[string]CalendarEventSearchResults, SessionState, State, Language, Error) {
	return queryN(j, "QueryCalendarEvents", NS_CALENDARS,
		[]CalendarEventComparator{{Property: CalendarEventPropertyStart, IsAscending: false}},
		func(accountId string, filter CalendarEventFilterElement, sortBy []CalendarEventComparator, position int, limit uint) CalendarEventQueryCommand {
			return CalendarEventQueryCommand{AccountId: accountId, Filter: filter, Sort: sortBy, Position: position, Limit: uintPtr(limit), CalculateTotal: calculateTotal}
		},
		func(accountId string, cmd Command, path string, rof string) CalendarEventGetRefCommand {
			return CalendarEventGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: cmd, Path: path, ResultOf: rof}}
		},
		func(query CalendarEventQueryResponse, get CalendarEventGetResponse) CalendarEventSearchResults {
			return CalendarEventSearchResults{
				Results:             get.List,
				CanCalculateChanges: query.CanCalculateChanges,
				Position:            query.Position,
				Total:               uintPtrIf(query.Total, calculateTotal),
				Limit:               query.Limit,
			}
		},
		accountIds,
		filter, sortBy, limit, position, ctx,
	)
}

type CalendarEventChanges = ChangesTemplate[CalendarEvent]

// Retrieve the changes in Calendar Events since a given State.
// @api:tags event,changes
func (j *Client) GetCalendarEventChanges(accountId string, sinceState State, maxChanges uint,
	ctx Context) (CalendarEventChanges, SessionState, State, Language, Error) {
	return changes(j, "GetCalendarEventChanges", NS_CALENDARS,
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

func (j *Client) CreateCalendarEvent(accountId string, event CalendarEvent, ctx Context) (*CalendarEvent, SessionState, State, Language, Error) {
	return create(j, "CreateCalendarEvent", NS_CALENDARS,
		func(accountId string, create map[string]CalendarEvent) CalendarEventSetCommand {
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

func (j *Client) DeleteCalendarEvent(accountId string, destroyIds []string, ctx Context) (map[string]SetError, SessionState, State, Language, Error) {
	return destroy(j, "DeleteCalendarEvent", NS_CALENDARS,
		func(accountId string, destroy []string) CalendarEventSetCommand {
			return CalendarEventSetCommand{AccountId: accountId, Destroy: destroy}
		},
		CalendarEventSetResponse{},
		accountId, destroyIds,
		ctx,
	)
}

func (j *Client) CreateCalendar(accountId string, calendar CalendarChange, ctx Context) (*Calendar, SessionState, State, Language, Error) {
	return create(j, "CreateCalendar", NS_CALENDARS,
		func(accountId string, create map[string]CalendarChange) CalendarSetCommand {
			return CalendarSetCommand{AccountId: accountId, Create: create}
		},
		func(accountId string, ref string) CalendarGetCommand {
			return CalendarGetCommand{AccountId: accountId, Ids: []string{ref}}
		},
		func(resp CalendarSetResponse) map[string]*Calendar {
			return resp.Created
		},
		func(resp CalendarGetResponse) []Calendar {
			return resp.List
		},
		accountId, calendar,
		ctx,
	)
}

func (j *Client) DeleteCalendar(accountId string, destroyIds []string, ctx Context) (map[string]SetError, SessionState, State, Language, Error) {
	return destroy(j, "DeleteCalendar", NS_CALENDARS,
		func(accountId string, destroy []string) CalendarSetCommand {
			return CalendarSetCommand{AccountId: accountId, Destroy: destroy}
		},
		CalendarSetResponse{},
		accountId, destroyIds,
		ctx,
	)
}

func (j *Client) UpdateCalendar(accountId string, id string, changes CalendarChange, ctx Context) (Calendar, SessionState, State, Language, Error) {
	return update(j, "UpdateCalendar", NS_CALENDARS,
		func(update map[string]PatchObject) CalendarSetCommand {
			return CalendarSetCommand{AccountId: accountId, Update: update}
		},
		func(id string) CalendarGetCommand {
			return CalendarGetCommand{AccountId: accountId, Ids: []string{id}}
		},
		func(resp CalendarSetResponse) map[string]SetError { return resp.NotUpdated },
		func(resp CalendarGetResponse) Calendar { return resp.List[0] },
		id, changes,
		ctx,
	)
}
