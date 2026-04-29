package jmap

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
)

var NS_OBJECTS = ns(JmapMail, JmapSubmission, JmapContacts, JmapCalendars, JmapQuota)

type Objects struct {
	Mailboxes        *MailboxGetResponse         `json:"mailboxes,omitempty"`
	Emails           *EmailGetResponse           `json:"emails,omitempty"`
	Calendars        *CalendarGetResponse        `json:"calendars,omitempty"`
	Events           *CalendarEventGetResponse   `json:"events,omitempty"`
	Addressbooks     *AddressBookGetResponse     `json:"addressbooks,omitempty"`
	Contacts         *ContactCardGetResponse     `json:"contacts,omitempty"`
	Quotas           *QuotaGetResponse           `json:"quotas,omitempty"`
	Identities       *IdentityGetResponse        `json:"identities,omitempty"`
	EmailSubmissions *EmailSubmissionGetResponse `json:"submissions,omitempty"`
}

// Retrieve objects of all types by their identifiers in a single batch.
// @api:tags changes
func (j *Client) GetObjects(accountId string, //NOSONAR
	mailboxIds []string, emailIds []string,
	addressbookIds []string, contactIds []string,
	calendarIds []string, eventIds []string,
	quotaIds []string, identityIds []string,
	emailSubmissionIds []string,
	ctx Context,
) (Result[Objects], Error) {
	l := j.logger("GetObjects", ctx).With()
	if len(mailboxIds) > 0 {
		l = l.Array("mailboxIds", log.SafeStringArray(mailboxIds))
	}
	if len(emailIds) > 0 {
		l = l.Array("emailIds", log.SafeStringArray(emailIds))
	}
	if len(addressbookIds) > 0 {
		l = l.Array("addressbookIds", log.SafeStringArray(addressbookIds))
	}
	if len(contactIds) > 0 {
		l = l.Array("contactIds", log.SafeStringArray(contactIds))
	}
	if len(calendarIds) > 0 {
		l = l.Array("calendarIds", log.SafeStringArray(calendarIds))
	}
	if len(eventIds) > 0 {
		l = l.Array("eventIds", log.SafeStringArray(eventIds))
	}
	if len(quotaIds) > 0 {
		l = l.Array("quotaIds", log.SafeStringArray(quotaIds))
	}
	if len(identityIds) > 0 {
		l = l.Array("identityIds", log.SafeStringArray(identityIds))
	}
	if len(emailSubmissionIds) > 0 {
		l = l.Array("emailSubmissionIds", log.SafeStringArray(emailSubmissionIds))
	}
	logger := log.From(l)
	ctx = ctx.WithLogger(logger)

	methodCalls := []Invocation{}
	if len(mailboxIds) > 0 {
		methodCalls = append(methodCalls, invocation(MailboxGetCommand{AccountId: accountId, Ids: mailboxIds}, "mailboxes"))
	}
	if len(emailIds) > 0 {
		methodCalls = append(methodCalls, invocation(EmailGetCommand{AccountId: accountId, Ids: emailIds}, "emails"))
	}
	if len(addressbookIds) > 0 {
		methodCalls = append(methodCalls, invocation(AddressBookGetCommand{AccountId: accountId, Ids: addressbookIds}, "addressbooks"))
	}
	if len(contactIds) > 0 {
		methodCalls = append(methodCalls, invocation(ContactCardGetCommand{AccountId: accountId, Ids: contactIds}, "contacts"))
	}
	if len(calendarIds) > 0 {
		methodCalls = append(methodCalls, invocation(CalendarGetCommand{AccountId: accountId, Ids: calendarIds}, "calendars"))
	}
	if len(eventIds) > 0 {
		methodCalls = append(methodCalls, invocation(CalendarEventGetCommand{AccountId: accountId, Ids: eventIds}, "events"))
	}
	if len(quotaIds) > 0 {
		methodCalls = append(methodCalls, invocation(QuotaGetCommand{AccountId: accountId, Ids: quotaIds}, "quotas"))
	}
	if len(identityIds) > 0 {
		methodCalls = append(methodCalls, invocation(IdentityGetCommand{AccountId: accountId, Ids: identityIds}, "identities"))
	}
	if len(emailSubmissionIds) > 0 {
		methodCalls = append(methodCalls, invocation(EmailSubmissionGetCommand{AccountId: accountId, Ids: emailSubmissionIds}, "emailSubmissionIds"))
	}

	cmd, err := j.request(ctx, NS_OBJECTS, methodCalls...)
	if err != nil {
		return ZeroResult[Objects](), err
	}

	return command(j, ctx, cmd, func(body *Response) (Objects, State, Error) {
		objs := Objects{}
		states := map[string]State{}

		var mailboxes MailboxGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandMailboxGet, "mailboxes", &mailboxes); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Mailboxes = &mailboxes
			states["mailbox"] = mailboxes.State
		}

		var emails EmailGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandEmailGet, "emails", &emails); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Emails = &emails
			states["email"] = emails.State
		}

		var calendars CalendarGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandCalendarGet, "calendars", &calendars); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Calendars = &calendars
			states["calendar"] = calendars.State
		}

		var events CalendarEventGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandCalendarEventGet, "events", &events); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Events = &events
			states["event"] = events.State
		}

		var addressbooks AddressBookGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandAddressBookGet, "addressbooks", &addressbooks); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Addressbooks = &addressbooks
			states["addressbook"] = addressbooks.State
		}

		var contacts ContactCardGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandContactCardGet, "contacts", &contacts); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Contacts = &contacts
			states["contact"] = contacts.State
		}

		var quotas QuotaGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandQuotaGet, "quotas", &quotas); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Quotas = &quotas
			states["quota"] = quotas.State
		}

		var identities IdentityGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandIdentityGet, "identities", &identities); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Identities = &identities
			states["identity"] = identities.State
		}

		var submissions EmailSubmissionGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandEmailSubmissionGet, "submissions", &submissions); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.EmailSubmissions = &submissions
			states["submissions"] = submissions.State
		}

		return objs, squashKeyedStates(states), nil
	})
}
