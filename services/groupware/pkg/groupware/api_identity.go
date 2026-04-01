package groupware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

// Get the list of identities that are associated with an account.
func (g *Groupware) GetIdentities(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))
		res, sessionState, state, lang, jerr := g.jmap.GetAllIdentities(accountId, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, res, sessionState, IdentityResponseObjectType, state)
	})
}

func (g *Groupware) GetIdentityById(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		id, err := req.PathParam(UriParamIdentityId)
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId).Str(logIdentityId, id))
		res, sessionState, state, lang, jerr := g.jmap.GetIdentities(accountId, req.session, req.ctx, logger, req.language(), []string{id})
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		if len(res) < 1 {
			return req.notFound(accountId, sessionState, IdentityResponseObjectType, state)
		}
		var body jmap.Identity = res[0]
		return req.respond(accountId, body, sessionState, IdentityResponseObjectType, state)
	})
}

func (g *Groupware) AddIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		var identity jmap.Identity
		err = req.body(&identity)
		if err != nil {
			return req.error(accountId, err)
		}

		created, sessionState, state, lang, jerr := g.jmap.CreateIdentity(accountId, req.session, req.ctx, logger, req.language(), identity)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, created, sessionState, IdentityResponseObjectType, state)
	})
}

func (g *Groupware) ModifyIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		var identity jmap.Identity
		err = req.body(&identity)
		if err != nil {
			return req.error(accountId, err)
		}

		updated, sessionState, state, lang, jerr := g.jmap.UpdateIdentity(accountId, req.session, req.ctx, logger, req.language(), identity)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, updated, sessionState, IdentityResponseObjectType, state)
	})
}

// Delete an identity.
func (g *Groupware) DeleteIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		id, err := req.PathParam(UriParamIdentityId)
		if err != nil {
			return req.error(accountId, err)
		}
		ids := strings.Split(id, ",")
		if len(ids) < 1 {
			return req.parameterErrorResponse(single(accountId), UriParamIdentityId, fmt.Sprintf("Invalid value for path parameter '%v': '%s': %s", UriParamIdentityId, log.SafeString(id), "empty list of identity ids"))
		}

		deletion, sessionState, state, lang, jerr := g.jmap.DeleteIdentity(accountId, req.session, req.ctx, logger, req.language(), ids)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		notDeletedIds := structs.Missing(ids, deletion)
		if len(notDeletedIds) == 0 {
			return req.noContent(accountId, sessionState, IdentityResponseObjectType, state)
		} else {
			logger.Error().Array("not-deleted", log.SafeStringArray(notDeletedIds)).Msgf("failed to delete %d identities", len(notDeletedIds))
			return req.errorS(accountId, req.apiError(&ErrorFailedToDeleteSomeIdentities,
				withMeta(map[string]any{"ids": notDeletedIds})), sessionState)
		}
	})
}

// Get changes to Identities since a given State
// @api:tags identity,changes
func (g *Groupware) GetIdentityChanges(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}

		l := req.logger.With().Str(logAccountId, accountId)

		var maxChanges uint = 0
		if v, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0); err != nil {
			return req.error(accountId, err)
		} else if ok {
			maxChanges = v
			l = l.Uint(QueryParamMaxChanges, v)
		}

		sinceState := jmap.State(req.OptHeaderParamDoc(HeaderParamSince, "Specifies the state identifier from which on to list identity changes"))
		l = l.Str(HeaderParamSince, log.SafeString(string(sinceState)))

		logger := log.From(l)
		changes, sessionState, state, lang, jerr := g.jmap.GetIdentityChanges(accountId, req.session, req.ctx, logger, req.language(), sinceState, maxChanges)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		var body jmap.IdentityChanges = changes

		return req.respond(accountId, body, sessionState, IdentityResponseObjectType, state)
	})
}
