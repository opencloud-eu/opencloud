package groupware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

var (
	defaultAccountIds = []string{"_", "*"}
)

const (
	UriParamAccountId                 = "accountid"     // Identifier of the account
	UriParamMailboxId                 = "mailboxid"     // Identifier of the mailbox
	UriParamEmailId                   = "emailid"       // Identifier of the email
	UriParamIdentityId                = "identityid"    // Identifier of the identity
	UriParamBlobId                    = "blobid"        // Identifier of theblob
	UriParamStreamId                  = "stream"        // Identifier of the stream
	UriParamAddressBookId             = "addressbookid" // Identifier of the address book
	UriParamCalendarId                = "calendarid"    // Identifier of the calendar
	UriParamTaskListId                = "tasklistid"    // Identifier of the tasklist
	UriParamContactId                 = "contactid"     // Identifier of the contact
	UriParamEventId                   = "eventid"       // Idenfitier of the event
	UriParamBlobName                  = "blobname"
	UriParamRole                      = "role"
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
	QueryParamOffset                  = "offset"
	QueryParamLimit                   = "limit"
	QueryParamDays                    = "days"
	QueryParamPartId                  = "partId"
	QueryParamAttachmentName          = "name"
	QueryParamAttachmentBlobId        = "blobId"
	QueryParamSeen                    = "seen"
	QueryParamUndesirable             = "undesirable"
	QueryParamMarkAsSeen              = "markAsSeen"
	QueryParamSort                    = "sort"
	QueryParamMailboxes               = "mailboxes"
	QueryParamEmails                  = "emails"
	QueryParamAddressbooks            = "addressbooks"
	QueryParamContacts                = "contacts"
	QueryParamCalendars               = "calendars"
	QueryParamEvents                  = "events"
	QueryParamQuotas                  = "quotas"
	QueryParamIdentities              = "identities"
	QueryParamEmailSubmissions        = "submissions"
	QueryParamId                      = "id"
	QueryParamCalculateTotal          = "total"
	HeaderParamSince                  = "if-none-match"
)

func (g *Groupware) Route(r chi.Router) {
	r.Get("/", g.Index)
	r.Route("/accounts", func(r chi.Router) {
		r.Get("/", g.GetAccountsWithTheirIdentities)
		r.Route("/all", func(r chi.Router) {
			r.Get("/", g.GetAccounts)
			r.Route("/mailboxes", func(r chi.Router) { //NOSONAR
				r.Get("/", g.GetMailboxesForAllAccounts) // ?role=
				r.Get("/changes", g.GetMailboxChangesForAllAccounts)
				r.Get("/roles", g.GetMailboxRoles)                       // ?role=
				r.Get("/roles/{role}", g.GetMailboxByRoleForAllAccounts) // ?role=
			})
			r.Route("/emails", func(r chi.Router) { //NOSONAR
				r.Get("/", g.GetEmailsForAllAccounts)
				r.Get("/latest/summary", g.GetLatestEmailsSummaryForAllAccounts) // ?limit=10&seen=true&undesirable=true
			})
			r.Route("/quota", func(r chi.Router) {
				r.Get("/", g.GetQuotaForAllAccounts)
			})
		})
		r.Route("/{accountid}", func(r chi.Router) {
			r.Get("/", g.GetAccountById)
			r.Route("/identities", func(r chi.Router) {
				r.Get("/", g.GetIdentities)
				r.Post("/", g.CreateIdentity)
				r.Route("/{identityid}", func(r chi.Router) {
					r.Get("/", g.GetIdentityById)
					r.Patch("/", g.ModifyIdentity)
					r.Delete("/", g.DeleteIdentity)
				})
			})
			r.Route("/vacation", func(r chi.Router) {
				r.Get("/", g.GetVacation)
				r.Put("/", g.SetVacation)
			})
			r.Route("/quota", func(r chi.Router) {
				r.Get("/", g.GetQuota)
			})
			r.Route("/mailboxes", func(r chi.Router) {
				r.Get("/", g.GetMailboxes) // ?name=&role=&subcribed=
				r.Post("/", g.CreateMailbox)
				r.Route("/{mailboxid}", func(r chi.Router) {
					r.Get("/", g.GetMailboxById)
					r.Patch("/", g.ModifyMailbox)
					r.Delete("/", g.DeleteMailbox)
					r.Get("/emails", g.GetAllEmailsInMailbox)
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
					report(r, "/", g.RelatedToEmail)
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
				r.Post("/", g.UploadBlob)
			})
			r.Route("/ical", func(r chi.Router) {
				r.Get("/{blobid}", g.ParseIcalBlob)
			})
			r.Route("/addressbooks", func(r chi.Router) {
				r.Get("/", g.GetAddressbooks)
				r.Post("/", g.CreateAddressBook)
				r.Route("/{addressbookid}", func(r chi.Router) {
					r.Get("/", g.GetAddressbookById)
					r.Patch("/", g.ModifyAddressBook)
					r.Delete("/", g.DeleteAddressBook)
					r.Get("/contacts", g.GetContactsInAddressbook) //NOSONAR
				})
			})
			r.Route("/contacts", func(r chi.Router) {
				r.Get("/", g.GetAllContacts)
				r.Post("/", g.CreateContact)
				r.Route("/{contactid}", func(r chi.Router) {
					r.Get("/", g.GetContactById)
					r.Patch("/", g.ModifyContact)
					r.Delete("/", g.DeleteContact)
				})
			})
			r.Route("/calendars", func(r chi.Router) {
				r.Get("/", g.GetCalendars)
				r.Post("/", g.CreateCalendar)
				r.Route("/{calendarid}", func(r chi.Router) {
					r.Get("/", g.GetCalendarById)
					r.Patch("/", g.ModifyCalendar)
					r.Delete("/", g.DeleteCalendar)
					r.Get("/events", g.GetEventsInCalendar) //NOSONAR
				})
			})
			r.Route("/events", func(r chi.Router) {
				r.Get("/", g.GetAllEvents)
				r.Post("/", g.CreateEvent)
				r.Route("/{eventid}", func(r chi.Router) {
					r.Get("/", g.GetEventById)
					r.Patch("/", g.ModifyEvent)
					r.Delete("/", g.DeleteEvent)
				})
			})
			r.Route("/tasklists", func(r chi.Router) {
				r.Get("/", g.GetTaskLists)
				r.Route("/{tasklistid}", func(r chi.Router) {
					r.Get("/", g.GetTaskListById)
					r.Get("/tasks", g.GetTasksInTaskList)
				})
			})
			r.Route("/changes", func(r chi.Router) {
				r.Get("/", g.GetChanges)
				r.Get("/mailboxes", g.GetMailboxChanges)
				r.Get("/emails", g.GetEmailChanges)
				r.Get("/addressbooks", g.GetAddressBookChanges)
				r.Get("/contacts", g.GetContactsChanges)
				r.Get("/calendars", g.GetCalendarChanges)
				r.Get("/events", g.GetEventChanges)
				// r.Get("/quotas", g.GetQuotaChanges)
				r.Get("/identities", g.GetIdentityChanges)
			})
			r.Route("/objects", func(r chi.Router) {
				r.Get("/", g.GetObjects)
				r.Post("/", g.GetObjects) // this is actually a read-only operation
			})
		})
	})

	r.HandleFunc("/events/{stream}", g.ServeSSE)

	r.NotFound(g.NotFound)
	r.MethodNotAllowed(g.MethodNotAllowed)
}

func report(r chi.Router, pattern string, h http.HandlerFunc) {
	r.MethodFunc("REPORT", pattern, h)
}
