package groupware

import (
	"github.com/go-chi/chi/v5"
)

const (
	UriParamAccount          = "account"
	UriParamMailboxId        = "mailbox"
	UriParamMessagesId       = "id"
	UriParamBlobId           = "blobid"
	UriParamBlobName         = "blobname"
	QueryParamBlobType       = "type"
	QueryParamSince          = "since"
	QueryParamMailboxId      = "mailbox"
	QueryParamNotInMailboxId = "notmailbox"
	QueryParamSearchText     = "text"
	QueryParamSearchFrom     = "from"
	QueryParamSearchTo       = "to"
	QueryParamSearchCc       = "cc"
	QueryParamSearchBcc      = "bcc"
	QueryParamSearchSubject  = "subject"
	QueryParamSearchBody     = "body"
	QueryParamSearchBefore   = "before"
	QueryParamSearchAfter    = "after"
	QueryParamSearchMinSize  = "minsize"
	QueryParamSearchMaxSize  = "maxsize"
	QueryParamOffset         = "offset"
	QueryParamLimit          = "limit"
	HeaderSince              = "if-none-match"
)

func (g Groupware) Route(r chi.Router) {
	r.Get("/", g.Index)
	r.Get("/accounts", g.GetAccounts)
	r.Route("/accounts/{account}", func(r chi.Router) {
		r.Get("/", g.GetAccount)
		r.Get("/identity", g.GetIdentity)
		r.Get("/vacation", g.GetVacation)
		r.Route("/mailboxes", func(r chi.Router) {
			r.Get("/", g.GetMailboxes) // ?name=&role=&subcribed=
			r.Get("/{mailbox}", g.GetMailbox)
			r.Get("/{mailbox}/messages", g.GetAllMessages)
		})
		r.Route("/messages", func(r chi.Router) {
			r.Get("/", g.GetMessages)
			r.Get("/{id}", g.GetMessagesById)
		})
		r.Route("/blobs", func(r chi.Router) {
			r.Get("/{blobid}", g.GetBlob)
			r.Get("/{blobid}/{blobname}", g.DownloadBlob) // ?type=
		})
	})
	r.NotFound(g.NotFound)
}
