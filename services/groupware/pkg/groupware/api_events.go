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
		curryNoNextMapQuery(
			g.jmap.QueryCalendarEvents,
			func(a, b jmap.CalendarEvent) int { return 0 }, // TODO
			func(canCalculateChanges jmap.ChangeCalculation, position, limit, total *uint, results []jmap.CalendarEvent) *jmap.CalendarEventSearchResults {
				return &jmap.CalendarEventSearchResults{
					Results:             results,
					CanCalculateChanges: canCalculateChanges,
					Position:            position,
					Limit:               limit,
					Total:               total,
				}
			},
		),
	)
}

func (g *Groupware) GetAllEvents(w http.ResponseWriter, r *http.Request) {
	getallpaged(Event, w, r, g,
		false,
		func(_ string) jmap.CalendarEventFilterElement { return jmap.CalendarEventFilterCondition{} },
		[]jmap.CalendarEventComparator{{Property: jmap.CalendarEventPropertyStart, IsAscending: true}},
		curryNoNextMapQuery(
			g.jmap.QueryCalendarEvents,
			func(a, b jmap.CalendarEvent) int { return 0 }, // TODO
			func(canCalculateChanges jmap.ChangeCalculation, position, limit, total *uint, results []jmap.CalendarEvent) *jmap.CalendarEventSearchResults {
				return &jmap.CalendarEventSearchResults{
					Results:             results,
					CanCalculateChanges: canCalculateChanges,
					Position:            position,
					Limit:               limit,
					Total:               total,
				}
			},
		),
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
			return req.errorV(accountId, err)
		}

		blobId, err := req.PathParam(UriParamBlobId)
		if err != nil {
			return req.errorV(accountId, err)
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
