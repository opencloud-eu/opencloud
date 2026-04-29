package jmap

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
)

// Note that Quota/changes is currently not supported in Stalwart, as it always gives a
// cannotCalculateChanges error back.

var NS_CHANGES = ns(JmapMail, JmapContacts, JmapCalendars) //, JmapQuota)

type ObjectChanges struct {
	MaxChanges       uint                            `json:"maxchanges,omitzero"`
	Mailboxes        *MailboxChangesResponse         `json:"mailboxes,omitempty"`
	Emails           *EmailChangesResponse           `json:"emails,omitempty"`
	Calendars        *CalendarChangesResponse        `json:"calendars,omitempty"`
	Events           *CalendarEventChangesResponse   `json:"events,omitempty"`
	Addressbooks     *AddressBookChangesResponse     `json:"addressbooks,omitempty"`
	Contacts         *ContactCardChangesResponse     `json:"contacts,omitempty"`
	Identities       *IdentityChangesResponse        `json:"identities,omitempty"`
	EmailSubmissions *EmailSubmissionChangesResponse `json:"submissions,omitempty"`
	// Quotas       *QuotaChangesResponse         `json:"quotas,omitempty"`
}

type StateMap struct {
	Mailboxes        *State `json:"mailboxes,omitempty"`
	Emails           *State `json:"emails,omitempty"`
	Calendars        *State `json:"calendars,omitempty"`
	Events           *State `json:"events,omitempty"`
	Addressbooks     *State `json:"addressbooks,omitempty"`
	Contacts         *State `json:"contacts,omitempty"`
	Identities       *State `json:"identities,omitempty"`
	EmailSubmissions *State `json:"submissions,omitempty"`
	// Quotas       *State `json:"quotas,omitempty"`
}

var _ zerolog.LogObjectMarshaler = StateMap{}

func (s StateMap) IsZero() bool {
	return s.Mailboxes == nil && s.Emails == nil && s.Calendars == nil &&
		s.Events == nil && s.Addressbooks == nil && s.Contacts == nil &&
		s.Identities == nil && s.EmailSubmissions == nil
	//s.Quotas == nil
}

func (s StateMap) MarshalZerologObject(e *zerolog.Event) {
	if s.Mailboxes != nil {
		e.Str("mailboxes", string(*s.Mailboxes))
	}
	if s.Emails != nil {
		e.Str("emails", string(*s.Emails))
	}
	if s.Calendars != nil {
		e.Str("calendars", string(*s.Calendars))
	}
	if s.Events != nil {
		e.Str("events", string(*s.Events))
	}
	if s.Addressbooks != nil {
		e.Str("addressbooks", string(*s.Addressbooks))
	}
	if s.Contacts != nil {
		e.Str("contacts", string(*s.Contacts))
	}
	if s.Identities != nil {
		e.Str("identities", string(*s.Identities))
	}
	if s.EmailSubmissions != nil {
		e.Str("submissions", string(*s.EmailSubmissions))
	}
	// if s.Quotas != nil { e.Str("quotas", string(*s.Quotas)) }
}

