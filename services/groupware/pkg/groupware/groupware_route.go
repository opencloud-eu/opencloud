package groupware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

var (
	defaultAccountIds = []string{"_", "*"}
)

const (
	UriParamAccountId                 = "accountid"
	UriParamMailboxId                 = "mailbox"
	UriParamEmailId                   = "emailid"
	UriParamIdentityId                = "identityid"
	UriParamBlobId                    = "blobid"
	UriParamBlobName                  = "blobname"
	UriParamStreamId                  = "stream"
	UriParamRole                      = "role"
	UriParamAddressBookId             = "addressbookid"
	UriParamCalendarId                = "calendarid"
	UriParamTaskListId                = "tasklistid"
	UriParamContactId                 = "contactid"
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
	QueryParamSearchMessageId         = "messageId"
	QueryParamSearchFetchBodies       = "fetchbodies"
	QueryParamSearchFetchEmails       = "fetchemails"
	QueryParamOffset                  = "offset"
	QueryParamLimit                   = "limit"
	QueryParamDays                    = "days"
	QueryParamPartId                  = "partId"
	QueryParamAttachmentName          = "name"
	QueryParamAttachmentBlobId        = "blobId"
	QueryParamSeen                    = "seen"
	QueryParamUndesirable             = "undesirable"
	QueryParamMarkAsSeen              = "markAsSeen"
	HeaderSince                       = "if-none-match"
)

func (g *Groupware) Route(r chi.Router) {
	r.Get("/", g.Index)
	r.Get("/accounts", g.GetAccounts)
	r.Route("/accounts/all", func(r chi.Router) {
		r.Get("/", g.GetAccounts)
		r.Route("/mailboxes", func(r chi.Router) {
			r.Get("/", g.GetMailboxesForAllAccounts) // ?role=
			r.Get("/changes", g.GetMailboxChangesForAllAccounts)
			r.Get("/roles", g.GetMailboxRoles)                       // ?role=
			r.Get("/roles/{role}", g.GetMailboxByRoleForAllAccounts) // ?role=
		})
		r.Route("/emails", func(r chi.Router) {
			r.Get("/", g.GetEmailsForAllAccounts)
			r.Get("/latest/summary", g.GetLatestEmailsSummaryForAllAccounts) // ?limit=10&seen=true&undesirable=true
		})
		r.Get("/quota", g.GetQuotaForAllAccounts)
	})
	r.Route("/accounts/{accountid}", func(r chi.Router) {
		r.Get("/", g.GetAccount)
		r.Route("/identities", func(r chi.Router) {
			r.Get("/", g.GetIdentities)
			r.Get("/{identityid}", g.GetIdentityById)
			r.Post("/", g.AddIdentity)
			r.Patch("/{identityid}", g.ModifyIdentity)
			r.Delete("/{identityid}", g.DeleteIdentity)
		})
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
			r.Delete("/", g.DeleteEmails)
			r.Get("/{emailid}", g.GetEmailsById) // Accept:message/rfc822
			r.Put("/{emailid}", g.ReplaceEmail)
			r.Patch("/{emailid}", g.UpdateEmail)
			r.Patch("/{emailid}/keywords", g.UpdateEmailKeywords)
			r.Post("/{emailid}/keywords", g.AddEmailKeywords)
			r.Delete("/{emailid}/keywords", g.RemoveEmailKeywords)
			r.Delete("/{emailid}", g.DeleteEmail)
			Report(r, "/{emailid}", g.RelatedToEmail)
			r.Get("/{emailid}/related", g.RelatedToEmail)
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
			r.Route("/contacts", func(r chi.Router) {
				r.Post("/", g.CreateContact)
				r.Delete("/{contactid}", g.DeleteContact)
			})
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
