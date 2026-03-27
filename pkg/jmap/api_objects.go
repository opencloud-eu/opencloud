package jmap

import (
	"context"

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

func (j *Client) GetObjects(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, //NOSONAR
	mailboxIds []string, emailIds []string,
	addressbookIds []string, contactIds []string,
	calendarIds []string, eventIds []string,
	quotaIds []string, identityIds []string,
	emailSubmissionIds []string,
) (Objects, SessionState, State, Language, Error) {
	l := j.logger("GetObjects", session, logger).With()
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
	logger = log.From(l)

	methodCalls := []Invocation{}
	if len(mailboxIds) > 0 {
		methodCalls = append(methodCalls, invocation(CommandMailboxGet, MailboxGetCommand{AccountId: accountId, Ids: mailboxIds}, "mailboxes"))
	}
	if len(emailIds) > 0 {
		methodCalls = append(methodCalls, invocation(CommandEmailGet, EmailGetCommand{AccountId: accountId, Ids: emailIds}, "emails"))
	}
	if len(addressbookIds) > 0 {
		methodCalls = append(methodCalls, invocation(CommandAddressBookGet, AddressBookGetCommand{AccountId: accountId, Ids: addressbookIds}, "addressbooks"))
	}
	if len(contactIds) > 0 {
		methodCalls = append(methodCalls, invocation(CommandContactCardGet, ContactCardGetCommand{AccountId: accountId, Ids: contactIds}, "contacts"))
	}
	if len(calendarIds) > 0 {
		methodCalls = append(methodCalls, invocation(CommandCalendarGet, CalendarGetCommand{AccountId: accountId, Ids: calendarIds}, "calendars"))
	}
	if len(eventIds) > 0 {
		methodCalls = append(methodCalls, invocation(CommandCalendarEventGet, CalendarEventGetCommand{AccountId: accountId, Ids: eventIds}, "events"))
	}
	if len(quotaIds) > 0 {
		methodCalls = append(methodCalls, invocation(CommandQuotaGet, QuotaGetCommand{AccountId: accountId, Ids: quotaIds}, "quotas"))
	}
	if len(identityIds) > 0 {
		methodCalls = append(methodCalls, invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId, Ids: identityIds}, "identities"))
	}
	if len(emailSubmissionIds) > 0 {
		methodCalls = append(methodCalls, invocation(CommandEmailSubmissionGet, EmailSubmissionGetCommand{AccountId: accountId, Ids: emailSubmissionIds}, "emailSubmissionIds"))
	}

	cmd, err := j.request(session, logger, NS_OBJECTS, methodCalls...)
	if err != nil {
		return Objects{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (Objects, State, Error) {
		objs := Objects{}
		states := map[string]State{}

		var mailboxes MailboxGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandMailboxGet, "mailboxes", &mailboxes); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Mailboxes = &mailboxes
			states["mailbox"] = mailboxes.State
		}

		var emails EmailGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandEmailGet, "emails", &emails); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Emails = &emails
			states["email"] = emails.State
		}

		var calendars CalendarGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandCalendarGet, "calendars", &calendars); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Calendars = &calendars
			states["calendar"] = calendars.State
		}

		var events CalendarEventGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandCalendarEventGet, "events", &events); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Events = &events
			states["event"] = events.State
		}

		var addressbooks AddressBookGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandAddressBookGet, "addressbooks", &addressbooks); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Addressbooks = &addressbooks
			states["addressbook"] = addressbooks.State
		}

		var contacts ContactCardGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandContactCardGet, "contacts", &contacts); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Contacts = &contacts
			states["contact"] = contacts.State
		}

		var quotas QuotaGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandQuotaGet, "quotas", &quotas); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Quotas = &quotas
			states["quota"] = quotas.State
		}

		var identities IdentityGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandIdentityGet, "identities", &identities); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.Identities = &identities
			states["identity"] = identities.State
		}

		var submissions EmailSubmissionGetResponse
		if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandEmailSubmissionGet, "submissions", &submissions); err != nil {
			return Objects{}, "", err
		} else if ok {
			objs.EmailSubmissions = &submissions
			states["submissions"] = submissions.State
		}

		return objs, squashKeyedStates(states), nil
	})
}
