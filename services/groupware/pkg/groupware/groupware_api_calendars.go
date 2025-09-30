package groupware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
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
		return response(AllCalendars, req.session.State)
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
// Get all calendars of an account.
//
// responses:
//
//	200: GetCalendarById200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetCalendarById(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		calendarId := chi.URLParam(r, UriParamAddressBookId)
		// TODO replace with proper implementation
		for _, calendar := range AllCalendars {
			if calendar.Id == calendarId {
				return response(calendar, req.session.State)
			}
		}
		return notFoundResponse(req.session.State)
	})
}

// When the request succeeds.
// swagger:response GetEventsInCalendar200
type SwaggerGetEventsInCalendar200 struct {
	// in: body
	Body []jmap.CalendarEvent
}

// swagger:route GET /groupware/accounts/{account}/calendars/{calendarid}/events event events_in_addressbook
// Get all the events in a calendarof an account by its identifier.
//
// responses:
//
//	200: GetEventsInCalendar200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetEventsInCalendar(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		calendarId := chi.URLParam(r, UriParamAddressBookId)
		// TODO replace with proper implementation
		events, ok := EventsMapByCalendarId[calendarId]
		if !ok {
			return notFoundResponse(req.session.State)
		}
		return response(events, req.session.State)
	})
}
