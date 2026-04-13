package groupware

import (
	"net/http"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Get all the events in a calendar of an account by its identifier.
func (g *Groupware) GetEventsInCalendar(w http.ResponseWriter, r *http.Request) { //NOSONAR
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

		offset, ok, err := req.parseIntParam(QueryParamOffset, 0)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Int(QueryParamOffset, offset)
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
		ctx := req.ctx.WithLogger(logger)
		eventsByAccountId, sessionState, state, lang, jerr := g.jmap.QueryCalendarEvents(single(accountId), filter, sortBy, offset, limit, true, ctx)
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

// Get changes to Contacts since a given State
// @api:tags event,changes
func (g *Groupware) GetEventChanges(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		var maxChanges uint = 0
		if v, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0); err != nil {
			return req.error(accountId, err)
		} else if ok {
			maxChanges = v
			l = l.Uint(QueryParamMaxChanges, v)
		}

		sinceState := jmap.State(req.OptHeaderParamDoc(HeaderParamSince, "Specifies the state identifier from which on to list event changes"))
		l = l.Str(HeaderParamSince, log.SafeString(string(sinceState)))

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		changes, sessionState, state, lang, jerr := g.jmap.GetCalendarEventChanges(accountId, sinceState, maxChanges, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		var body jmap.CalendarEventChanges = changes

		return req.respond(accountId, body, sessionState, ContactResponseObjectType, state)
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
		ctx := req.ctx.WithLogger(logger)
		created, sessionState, state, lang, jerr := g.jmap.CreateCalendarEvent(accountId, create, ctx)
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
		ctx := req.ctx.WithLogger(logger)
		deleted, sessionState, state, lang, jerr := g.jmap.DeleteCalendarEvent(accountId, single(eventId), ctx)
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

func (g *Groupware) ModifyCalendarEvent(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needCalendarWithAccount()
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))
		id, err := req.PathParamDoc(UriParamEventId, "The unique identifier of the Calendar Event to modify")
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(UriParamEventId, log.SafeString(id))

		var change jmap.CalendarEventChange
		err = req.body(&change)
		if err != nil {
			return req.error(accountId, err)
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		updated, sessionState, state, lang, jerr := g.jmap.UpdateCalendarEvent(accountId, id, change, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, updated, sessionState, EventResponseObjectType, state)
	})
}

// Parse a blob that contains an iCal file and return it as JSCalendar.
//
// @api:tags calendar,blob
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
		ctx := req.ctx.WithLogger(logger)
		resp, sessionState, state, lang, jerr := g.jmap.ParseICalendarBlob(accountId, blobIds, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, resp, sessionState, EventResponseObjectType, state)
	})
}
