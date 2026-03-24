package groupware

import (
	"net/http"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Get all calendars of an account.
func (g *Groupware) GetCalendars(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}

		calendars, sessionState, state, lang, jerr := g.jmap.GetCalendars(accountId, req.session, req.ctx, req.logger, req.language(), nil)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, calendars, sessionState, CalendarResponseObjectType, state)
	})
}

// Get a calendar of an account by its identifier.
func (g *Groupware) GetCalendarById(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		calendarId, err := req.PathParam(UriParamCalendarId)
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(UriParamCalendarId, log.SafeString(calendarId))

		logger := log.From(l)
		calendars, sessionState, state, lang, jerr := g.jmap.GetCalendars(accountId, req.session, req.ctx, logger, req.language(), []string{calendarId})
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if len(calendars.NotFound) > 0 {
			return req.notFound(accountId, sessionState, CalendarResponseObjectType, state)
		} else {
			return req.respond(accountId, calendars.Calendars[0], sessionState, CalendarResponseObjectType, state)
		}
	})
}

// Get all the events in a calendar of an account by its identifier.
func (g *Groupware) GetEventsInCalendar(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		calendarId, err := req.PathParam(UriParamCalendarId)
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(UriParamCalendarId, log.SafeString(calendarId))

		offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Uint(QueryParamOffset, offset)
		}

		limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaults.contactLimit)
		if err != nil {
			return req.error(accountId, err)
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
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if events, ok := eventsByAccountId[accountId]; ok {
			return req.respond(accountId, events, sessionState, EventResponseObjectType, state)
		} else {
			return req.notFound(accountId, sessionState, EventResponseObjectType, state)
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
			return req.error(accountId, err)
		}

		logger := log.From(l)
		created, sessionState, state, lang, jerr := g.jmap.CreateCalendarEvent(accountId, req.session, req.ctx, logger, req.language(), create)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, created, sessionState, EventResponseObjectType, state)
	})
}

func (g *Groupware) DeleteCalendarEvent(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		eventId, err := req.PathParam(UriParamEventId)
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(UriParamEventId, log.SafeString(eventId))

		logger := log.From(l)

		deleted, sessionState, state, lang, jerr := g.jmap.DeleteCalendarEvent(accountId, []string{eventId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		for _, e := range deleted {
			desc := e.Description
			if desc != "" {
				return req.errorS(accountId, apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
					withDetail(e.Description),
				), sessionState)
			} else {
				return req.errorS(accountId, apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
				), sessionState)
			}
		}
		return req.noContent(accountId, sessionState, EventResponseObjectType, state)
	})
}

func (g *Groupware) ParseIcalBlob(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForBlob()
		if err != nil {
			return req.error(accountId, err)
		}

		blobId, err := req.PathParam(UriParamBlobId)
		if err != nil {
			return req.error(accountId, err)
		}

		blobIds := strings.Split(blobId, ",")
		l := req.logger.With().Array(UriParamBlobId, log.SafeStringArray(blobIds))
		logger := log.From(l)

		resp, sessionState, state, lang, jerr := g.jmap.ParseICalendarBlob(accountId, req.session, req.ctx, logger, req.language(), blobIds)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, resp, sessionState, EventResponseObjectType, state)
	})
}
