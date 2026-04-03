package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

type ObjectsRequest struct {
	Mailboxes        []string `json:"mailboxes,omitempty"`
	Emails           []string `json:"emails,omitempty"`
	Addressbooks     []string `json:"addressbooks,omitempty"`
	Contacts         []string `json:"contacts,omitempty"`
	Calendars        []string `json:"calendars,omitempty"`
	Events           []string `json:"events,omitempty"`
	Quotas           []string `json:"quotas,omitempty"`
	Identities       []string `json:"identities,omitempty"`
	EmailSubmissions []string `json:"submissions,omitempty"`
}

// Retrieve changes for multiple or all Groupware objects, based on their respective state token.
//
// Since each object type has its own state token, the request must include the token for each
// object type separately.
//
// This is done through individual query parameter, as follows:
// `?emails=rafrag&contacts=rbsxqeay&events=n`
//
// Additionally, the `maxchanges` query parameter may be used to limit the number of changes
// to retrieve for each object type -- this is applied to each object type, not overall.
//
// If `maxchanges` is not specifed or if `maxchanges` has the value `0`, then there is no limit
// and all the changes from the specified state to now are included in the result.
//
// The response then includes the new state after that maximum number if changes,
// as well as a `hasMoreChanges` boolean flag which can be used to paginate the retrieval of
// changes and the objects associated with the identifiers.
//
// @api:tags mailbox,email,addressbook,contact,calendar,event,quota,identity
func (g *Groupware) GetObjects(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(logAccountId, accountId)

		mailboxIds := []string{}
		emailIds := []string{}
		addressbookIds := []string{}
		contactIds := []string{}
		calendarIds := []string{}
		eventIds := []string{}
		quotaIds := []string{}
		identityIds := []string{}
		emailSubmissionIds := []string{}
		{
			var objects ObjectsRequest
			if ok, err := req.optBody(&objects); err != nil {
				return req.error(accountId, err)
			} else if ok {
				mailboxIds = append(mailboxIds, objects.Mailboxes...)
				emailIds = append(emailIds, objects.Emails...)
				addressbookIds = append(addressbookIds, objects.Addressbooks...)
				contactIds = append(contactIds, objects.Contacts...)
				calendarIds = append(calendarIds, objects.Calendars...)
				eventIds = append(eventIds, objects.Events...)
				quotaIds = append(quotaIds, objects.Quotas...)
				identityIds = append(identityIds, objects.Identities...)
				emailSubmissionIds = append(emailSubmissionIds, objects.EmailSubmissions...)
			}
		}

		if list, ok, err := req.parseOptStringListParam(QueryParamMailboxes); err != nil {
			return req.error(accountId, err)
		} else if ok {
			mailboxIds = append(mailboxIds, list...)
		}
		if list, ok, err := req.parseOptStringListParam(QueryParamEmails); err != nil {
			return req.error(accountId, err)
		} else if ok {
			emailIds = append(emailIds, list...)
		}
		if list, ok, err := req.parseOptStringListParam(QueryParamAddressbooks); err != nil {
			return req.error(accountId, err)
		} else if ok {
			addressbookIds = append(addressbookIds, list...)
		}
		if list, ok, err := req.parseOptStringListParam(QueryParamContacts); err != nil {
			return req.error(accountId, err)
		} else if ok {
			contactIds = append(contactIds, list...)
		}
		if list, ok, err := req.parseOptStringListParam(QueryParamCalendars); err != nil {
			return req.error(accountId, err)
		} else if ok {
			calendarIds = append(calendarIds, list...)
		}
		if list, ok, err := req.parseOptStringListParam(QueryParamEvents); err != nil {
			return req.error(accountId, err)
		} else if ok {
			eventIds = append(eventIds, list...)
		}
		if list, ok, err := req.parseOptStringListParam(QueryParamQuotas); err != nil {
			return req.error(accountId, err)
		} else if ok {
			quotaIds = append(quotaIds, list...)
		}
		if list, ok, err := req.parseOptStringListParam(QueryParamIdentities); err != nil {
			return req.error(accountId, err)
		} else if ok {
			identityIds = append(identityIds, list...)
		}
		if list, ok, err := req.parseOptStringListParam(QueryParamEmailSubmissions); err != nil {
			return req.error(accountId, err)
		} else if ok {
			emailSubmissionIds = append(emailSubmissionIds, list...)
		}

		logger := log.From(l)
		objs, sessionState, state, lang, jerr := g.jmap.GetObjects(accountId, req.session, req.ctx, logger, req.language(),
			mailboxIds, emailIds, addressbookIds, contactIds, calendarIds, eventIds, quotaIds, identityIds, emailSubmissionIds)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		var body jmap.Objects = objs

		return req.respond(accountId, body, sessionState, "", state)
	})
}
