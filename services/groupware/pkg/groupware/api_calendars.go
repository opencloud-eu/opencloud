package groupware

import (
	"net/http"
)

// Get all calendars of an account.
func (g *Groupware) GetCalendars(w http.ResponseWriter, r *http.Request) {
	getall(Calendar, w, r, g, g.jmap.GetCalendars)
}

// Get a calendar of an account by its identifier.
func (g *Groupware) GetCalendarById(w http.ResponseWriter, r *http.Request) {
	get(Calendar, w, r, g, g.jmap.GetCalendars)
}

// Get the changes to Calendars since a certain State.
// @api:tags calendar,changes
func (g *Groupware) GetCalendarChanges(w http.ResponseWriter, r *http.Request) {
	changes(Calendar, w, r, g, g.jmap.GetCalendarChanges)
}

func (g *Groupware) CreateCalendar(w http.ResponseWriter, r *http.Request) {
	create(Calendar, w, r, g, nil, g.jmap.CreateCalendar)
}

func (g *Groupware) DeleteCalendar(w http.ResponseWriter, r *http.Request) {
	delete(Calendar, w, r, g, g.jmap.DeleteCalendar)
}

func (g *Groupware) ModifyCalendar(w http.ResponseWriter, r *http.Request) {
	modify(Calendar, w, r, g, g.jmap.UpdateCalendar)
}
