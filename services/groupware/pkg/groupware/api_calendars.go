package groupware

import (
	"net/http"

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

// Get the changes to Calendars since a certain State.
// @api:tags calendar,changes
func (g *Groupware) GetCalendarChanges(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		maxChanges, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}

		sinceState := jmap.State(req.OptHeaderParamDoc(HeaderParamSince, "Optionally specifies the state identifier from which on to list calendar changes"))
		if sinceState != "" {
			l = l.Str(HeaderParamSince, log.SafeString(string(sinceState)))
		}

		logger := log.From(l)

		changes, sessionState, state, lang, jerr := g.jmap.GetCalendarChanges(accountId, req.session, req.ctx, logger, req.language(), sinceState, maxChanges)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, changes, sessionState, CalendarResponseObjectType, state)
	})
}

func (g *Groupware) CreateCalendar(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		var create jmap.CalendarChange
		err := req.bodydoc(&create, "The calendar to create")
		if err != nil {
			return req.error(accountId, err)
		}

		logger := log.From(l)
		created, sessionState, state, lang, jerr := g.jmap.CreateCalendar(accountId, req.session, req.ctx, logger, req.language(), create)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, created, sessionState, ContactResponseObjectType, state)
	})
}

func (g *Groupware) DeleteCalendar(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		calendarId, err := req.PathParam(UriParamCalendarId)
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(UriParamCalendarId, log.SafeString(calendarId))

		logger := log.From(l)

		deleted, sessionState, state, lang, jerr := g.jmap.DeleteCalendar(accountId, []string{calendarId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		for _, e := range deleted {
			desc := e.Description
			if desc != "" {
				return req.error(accountId, apiError(
					req.errorId(),
					ErrorFailedToDeleteCalendar,
					withDetail(e.Description),
				))
			} else {
				return req.error(accountId, apiError(
					req.errorId(),
					ErrorFailedToDeleteCalendar,
				))
			}
		}
		return req.noContent(accountId, sessionState, CalendarResponseObjectType, state)
	})
}
