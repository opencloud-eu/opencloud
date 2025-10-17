package groupware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// When the request suceeds.
// swagger:response GetIdentitiesResponse
type SwaggerGetIdentitiesResponse struct {
	// in: body
	Body struct {
		*jmap.Identities
	}
}

// swagger:route GET /groupware/accounts/{account}/identities identity identities
// Get the list of identities that are associated with an account.
//
// responses:
//
//	200: GetIdentitiesResponse
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetIdentities(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))
		res, sessionState, lang, jerr := g.jmap.GetAllIdentities(accountId, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		return etagResponse(res, sessionState, res.State, lang)
	})
}

func (g *Groupware) GetIdentityById(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdWithoutFallback()
		if err != nil {
			return errorResponse(err)
		}
		id := chi.URLParam(r, UriParamIdentityId)
		logger := log.From(req.logger.With().Str(logAccountId, accountId).Str(logIdentityId, id))
		res, sessionState, lang, jerr := g.jmap.GetIdentities(accountId, req.session, req.ctx, logger, req.language(), []string{id})
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		if len(res.Identities) < 1 {
			return notFoundResponse(sessionState)
		}
		return etagResponse(res.Identities[0], sessionState, res.State, lang)
	})
}

func (g *Groupware) AddIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdWithoutFallback()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		var identity jmap.Identity
		err = req.body(&identity)
		if err != nil {
			return errorResponse(err)
		}

		newState, sessionState, _, jerr := g.jmap.CreateIdentity(accountId, req.session, req.ctx, logger, req.language(), identity)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		return noContentResponseWithEtag(sessionState, newState)
	})
}

func (g *Groupware) ModifyIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdWithoutFallback()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		var identity jmap.Identity
		err = req.body(&identity)
		if err != nil {
			return errorResponse(err)
		}

		newState, sessionState, _, jerr := g.jmap.UpdateIdentity(accountId, req.session, req.ctx, logger, req.language(), identity)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		return noContentResponseWithEtag(sessionState, newState)
	})
}
