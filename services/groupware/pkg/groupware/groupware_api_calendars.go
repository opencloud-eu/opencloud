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
		var _ string = accountId

		return response(AllCalendars, req.session.State, "")
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
		var _ string = accountId

		calendarId := chi.URLParam(r, UriParamCalendarId)
		// TODO replace with proper implementation
		for _, calendar := range AllCalendars {
			if calendar.Id == calendarId {
				return response(calendar, req.session.State, "")
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
		var _ string = accountId

		calendarId := chi.URLParam(r, UriParamCalendarId)
		// TODO replace with proper implementation
		events, ok := EventsMapByCalendarId[calendarId]
		if !ok {
			return notFoundResponse(req.session.State)
		}
		return response(events, req.session.State, "")
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