// Retrieve the changes in any type of objects at once since a given State.
// @api:tags changes
func (j *Client) GetChanges(accountId string, stateMap StateMap, maxChanges uint, ctx Context) (Result[ObjectChanges], Error) { //NOSONAR
	logger := log.From(j.logger("GetChanges", ctx).With().Object("state", stateMap).Uint("maxChanges", maxChanges))
	ctx = ctx.WithLogger(logger)

	methodCalls := []Invocation{}
	if stateMap.Mailboxes != nil {
		methodCalls = append(methodCalls, invocation(MailboxChangesCommand{AccountId: accountId, SinceState: *stateMap.Mailboxes, MaxChanges: uintPtr(maxChanges)}, "mailboxes"))
	}
	if stateMap.Emails != nil {
		methodCalls = append(methodCalls, invocation(EmailChangesCommand{AccountId: accountId, SinceState: *stateMap.Emails, MaxChanges: uintPtr(maxChanges)}, "emails"))
	}
	if stateMap.Calendars != nil {
		methodCalls = append(methodCalls, invocation(CalendarChangesCommand{AccountId: accountId, SinceState: *stateMap.Calendars, MaxChanges: uintPtr(maxChanges)}, "calendars"))
	}
	if stateMap.Events != nil {
		methodCalls = append(methodCalls, invocation(CalendarEventChangesCommand{AccountId: accountId, SinceState: *stateMap.Events, MaxChanges: uintPtr(maxChanges)}, "events"))
	}
	if stateMap.Addressbooks != nil {
		methodCalls = append(methodCalls, invocation(AddressBookChangesCommand{AccountId: accountId, SinceState: *stateMap.Addressbooks, MaxChanges: uintPtr(maxChanges)}, "addressbooks"))
	}
	if stateMap.Contacts != nil {
		methodCalls = append(methodCalls, invocation(ContactCardChangesCommand{AccountId: accountId, SinceState: *stateMap.Contacts, MaxChanges: uintPtr(maxChanges)}, "contacts"))
	}
	if stateMap.Identities != nil {
		methodCalls = append(methodCalls, invocation(IdentityChangesCommand{AccountId: accountId, SinceState: *stateMap.Identities, MaxChanges: uintPtr(maxChanges)}, "identities"))
	}
	if stateMap.EmailSubmissions != nil {
		methodCalls = append(methodCalls, invocation(EmailSubmissionChangesCommand{AccountId: accountId, SinceState: *stateMap.EmailSubmissions, MaxChanges: uintPtr(maxChanges)}, "submissions"))
	}
	// if stateMap.Quotas != nil { methodCalls = append(methodCalls, invocation(CommandQuotaChanges, QuotaChangesCommand{AccountId: accountId, SinceState: *stateMap.Quotas, MaxChanges: posUIntPtr(maxChanges)}, "quotas")) }

	cmd, err := j.request(ctx, NS_CHANGES, methodCalls...)
	if err != nil {
		return ZeroResult[ObjectChanges](), err
	}

	return command(j, ctx, cmd, func(body *Response) (ObjectChanges, State, Error) {
		changes := ObjectChanges{
			MaxChanges: maxChanges,
		}
		states := map[string]State{}

		var mailboxes MailboxChangesResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandMailboxChanges, "mailboxes", &mailboxes); err != nil {
			return ObjectChanges{}, "", err
		} else if ok {
			changes.Mailboxes = &mailboxes
			states["mailbox"] = mailboxes.NewState
		}

		var emails EmailChangesResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandEmailChanges, "emails", &emails); err != nil {
			return ObjectChanges{}, "", err
		} else if ok {
			changes.Emails = &emails
			states["emails"] = emails.NewState
		}

		var calendars CalendarChangesResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandCalendarChanges, "calendars", &calendars); err != nil {
			return ObjectChanges{}, "", err
		} else if ok {
			changes.Calendars = &calendars
			states["calendars"] = calendars.NewState
		}

		var events CalendarEventChangesResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandCalendarEventChanges, "events", &events); err != nil {
			return ObjectChanges{}, "", err
		} else if ok {
			changes.Events = &events
			states["events"] = events.NewState
		}

		var addressbooks AddressBookChangesResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandAddressBookChanges, "addressbooks", &addressbooks); err != nil {
			return ObjectChanges{}, "", err
		} else if ok {
			changes.Addressbooks = &addressbooks
			states["addressbooks"] = addressbooks.NewState
		}

		var contacts ContactCardChangesResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandContactCardChanges, "contacts", &contacts); err != nil {
			return ObjectChanges{}, "", err
		} else if ok {
			changes.Contacts = &contacts
			states["contacts"] = contacts.NewState
		}

		var identities IdentityChangesResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandIdentityChanges, "identities", &identities); err != nil {
			return ObjectChanges{}, "", err
		} else if ok {
			changes.Identities = &identities
			states["identities"] = identities.NewState
		}

		var submissions EmailSubmissionChangesResponse
		if ok, err := tryRetrieveResponseMatchParameters(ctx, body, CommandEmailSubmissionChanges, "submissions", &submissions); err != nil {
			return ObjectChanges{}, "", err
		} else if ok {
			changes.EmailSubmissions = &submissions
			states["submissions"] = submissions.NewState
		}

		/*
			var quotas QuotaChangesResponse
			if ok, err := tryRetrieveResponseMatchParameters(logger, body, CommandQuotaChanges, "quotas", &quotas); err != nil {
				return Changes{}, "", err
			} else if ok {
				changes.Quotas = &quotas
				states["quotas"] = quotas.NewState
			}
		*/

		return changes, squashKeyedStates(states), nil
	})
}
