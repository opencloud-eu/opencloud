package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Get vacation notice information.
//
// A vacation response sends an automatic reply when a message is delivered to the mail store, informing the original
// sender that their message may not be read for some time.
//
// The VacationResponse object represents the state of vacation-response-related settings for an account.
func (g *Groupware) GetVacation(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForVacationResponse()
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		res, sessionState, state, lang, jerr := g.jmap.GetVacationResponse(accountId, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, res, sessionState, VacationResponseResponseObjectType, state)
	})
}

// Set the vacation notice information.
//
// A vacation response sends an automatic reply when a message is delivered to the mail store, informing the original
// sender that their message may not be read for some time.
func (g *Groupware) SetVacation(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForVacationResponse()
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		var body jmap.VacationResponsePayload
		err = req.body(&body)
		if err != nil {
			return req.error(accountId, err)
		}

		res, sessionState, state, lang, jerr := g.jmap.SetVacationResponse(accountId, body, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, res, sessionState, VacationResponseResponseObjectType, state)
	})
}
