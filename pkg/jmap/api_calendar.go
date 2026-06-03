package jmap

var NS_CALENDARS = ns(JmapCalendars)

func (j *Client) ParseICalendarBlob(accountId AccountId, blobIds []string, ctx Context) (Result[CalendarEventParseResponse], error) {
	logger := j.logger("ParseICalendarBlob", ctx)

	parse := CalendarEventParseCommand{AccountId: accountId, BlobIds: blobIds}
	cmd, err := j.request(ctx.WithLogger(logger), NS_CALENDARS,
		invocation(parse, "0"),
	)
	if err != nil {
		return ZeroResultV[CalendarEventParseResponse](), err
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

func (j *Client) GetCalendars(accountId AccountId, ids []string, ctx Context) (Result[CalendarGetResponse], error) {
	return get(j, "GetCalendars", CalendarType,
		func(accountId AccountId, ids []string) CalendarGetCommand {
			return CalendarGetCommand{AccountId: accountId, Ids: ids}
		},
		CalendarGetResponse{},
		identity1,
		accountId, ids,
		ctx,
	)
}

type CalendarChanges ChangesTemplate[Calendar]

var _ Changes[Calendar] = CalendarChanges{}

func (c CalendarChanges) GetHasMoreChanges() bool { return c.HasMoreChanges }
func (c CalendarChanges) GetOldState() State      { return c.OldState }
func (c CalendarChanges) GetNewState() State      { return c.NewState }
func (c CalendarChanges) GetCreated() []Calendar  { return c.Created }
func (c CalendarChanges) GetUpdated() []Calendar  { return c.Updated }
func (c CalendarChanges) GetDestroyed() []string  { return c.Destroyed }

// Retrieve Calendar changes since a given state.
// @apidoc calendar,changes
func (j *Client) GetCalendarChanges(accountId AccountId, sinceState State, maxChanges uint, ctx Context) (Result[CalendarChanges], error) {
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

func (j *Client) CreateCalendar(accountId AccountId, calendar CalendarChange, ctx Context) (Result[*Calendar], error) {
	return create(j, "CreateCalendar", CalendarEventType,
		func(accountId AccountId, create map[string]CalendarChange) CalendarSetCommand {
			return CalendarSetCommand{AccountId: accountId, Create: create}
		},
		func(accountId AccountId, ref string) CalendarGetCommand {
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

func (j *Client) DeleteCalendar(accountId AccountId, destroyIds []string, ctx Context) (Result[map[string]SetError], error) {
	return destroy(j, "DeleteCalendar", CalendarEventType,
		func(accountId AccountId, destroy []string) CalendarSetCommand {
			return CalendarSetCommand{AccountId: accountId, Destroy: destroy}
		},
		CalendarSetResponse{},
		accountId, destroyIds,
		ctx,
	)
}

func (j *Client) UpdateCalendar(accountId AccountId, id string, changes CalendarChange, ctx Context) (Result[Calendar], error) {
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
