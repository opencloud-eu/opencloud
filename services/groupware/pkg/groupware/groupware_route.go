package groupware

import (
	"github.com/go-chi/chi/v5"
)

const (
	UriParamAccount    = "account"
	UriParamMailboxId  = "mailbox"
	QueryParamPage     = "page"
	QueryParamSize     = "size"
	UriParamMessagesId = "id"
)

func (g Groupware) Route(r chi.Router) {
	r.Get("/", g.Index)
	r.Get("/accounts", g.GetAccounts)
	r.Route("/accounts/{account}", func(r chi.Router) {
		r.Get("/", g.GetAccount)
		r.Get("/mailboxes", g.GetMailboxes) // ?name=&role=&subcribed=
		r.Get("/mailboxes/{mailbox}", g.GetMailbox)
		r.Get("/mailboxes/{mailbox}/messages", g.GetAllMessages)
		r.Get("/messages/{id}", g.GetMessagesById)
		r.Get("/identity", g.GetIdentity)
		r.Get("/vacation", g.GetVacation)
	})
	r.NotFound(g.NotFound)
}
