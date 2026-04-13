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
	return get(j, "GetCalendars", CalendarType,
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
	return changes(j, "GetCalendarChanges", CalendarType,
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

func (j *Client) CreateCalendar(accountId string, calendar CalendarChange, ctx Context) (*Calendar, SessionState, State, Language, Error) {
	return create(j, "CreateCalendar", CalendarEventType,
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
	return destroy(j, "DeleteCalendar", CalendarEventType,
		func(accountId string, destroy []string) CalendarSetCommand {
			return CalendarSetCommand{AccountId: accountId, Destroy: destroy}
		},
		CalendarSetResponse{},
		accountId, destroyIds,
		ctx,
	)
}

func (j *Client) UpdateCalendar(accountId string, id string, changes CalendarChange, ctx Context) (Calendar, SessionState, State, Language, Error) {
	return update(j, "UpdateCalendar", CalendarEventType,
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
