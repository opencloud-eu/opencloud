package jmap

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

func (j *Client) ParseICalendarBlob(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, blobIds []string) (CalendarEventParseResponse, SessionState, State, Language, Error) {
	logger = j.logger("ParseICalendarBlob", session, logger)

	cmd, err := j.request(session, logger,
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
	return getTemplate(j, "GetCalendars", CommandCalendarGet,
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

func (j *Client) QueryCalendarEvents(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string,
	filter CalendarEventFilterElement, sortBy []CalendarEventComparator,
	position uint, limit uint) (map[string][]CalendarEvent, SessionState, State, Language, Error) {
	logger = j.logger("QueryCalendarEvents", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	if sortBy == nil {
		sortBy = []CalendarEventComparator{{Property: CalendarEventPropertyUpdated, IsAscending: false}}
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
		}, mcid(accountId, "1"))
	}
	cmd, err := j.request(session, logger, invocations...)
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
				// TODO what to do when there are not-found emails here? potentially nothing, they could have been deleted between query and get?
			}
			resp[accountId] = response.List
			stateByAccountId[accountId] = response.State
		}
		return resp, squashState(stateByAccountId), nil
	})
}

func (j *Client) CreateCalendarEvent(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, create CalendarEvent) (*CalendarEvent, SessionState, State, Language, Error) {
	return createTemplate(j, "CreateCalendarEvent", CalendarEventType, CommandCalendarEventSet, CommandCalendarEventGet,
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
	return deleteTemplate(j, "DeleteCalendarEvent", CommandCalendarEventSet,
		func(accountId string, destroy []string) CalendarEventSetCommand {
			return CalendarEventSetCommand{AccountId: accountId, Destroy: destroy}
		},
		func(resp CalendarEventSetResponse) map[string]SetError { return resp.NotDestroyed },
		func(resp CalendarEventSetResponse) State { return resp.NewState },
		accountId, destroy, session, ctx, logger, acceptLanguage)
}

func getTemplate[GETREQ any, GETRESP any, RESP any](
	client *Client, name string, getCommand Command,
	getCommandFactory func(string, []string) GETREQ,
	mapper func(GETRESP) RESP,
	stateMapper func(GETRESP) State,
	accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (RESP, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	var zero RESP

	cmd, err := client.request(session, logger,
		invocation(getCommand, getCommandFactory(accountId, ids), "0"),
	)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		var response GETRESP
		err = retrieveResponseMatchParameters(logger, body, getCommand, "0", &response)
		if err != nil {
			return zero, "", err
		}

		return mapper(response), stateMapper(response), nil
	})
}

func createTemplate[T any, SETREQ any, GETREQ any, SETRESP any, GETRESP any](
	client *Client, name string, t ObjectType, setCommand Command, getCommand Command,
	setCommandFactory func(string, map[string]T) SETREQ,
	getCommandFactory func(string, string) GETREQ,
	createdMapper func(SETRESP) map[string]*T,
	notCreatedMapper func(SETRESP) map[string]SetError,
	listMapper func(GETRESP) []T,
	stateMapper func(SETRESP) State,
	accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, create T) (*T, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	createMap := map[string]T{"c": create}
	cmd, err := client.request(session, logger,
		invocation(setCommand, setCommandFactory(accountId, createMap), "0"),
		invocation(getCommand, getCommandFactory(accountId, "#c"), "1"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (*T, State, Error) {
		var setResponse SETRESP
		err = retrieveResponseMatchParameters(logger, body, setCommand, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}

		notCreatedMap := notCreatedMapper(setResponse)
		setErr, notok := notCreatedMap["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", setResponse, setErr)
			return nil, "", setErrorError(setErr, t)
		}

		createdMap := createdMapper(setResponse)
		if created, ok := createdMap["c"]; !ok || created == nil {
			berr := fmt.Errorf("failed to find %s in %s response", string(t), string(setCommand))
			logger.Error().Err(berr)
			return nil, "", simpleError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		var getResponse GETRESP
		err = retrieveResponseMatchParameters(logger, body, getCommand, "1", &getResponse)
		if err != nil {
			return nil, "", err
		}

		list := listMapper(getResponse)

		if len(list) < 1 {
			berr := fmt.Errorf("failed to find %s in %s response", string(t), string(getCommand))
			logger.Error().Err(berr)
			return nil, "", simpleError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		return &list[0], stateMapper(setResponse), nil
	})
}

func deleteTemplate[REQ any, RESP any](client *Client, name string, c Command,
	commandFactory func(string, []string) REQ,
	notDestroyedMapper func(RESP) map[string]SetError,
	stateMapper func(RESP) State,
	accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]SetError, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	cmd, err := client.request(session, logger,
		invocation(c, commandFactory(accountId, destroy), "0"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]SetError, State, Error) {
		var setResponse RESP
		err = retrieveResponseMatchParameters(logger, body, c, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}
		return notDestroyedMapper(setResponse), stateMapper(setResponse), nil
	})
}
