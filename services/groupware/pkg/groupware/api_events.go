package groupware

import (
	"net/http"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Get all the events in a calendar of an account by its identifier.
func (g *Groupware) GetEventsInCalendar(w http.ResponseWriter, r *http.Request) { //NOSONAR
	getallpaged(Event, w, r, g,
		true,
		func(calendarId string) jmap.CalendarEventFilterElement {
			return jmap.CalendarEventFilterCondition{InCalendar: calendarId}
		},
		[]jmap.CalendarEventComparator{{Property: jmap.CalendarEventPropertyStart, IsAscending: true}},
		curryMapQuery(g.jmap.QueryCalendarEvents),
	)

	/*
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

			position, ok, err := req.parseIntParam(QueryParamPosition, 0)
			if err != nil {
				return req.error(accountId, err)
			}
			if ok {
				l = l.Int(QueryParamPosition, position)
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
			sortBy := []jmap.CalendarEventComparator{{Property: jmap.CalendarEventPropertyStart, IsAscending: false}}

			logger := log.From(l)
			ctx := req.ctx.WithLogger(logger)
			eventsByAccountId, sessionState, state, lang, jerr := g.jmap.QueryCalendarEvents(single(accountId), filter, sortBy, position, limit, true, ctx)
			if jerr != nil {
				return req.jmapError(accountId, jerr, sessionState, lang)
			}

			if events, ok := eventsByAccountId[accountId]; ok {
				return req.respond(accountId, events, sessionState, EventResponseObjectType, state, lang)
			} else {
				return req.notFound(accountId, sessionState, EventResponseObjectType, state)
			}
		})
	*/
}

func curryMapQuery[SRES jmap.SearchResults[T], T jmap.Foo, FILTER any, COMP any](
	f func(accountIds []string, filter FILTER, sortBy []COMP, position int, anchor string, anchorOffset *int, limit *uint, calculateTotal bool, ctx jmap.Context) (jmap.Result[map[string]SRES], jmap.Error),
) func(req Request, accountId string, filter FILTER, sortBy []COMP, position int, anchor string, anchorOffset *int, limit *uint, ctx jmap.Context) (jmap.Result[SRES], jmap.Error) {
	return func(req Request, accountId string, filter FILTER, sortBy []COMP, position int, anchor string, anchorOffset *int, limit *uint, ctx jmap.Context) (jmap.Result[SRES], jmap.Error) { //NOSONAR
		result, err := f(single(accountId), filter, sortBy, position, anchor, anchorOffset, limit, true, ctx)
		if err != nil {
			return jmap.ZeroResult[SRES](), err
		} else {
			return jmap.RefineResult(result, func(m map[string]SRES) SRES { return m[accountId] }), err
		}
	}
}

func (g *Groupware) GetAllEvents(w http.ResponseWriter, r *http.Request) {
	getallpaged(Event, w, r, g,
		false,
		func(_ string) jmap.CalendarEventFilterElement { return jmap.CalendarEventFilterCondition{} },
		[]jmap.CalendarEventComparator{{Property: jmap.CalendarEventPropertyStart, IsAscending: true}},
		curryMapQuery(g.jmap.QueryCalendarEvents),
	)
}

func (g *Groupware) GetEventById(w http.ResponseWriter, r *http.Request) {
	get(Event, w, r, g, g.jmap.GetCalendarEvents)
}

// Get changes to Calendar Events since a given State
// @api:tags event,changes
func (g *Groupware) GetEventChanges(w http.ResponseWriter, r *http.Request) {
	changes(Event, w, r, g, g.jmap.GetCalendarEventChanges)
}

func (g *Groupware) CreateEvent(w http.ResponseWriter, r *http.Request) {
	create(Event, w, r, g, nil, g.jmap.CreateCalendarEvent)
}

func (g *Groupware) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	delete(Event, w, r, g, g.jmap.DeleteCalendarEvent)
}

func (g *Groupware) ModifyEvent(w http.ResponseWriter, r *http.Request) {
	modify(Event, w, r, g, g.jmap.UpdateCalendarEvent)
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
		result, jerr := g.jmap.ParseICalendarBlob(accountId, blobIds, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, result)
		}
		return req.respond(accountId, result.Payload, EventResponseObjectType, result)
	})
}
