package groupware

import (
	"github.com/go-chi/chi/v5"
)

const (
	defaultAccountId = "*"

	UriParamAccountId                 = "accountid"
	UriParamMailboxId                 = "mailbox"
	UriParamMessageId                 = "messageid"
	UriParamBlobId                    = "blobid"
	UriParamBlobName                  = "blobname"
	UriParamStreamId                  = "stream"
	QueryParamMailboxSearchName       = "name"
	QueryParamMailboxSearchRole       = "role"
	QueryParamMailboxSearchSubscribed = "subscribed"
	QueryParamBlobType                = "type"
	QueryParamSince                   = "since"
	QueryParamMaxChanges              = "maxchanges"
	QueryParamMailboxId               = "mailbox"
	QueryParamNotInMailboxId          = "notmailbox"
	QueryParamSearchText              = "text"
	QueryParamSearchFrom              = "from"
	QueryParamSearchTo                = "to"
	QueryParamSearchCc                = "cc"
	QueryParamSearchBcc               = "bcc"
	QueryParamSearchSubject           = "subject"
	QueryParamSearchBody              = "body"
	QueryParamSearchBefore            = "before"
	QueryParamSearchAfter             = "after"
	QueryParamSearchMinSize           = "minsize"
	QueryParamSearchMaxSize           = "maxsize"
	QueryParamSearchKeyword           = "keyword"
	QueryParamSearchFetchBodies       = "fetchbodies"
	QueryParamSearchFetchEmails       = "fetchemails"
	QueryParamOffset                  = "offset"
	QueryParamLimit                   = "limit"
	QueryParamDays                    = "days"
	HeaderSince                       = "if-none-match"
)

func (g *Groupware) Route(r chi.Router) {
	r.HandleFunc("/events/{stream}", g.ServeSSE)

	r.Get("/", g.Index)
	r.Get("/accounts", g.GetAccounts)
	r.Route("/accounts/{accountid}", func(r chi.Router) {
		r.Get("/", g.GetAccount)
		r.Get("/bootstrap", g.GetAccountBootstrap)
		r.Get("/identities", g.GetIdentities)
		r.Get("/vacation", g.GetVacation)
		r.Put("/vacation", g.SetVacation)
		r.Route("/mailboxes", func(r chi.Router) {
			r.Get("/", g.GetMailboxes) // ?name=&role=&subcribed=
			r.Get("/{mailbox}", g.GetMailbox)
			r.Get("/{mailbox}/messages", g.GetAllMessagesInMailbox)
		})
		r.Route("/messages", func(r chi.Router) {
			r.Get("/", g.GetMessages) // ?fetchemails=true&fetchbodies=true&text=&subject=&body=&keyword=&keyword=&...
			r.Post("/", g.CreateMessage)
			r.Get("/{messageid}", g.GetMessagesById)
			// r.Put("/{messageid}", g.ReplaceMessage) // TODO
			r.Patch("/{messageid}", g.UpdateMessage)
			r.Delete("/{messageid}", g.DeleteMessage)
			r.MethodFunc("REPORT", "/{messageid}", g.RelatedToMessage)
		})
		r.Route("/blobs", func(r chi.Router) {
			r.Get("/{blobid}", g.GetBlob)
			r.Get("/{blobid}/{blobname}", g.DownloadBlob) // ?type=
		})
	})

	r.NotFound(g.NotFound)
	r.MethodNotAllowed(g.MethodNotAllowed)
}
