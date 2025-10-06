package groupware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

const (
	defaultAccountId = "_"

	UriParamAccountId                 = "accountid"
	UriParamMailboxId                 = "mailbox"
	UriParamEmailId                   = "emailid"
	UriParamBlobId                    = "blobid"
	UriParamBlobName                  = "blobname"
	UriParamStreamId                  = "stream"
	UriParamRole                      = "role"
	UriParamAddressBookId             = "addressbookid"
	UriParamCalendarId                = "calendarid"
	UriParamTaskListId                = "tasklistid"
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
	QueryParamPartId                  = "partId"
	QueryParamAttachmentName          = "name"
	QueryParamAttachmentBlobId        = "blobId"
	QueryParamUnread                  = "unread"
	QueryParamUndesirable             = "undesirable"
	HeaderSince                       = "if-none-match"
)

func (g *Groupware) Route(r chi.Router) {
	r.Get("/", g.Index)
	r.Get("/accounts", g.GetAccounts)
	r.Route("/accounts/all", func(r chi.Router) {
		r.Route("/mailboxes", func(r chi.Router) {
			r.Get("/", g.GetMailboxesForAllAccounts) // ?role=
			r.Get("/changes", g.GetMailboxChangesForAllAccounts)
			r.Get("/roles", g.GetMailboxRoles)                       // ?role=
			r.Get("/roles/{role}", g.GetMailboxByRoleForAllAccounts) // ?role=
		})
		r.Route("/emails", func(r chi.Router) {
			r.Get("/latest/summary", g.GetLatestEmailsSummaryForAllAccounts)
		})
		r.Get("/quota", g.GetQuotaForAllAccounts)
	})
	r.Route("/accounts/{accountid}", func(r chi.Router) {
		r.Get("/", g.GetAccount)
		r.Get("/bootstrap", g.GetAccountBootstrap)
		r.Get("/identities", g.GetIdentities)
		r.Get("/vacation", g.GetVacation)
		r.Put("/vacation", g.SetVacation)
		r.Get("/quota", g.GetQuota)
		r.Route("/mailboxes", func(r chi.Router) {
			r.Get("/", g.GetMailboxes) // ?name=&role=&subcribed=
			r.Get("/{mailbox}", g.GetMailbox)
			r.Get("/{mailbox}/emails", g.GetAllEmailsInMailbox)
			r.Get("/{mailbox}/changes", g.GetMailboxChanges)
		})
		r.Route("/emails", func(r chi.Router) {
			r.Get("/", g.GetEmails) // ?fetchemails=true&fetchbodies=true&text=&subject=&body=&keyword=&keyword=&...
			r.Post("/", g.CreateEmail)
			r.Get("/{emailid}", g.GetEmailsById)
			// r.Put("/{emailid}", g.ReplaceEmail) // TODO
			r.Patch("/{emailid}", g.UpdateEmail)
			r.Delete("/{emailid}", g.DeleteEmail)
			Report(r, "/{emailid}", g.RelatedToEmail)
			r.Get("/{emailid}/attachments", g.GetEmailAttachments) // ?partId=&name=?&blobId=?
		})
		r.Route("/blobs", func(r chi.Router) {
			r.Get("/{blobid}", g.GetBlobMeta)
			r.Get("/{blobid}/{blobname}", g.DownloadBlob) // ?type=
		})
		r.Route("/addressbooks", func(r chi.Router) {
			r.Get("/", g.GetAddressbooks)
			r.Get("/{addressbookid}", g.GetAddressbook)
			r.Get("/{addressbookid}/contacts", g.GetContactsInAddressbook)
		})
		r.Route("/calendars", func(r chi.Router) {
			r.Get("/", g.GetCalendars)
			r.Get("/{calendarid}", g.GetCalendarById)
			r.Get("/{calendarid}/events", g.GetEventsInCalendar)
		})
		r.Route("/tasklists", func(r chi.Router) {
			r.Get("/", g.GetTaskLists)
			r.Get("/{tasklistid}", g.GetTaskListById)
			r.Get("/{tasklistid}/tasks", g.GetTasksInTaskList)
		})
	})

	r.HandleFunc("/events/{stream}", g.ServeSSE)

	r.NotFound(g.NotFound)
	r.MethodNotAllowed(g.MethodNotAllowed)
}

func Report(r chi.Router, pattern string, h http.HandlerFunc) {
	r.MethodFunc("REPORT", pattern, h)
}
