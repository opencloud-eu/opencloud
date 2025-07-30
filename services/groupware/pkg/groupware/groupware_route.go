package groupware

import (
	"github.com/go-chi/chi/v5"
)

func (g Groupware) Route(r chi.Router) {
	r.Get("/", g.Index)
	r.Route("/accounts/{account}", func(r chi.Router) {
		r.Get("/", g.GetAccount)
		r.Get("/mailboxes", g.GetMailboxes) // ?name=&role=&subcribed=
		r.Get("/mailboxes/{id}", g.GetMailboxById)
		r.Get("/{mailbox}/messages", g.GetMessages)
		r.Get("/identity", g.GetIdentity)
		r.Get("/vacation", g.GetVacation)
	})
}
