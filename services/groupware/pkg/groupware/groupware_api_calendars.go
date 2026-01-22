package groupware

import (
	"net/http"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// When the request succeeds.
// swagger:response GetCalendars200
type SwaggerGetCalendars200 struct {
	// in: body
	Body []jmap.Calendar
}

// swagger:route GET /groupware/accounts/{account}/calendars calendar calendars
// Get all calendars of an account.
//
// responses:
//
//	200: GetCalendars200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetCalendars(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}

		calendars, sessionState, state, lang, jerr := g.jmap.GetCalendars(accountId, req.session, req.ctx, req.logger, req.language(), nil)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		return etagResponse(single(accountId), calendars, sessionState, CalendarResponseObjectType, state, lang)
	})
}

// When the request succeeds.
// swagger:response GetCalendarById200
type SwaggerGetCalendarById200 struct {
	// in: body
	Body struct {
		*jmap.Calendar
	}
}

// swagger:route GET /groupware/accounts/{account}/calendars/{calendarid} calendar calendar_by_id
// Get a calendar of an account by its identifier.
//
// responses:
//
//	200: GetCalendarById200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetCalendarById(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		calendarId, err := req.PathParam(UriParamCalendarId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		l = l.Str(UriParamCalendarId, log.SafeString(calendarId))

		logger := log.From(l)
		calendars, sessionState, state, lang, jerr := g.jmap.GetCalendars(accountId, req.session, req.ctx, logger, req.language(), []string{calendarId})
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		if len(calendars.NotFound) > 0 {
			return notFoundResponse(single(accountId), sessionState)
		} else {
			return etagResponse(single(accountId), calendars.Calendars[0], sessionState, CalendarResponseObjectType, state, lang)
		}
	})
}

// When the request succeeds.
// swagger:response GetEventsInCalendar200
type SwaggerGetEventsInCalendar200 struct {
	// in: body
	Body []jmap.CalendarEvent
}

// swagger:route GET /groupware/accounts/{account}/calendars/{calendarid}/events event events_in_addressbook
// Get all the events in a calendar of an account by its identifier.
//
// responses:
//
//	200: GetEventsInCalendar200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetEventsInCalendar(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		calendarId, err := req.PathParam(UriParamCalendarId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		l = l.Str(UriParamCalendarId, log.SafeString(calendarId))

		offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		if ok {
			l = l.Uint(QueryParamOffset, offset)
		}

		limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaults.contactLimit)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		if ok {
			l = l.Uint(QueryParamLimit, limit)
		}

		filter := jmap.CalendarEventFilterCondition{
			InCalendar: calendarId,
		}
		sortBy := []jmap.CalendarEventComparator{{Property: jmap.CalendarEventPropertyUpdated, IsAscending: false}}

		logger := log.From(l)
		eventsByAccountId, sessionState, state, lang, jerr := g.jmap.QueryCalendarEvents(single(accountId), req.session, req.ctx, logger, req.language(), filter, sortBy, offset, limit)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		if events, ok := eventsByAccountId[accountId]; ok {
			return etagResponse(single(accountId), events, sessionState, EventResponseObjectType, state, lang)
		} else {
			return notFoundResponse(single(accountId), sessionState)
		}
	})
}

func (g *Groupware) CreateCalendarEvent(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		var create jmap.CalendarEvent
		err := req.body(&create)
		if err != nil {
			return errorResponse(single(accountId), err)
		}

		logger := log.From(l)
		created, sessionState, state, lang, jerr := g.jmap.CreateCalendarEvent(accountId, req.session, req.ctx, logger, req.language(), create)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}
		return etagResponse(single(accountId), created, sessionState, EventResponseObjectType, state, lang)
	})
}

// @api:tag XYZ
func (g *Groupware) DeleteCalendarEvent(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		eventId, err := req.PathParam(UriParamEventId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		l.Str(UriParamEventId, log.SafeString(eventId))

		logger := log.From(l)

		deleted, sessionState, state, _, jerr := g.jmap.DeleteCalendarEvent(accountId, []string{eventId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		for _, e := range deleted {
			desc := e.Description
			if desc != "" {
				return errorResponseWithSessionState(single(accountId), apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
					withDetail(e.Description),
				), sessionState)
			} else {
				return errorResponseWithSessionState(single(accountId), apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
				), sessionState)
			}
		}
		return noContentResponseWithEtag(single(accountId), sessionState, EventResponseObjectType, state)
	})
}

func (g *Groupware) ParseIcalBlob(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForBlob()
		if err != nil {
			return errorResponse(single(accountId), err)
		}

		blobId, err := req.PathParam(UriParamBlobId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}

		blobIds := strings.Split(blobId, ",")
		l := req.logger.With().Array(UriParamBlobId, log.SafeStringArray(blobIds))
		logger := log.From(l)

		resp, sessionState, state, lang, jerr := g.jmap.ParseICalendarBlob(accountId, req.session, req.ctx, logger, req.language(), blobIds)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}
		return etagResponse(single(accountId), resp, sessionState, EventResponseObjectType, state, lang)
	})
}
