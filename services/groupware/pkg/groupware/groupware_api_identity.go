package groupware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

// When the request suceeds.
// swagger:response GetIdentitiesResponse
type SwaggerGetIdentitiesResponse struct {
	// in: body
	Body []jmap.Identity
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
			return errorResponse(single(accountId), err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))
		res, sessionState, state, lang, jerr := g.jmap.GetAllIdentities(accountId, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}
		return etagResponse(single(accountId), res, sessionState, IdentityResponseObjectType, state, lang)
	})
}

func (g *Groupware) GetIdentityById(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		id := chi.URLParam(r, UriParamIdentityId)
		logger := log.From(req.logger.With().Str(logAccountId, accountId).Str(logIdentityId, id))
		res, sessionState, state, lang, jerr := g.jmap.GetIdentities(accountId, req.session, req.ctx, logger, req.language(), []string{id})
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}
		if len(res) < 1 {
			return notFoundResponse(single(accountId), sessionState)
		}
		return etagResponse(single(accountId), res[0], sessionState, IdentityResponseObjectType, state, lang)
	})
}

func (g *Groupware) AddIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		var identity jmap.Identity
		err = req.body(&identity)
		if err != nil {
			return errorResponse(single(accountId), err)
		}

		created, sessionState, state, lang, jerr := g.jmap.CreateIdentity(accountId, req.session, req.ctx, logger, req.language(), identity)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}
		return etagResponse(single(accountId), created, sessionState, IdentityResponseObjectType, state, lang)
	})
}

func (g *Groupware) ModifyIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		var identity jmap.Identity
		err = req.body(&identity)
		if err != nil {
			return errorResponse(single(accountId), err)
		}

		updated, sessionState, state, lang, jerr := g.jmap.UpdateIdentity(accountId, req.session, req.ctx, logger, req.language(), identity)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}
		return etagResponse(single(accountId), updated, sessionState, IdentityResponseObjectType, state, lang)
	})
}

func (g *Groupware) DeleteIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		id := chi.URLParam(r, UriParamIdentityId)
		ids := strings.Split(id, ",")
		if len(ids) < 1 {
			return req.parameterErrorResponse(single(accountId), UriParamEmailId, fmt.Sprintf("Invalid value for path parameter '%v': '%s': %s", UriParamIdentityId, log.SafeString(id), "empty list of identity ids"))
		}

		deletion, sessionState, state, _, jerr := g.jmap.DeleteIdentity(accountId, req.session, req.ctx, logger, req.language(), ids)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		notDeletedIds := structs.Missing(ids, deletion)
		if len(notDeletedIds) == 0 {
			return noContentResponseWithEtag(single(accountId), sessionState, IdentityResponseObjectType, state)
		} else {
			logger.Error().Array("not-deleted", log.SafeStringArray(notDeletedIds)).Msgf("failed to delete %d identities", len(notDeletedIds))
			return errorResponseWithSessionState(single(accountId), req.apiError(&ErrorFailedToDeleteSomeIdentities,
				withMeta(map[string]any{"ids": notDeletedIds})), sessionState)
		}
	})
}
