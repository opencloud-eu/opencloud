package groupware

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
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
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(calendars, sessionState, state, lang)
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

		calendarId := chi.URLParam(r, UriParamCalendarId)
		l = l.Str(UriParamCalendarId, log.SafeString(calendarId))

		logger := log.From(l)
		calendars, sessionState, state, lang, jerr := g.jmap.GetCalendars(accountId, req.session, req.ctx, logger, req.language(), []string{calendarId})
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if len(calendars.NotFound) > 0 {
			return notFoundResponse(sessionState)
		} else {
			return etagResponse(calendars.Calendars[0], sessionState, state, lang)
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

		calendarId := chi.URLParam(r, UriParamCalendarId)
		l = l.Str(UriParamCalendarId, log.SafeString(calendarId))

		offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint(QueryParamOffset, offset)
		}

		limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaultContactLimit)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint(QueryParamLimit, limit)
		}

		filter := jmap.CalendarEventFilterCondition{
			InCalendar: calendarId,
		}
		sortBy := []jmap.CalendarEventComparator{{Property: jmap.CalendarEventPropertyUpdated, IsAscending: false}}

		logger := log.From(l)
		eventsByAccountId, sessionState, state, lang, jerr := g.jmap.QueryCalendarEvents([]string{accountId}, req.session, req.ctx, logger, req.language(), filter, sortBy, offset, limit)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if events, ok := eventsByAccountId[accountId]; ok {
			return etagResponse(events, sessionState, state, lang)
		} else {
			return notFoundResponse(sessionState)
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

		calendarId := chi.URLParam(r, UriParamCalendarId)
		l = l.Str(UriParamCalendarId, log.SafeString(calendarId))

		var create jmap.CalendarEvent
		err := req.body(&create)
		if err != nil {
			return errorResponse(err)
		}

		logger := log.From(l)
		created, sessionState, state, lang, jerr := g.jmap.CreateCalendarEvent(accountId, req.session, req.ctx, logger, req.language(), create)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		return etagResponse(created, sessionState, state, lang)
	})
}

func (g *Groupware) DeleteCalendarEvent(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		calendarId := chi.URLParam(r, UriParamCalendarId)
		eventId := chi.URLParam(r, UriParamEventId)
		l.Str(UriParamCalendarId, log.SafeString(calendarId)).Str(UriParamEventId, log.SafeString(eventId))

		logger := log.From(l)

		deleted, sessionState, state, _, jerr := g.jmap.DeleteCalendarEvent(accountId, []string{eventId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		for _, e := range deleted {
			desc := e.Description
			if desc != "" {
				return errorResponseWithSessionState(apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
					withDetail(e.Description),
				), sessionState)
			} else {
				return errorResponseWithSessionState(apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
				), sessionState)
			}
		}
		return noContentResponseWithEtag(sessionState, state)
	})
}

func (g *Groupware) ParseIcalBlob(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForBlob()
		if err != nil {
			return errorResponse(err)
		}

		blobId := chi.URLParam(r, UriParamBlobId)

		blobIds := strings.Split(blobId, ",")
		l := req.logger.With().Array(UriParamBlobId, log.SafeStringArray(blobIds))
		logger := log.From(l)

		resp, sessionState, state, lang, jerr := g.jmap.ParseICalendarBlob(accountId, req.session, req.ctx, logger, req.language(), blobIds)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		return etagResponse(resp, sessionState, state, lang)
	})
}
