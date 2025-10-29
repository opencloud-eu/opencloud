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
	UriParamMailboxId                 = "mailboxid"
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
	QueryParamIdentityId              = "identity"
	QueryParamMoveFromMailboxId       = "move-from"
	QueryParamMoveToMailboxId         = "move-to"
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
	r.Route("/accounts", func(r chi.Router) {
		r.Get("/", g.GetAccounts)
		r.Route("/all", func(r chi.Router) {
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
			r.Route("/quota", func(r chi.Router) {
				r.Get("/", g.GetQuotaForAllAccounts)
			})
		})
		r.Route("/{accountid}", func(r chi.Router) {
			r.Get("/", g.GetAccount)
			r.Route("/identities", func(r chi.Router) {
				r.Get("/", g.GetIdentities)
				r.Post("/", g.AddIdentity)
				r.Route("/{identityid}", func(r chi.Router) {
					r.Get("/", g.GetIdentityById)
					r.Patch("/", g.ModifyIdentity)
					r.Delete("/", g.DeleteIdentity)
				})
			})
			r.Get("/vacation", g.GetVacation)
			r.Put("/vacation", g.SetVacation)
			r.Get("/quota", g.GetQuota)
			r.Route("/mailboxes", func(r chi.Router) {
				r.Get("/", g.GetMailboxes) // ?name=&role=&subcribed=
				r.Post("/", g.CreateMailbox)
				r.Route("/{mailboxid}", func(r chi.Router) {
					r.Get("/", g.GetMailbox)
					r.Get("/emails", g.GetAllEmailsInMailbox)
					r.Get("/changes", g.GetMailboxChanges)
					r.Patch("/", g.UpdateMailbox)
					r.Delete("/", g.DeleteMailbox)
				})
			})
			r.Route("/emails", func(r chi.Router) {
				r.Get("/", g.GetEmails) // ?fetchemails=true&fetchbodies=true&text=&subject=&body=&keyword=&keyword=&...
				r.Post("/", g.CreateEmail)
				r.Delete("/", g.DeleteEmails)
				r.Route("/{emailid}", func(r chi.Router) {
					r.Get("/", g.GetEmailsById) // Accept:message/rfc822
					r.Put("/", g.ReplaceEmail)
					r.Post("/", g.SendEmail)
					r.Patch("/", g.UpdateEmail)
					r.Delete("/", g.DeleteEmail)
					Report(r, "/", g.RelatedToEmail)
					r.Route("/related", func(r chi.Router) {
						r.Get("/", g.RelatedToEmail)
					})
					r.Route("/keywords", func(r chi.Router) {
						r.Patch("/", g.UpdateEmailKeywords)
						r.Post("/", g.AddEmailKeywords)
						r.Delete("/", g.RemoveEmailKeywords)
					})
					r.Route("/attachments", func(r chi.Router) {
						r.Get("/", g.GetEmailAttachments) // ?partId=&name=?&blobId=?
					})
				})
			})
			r.Route("/blobs", func(r chi.Router) {
				r.Get("/{blobid}", g.GetBlobMeta)
				r.Get("/{blobid}/{blobname}", g.DownloadBlob) // ?type=
			})
			r.Route("/ical", func(r chi.Router) {
				r.Get("/{blobid}", g.ParseIcalBlob)
			})
			r.Route("/addressbooks", func(r chi.Router) {
				r.Get("/", g.GetAddressbooks)
				r.Route("/{addressbookid}", func(r chi.Router) {
					r.Get("/", g.GetAddressbook)
					r.Get("/contacts", g.GetContactsInAddressbook)
				})
				r.Route("/contacts", func(r chi.Router) {
					r.Post("/", g.CreateContact)
					r.Delete("/{contactid}", g.DeleteContact)
				})
			})
			r.Route("/calendars", func(r chi.Router) {
				r.Get("/", g.GetCalendars)
				r.Route("/{calendarid}", func(r chi.Router) {
					r.Get("/", g.GetCalendarById)
					r.Get("/events", g.GetEventsInCalendar)
				})
			})
			r.Route("/tasklists", func(r chi.Router) {
				r.Get("/", g.GetTaskLists)
				r.Route("/{tasklistid}", func(r chi.Router) {
					r.Get("/", g.GetTaskListById)
					r.Get("/tasks", g.GetTasksInTaskList)
				})
			})
		})
	})

	r.HandleFunc("/events/{stream}", g.ServeSSE)

	r.NotFound(g.NotFound)
	r.MethodNotAllowed(g.MethodNotAllowed)
}

func Report(r chi.Router, pattern string, h http.HandlerFunc) {
	r.MethodFunc("REPORT", pattern, h)
}
