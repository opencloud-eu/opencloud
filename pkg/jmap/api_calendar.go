package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

var NS_CALENDARS = ns(JmapCalendars)

func (j *Client) ParseICalendarBlob(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, blobIds []string) (CalendarEventParseResponse, SessionState, State, Language, Error) {
	logger = j.logger("ParseICalendarBlob", session, logger)

	cmd, err := j.request(session, logger, NS_CALENDARS,
		invocation(CommandCalendarEventParse, CalendarEventParseCommand{AccountId: accountId, BlobIds: blobIds}, "0"),
	)
	if err != nil {
		return CalendarEventParseResponse{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (CalendarEventParseResponse, State, Error) {
		var response CalendarEventParseResponse
		err = retrieveResponseMatchParameters(logger, body, CommandCalendarEventParse, "0", &response)
		if err != nil {
			return CalendarEventParseResponse{}, "", err
		}
		return response, "", nil
	})
}

type CalendarsResponse struct {
	Calendars []Calendar `json:"calendars"`
	NotFound  []string   `json:"notFound,omitempty"`
}

func (j *Client) GetCalendars(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (CalendarsResponse, SessionState, State, Language, Error) {
	return getTemplate(j, "GetCalendars", NS_CALENDARS, CommandCalendarGet,
		func(accountId string, ids []string) CalendarGetCommand {
			return CalendarGetCommand{AccountId: accountId, Ids: ids}
		},
		func(resp CalendarGetResponse) CalendarsResponse {
			return CalendarsResponse{Calendars: resp.List, NotFound: resp.NotFound}
		},
		func(resp CalendarGetResponse) State { return resp.State },
		accountId, session, ctx, logger, acceptLanguage, ids,
	)
}

type CalendarChanges struct {
	HasMoreChanges bool       `json:"hasMoreChanges"`
	OldState       State      `json:"oldState,omitempty"`
	NewState       State      `json:"newState"`
	Created        []Calendar `json:"created,omitempty"`
	Updated        []Calendar `json:"updated,omitempty"`
	Destroyed      []string   `json:"destroyed,omitempty"`
}

// Retrieve Calendar changes since a given state.
// @apidoc calendar,changes
func (j *Client) GetCalendarChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, sinceState State, maxChanges uint) (CalendarChanges, SessionState, State, Language, Error) {
	return changesTemplate(j, "GetCalendarChanges", NS_CALENDARS,
		CommandCalendarChanges, CommandCalendarGet,
		func() CalendarChangesCommand {
			return CalendarChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
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
		func(resp CalendarChangesResponse) (State, State, bool, []string) {
			return resp.OldState, resp.NewState, resp.HasMoreChanges, resp.Destroyed
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
		func(resp CalendarGetResponse) State { return resp.State },
		session, ctx, logger, acceptLanguage,
	)
}

func (j *Client) QueryCalendarEvents(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, //NOSONAR
	filter CalendarEventFilterElement, sortBy []CalendarEventComparator,
	position uint, limit uint) (map[string][]CalendarEvent, SessionState, State, Language, Error) {
	logger = j.logger("QueryCalendarEvents", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	if sortBy == nil {
		sortBy = []CalendarEventComparator{{Property: CalendarEventPropertyStart, IsAscending: false}}
	}

	invocations := make([]Invocation, len(uniqueAccountIds)*2)
	for i, accountId := range uniqueAccountIds {
		query := CalendarEventQueryCommand{
			AccountId: accountId,
			Filter:    filter,
			Sort:      sortBy,
		}
		if limit > 0 {
			query.Limit = limit
		}
		if position > 0 {
			query.Position = position
		}
		invocations[i*2+0] = invocation(CommandCalendarEventQuery, query, mcid(accountId, "0"))
		invocations[i*2+1] = invocation(CommandCalendarEventGet, CalendarEventGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				Name:     CommandCalendarEventQuery,
				Path:     "/ids/*",
				ResultOf: mcid(accountId, "0"),
			},
			// Properties: CalendarEventProperties, // to also retrieve UTCStart and UTCEnd
		}, mcid(accountId, "1"))
	}
	cmd, err := j.request(session, logger, NS_CALENDARS, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string][]CalendarEvent, State, Error) {
		resp := map[string][]CalendarEvent{}
		stateByAccountId := map[string]State{}
		for _, accountId := range uniqueAccountIds {
			var response CalendarEventGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandCalendarEventGet, mcid(accountId, "1"), &response)
			if err != nil {
				return nil, "", err
			}
			if len(response.NotFound) > 0 {
				// TODO what to do when there are not-found calendarevents here? potentially nothing, they could have been deleted between query and get?
			}
			resp[accountId] = response.List
			stateByAccountId[accountId] = response.State
		}
		return resp, squashState(stateByAccountId), nil
	})
}

type CalendarEventChanges struct {
	OldState       State           `json:"oldState,omitempty"`
	NewState       State           `json:"newState"`
	HasMoreChanges bool            `json:"hasMoreChanges"`
	Created        []CalendarEvent `json:"created,omitempty"`
	Updated        []CalendarEvent `json:"updated,omitempty"`
	Destroyed      []string        `json:"destroyed,omitempty"`
}

func (j *Client) GetCalendarEventChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, sinceState State, maxChanges uint) (CalendarEventChanges, SessionState, State, Language, Error) {
	return changesTemplate(j, "GetCalendarEventChanges", NS_CALENDARS,
		CommandCalendarEventChanges, CommandCalendarEventGet,
		func() CalendarEventChangesCommand {
			return CalendarEventChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
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
		func(resp CalendarEventChangesResponse) (State, State, bool, []string) {
			return resp.OldState, resp.NewState, resp.HasMoreChanges, resp.Destroyed
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
		func(resp CalendarEventGetResponse) State { return resp.State },
		session, ctx, logger, acceptLanguage,
	)
}

func (j *Client) CreateCalendarEvent(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, create CalendarEvent) (*CalendarEvent, SessionState, State, Language, Error) {
	return createTemplate(j, "CreateCalendarEvent", NS_CALENDARS, CalendarEventType, CommandCalendarEventSet, CommandCalendarEventGet,
		func(accountId string, create map[string]CalendarEvent) CalendarEventSetCommand {
			return CalendarEventSetCommand{AccountId: accountId, Create: create}
		},
		func(accountId string, ref string) CalendarEventGetCommand {
			return CalendarEventGetCommand{AccountId: accountId, Ids: []string{ref}}
		},
		func(resp CalendarEventSetResponse) map[string]*CalendarEvent {
			return resp.Created
		},
		func(resp CalendarEventSetResponse) map[string]SetError {
			return resp.NotCreated
		},
		func(resp CalendarEventGetResponse) []CalendarEvent {
			return resp.List
		},
		func(resp CalendarEventSetResponse) State {
			return resp.NewState
		},
		accountId, session, ctx, logger, acceptLanguage, create)
}

func (j *Client) DeleteCalendarEvent(accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]SetError, SessionState, State, Language, Error) {
	return deleteTemplate(j, "DeleteCalendarEvent", NS_CALENDARS, CommandCalendarEventSet,
		func(accountId string, destroy []string) CalendarEventSetCommand {
			return CalendarEventSetCommand{AccountId: accountId, Destroy: destroy}
		},
		func(resp CalendarEventSetResponse) map[string]SetError { return resp.NotDestroyed },
		func(resp CalendarEventSetResponse) State { return resp.NewState },
		accountId, destroy, session, ctx, logger, acceptLanguage)
}
