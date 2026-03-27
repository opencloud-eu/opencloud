package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

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
func (g *Groupware) GetChanges(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(logAccountId, accountId)

		var maxChanges uint = 0
		if v, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0); err != nil {
			return req.error(accountId, err)
		} else if ok {
			maxChanges = v
			l = l.Uint(QueryParamMaxChanges, v)
		}

		sinceState := jmap.StateMap{}
		{
			if state, ok := req.getStringParam(QueryParamMailboxes, ""); ok {
				sinceState.Mailboxes = ptr(toState(state))
			}
			if state, ok := req.getStringParam(QueryParamEmails, ""); ok {
				sinceState.Emails = ptr(toState(state))
			}
			if state, ok := req.getStringParam(QueryParamAddressbooks, ""); ok {
				sinceState.Addressbooks = ptr(toState(state))
			}
			if state, ok := req.getStringParam(QueryParamContacts, ""); ok {
				sinceState.Contacts = ptr(toState(state))
			}
			if state, ok := req.getStringParam(QueryParamCalendars, ""); ok {
				sinceState.Calendars = ptr(toState(state))
			}
			if state, ok := req.getStringParam(QueryParamEvents, ""); ok {
				sinceState.Events = ptr(toState(state))
			}
			if state, ok := req.getStringParam(QueryParamIdentities, ""); ok {
				sinceState.Identities = ptr(toState(state))
			}
			if state, ok := req.getStringParam(QueryParamEmailSubmissions, ""); ok {
				sinceState.EmailSubmissions = ptr(toState(state))
			}
			//if state, ok := req.getStringParam(QueryParamQuotas, ""); ok { sinceState.Quotas = ptr(toState(state)) }
			if sinceState.IsZero() {
				return req.noop(accountId)
			}
		}

		logger := log.From(l)
		changes, sessionState, state, lang, jerr := g.jmap.GetChanges(accountId, req.session, req.ctx, logger, req.language(), sinceState, maxChanges)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		var body jmap.Changes = changes

		return req.respond(accountId, body, sessionState, "", state)
	})
}
