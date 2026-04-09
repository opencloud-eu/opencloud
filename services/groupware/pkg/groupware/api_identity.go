package groupware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Get the list of identities that are associated with an account.
func (g *Groupware) GetIdentities(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))
		ctx := req.ctx.WithLogger(logger)
		res, sessionState, state, lang, jerr := g.jmap.GetAllIdentities(accountId, ctx)
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
		ctx := req.ctx.WithLogger(logger)
		res, sessionState, state, lang, jerr := g.jmap.GetIdentities(accountId, single(id), ctx)
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
		ctx := req.ctx.WithLogger(logger)

		var identity jmap.IdentityChange
		err = req.body(&identity)
		if err != nil {
			return req.error(accountId, err)
		}

		created, sessionState, state, lang, jerr := g.jmap.CreateIdentity(accountId, identity, ctx)
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
		id, err := req.PathParamDoc(UriParamIdentityId, "The unique identifier of the Identity to modify")
		if err != nil {
			return req.error(accountId, err)
		}

		logger := log.From(req.logger.With().Str(logAccountId, accountId).Str(UriParamIdentityId, log.SafeString(id)))
		ctx := req.ctx.WithLogger(logger)

		var identity jmap.IdentityChange
		err = req.body(&identity)
		if err != nil {
			return req.error(accountId, err)
		}

		updated, sessionState, state, lang, jerr := g.jmap.UpdateIdentity(accountId, id, identity, ctx)
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

		id, err := req.PathParam(UriParamIdentityId)
		if err != nil {
			return req.error(accountId, err)
		}
		ids := strings.Split(id, ",")
		if len(ids) < 1 {
			return req.parameterErrorResponse(single(accountId), UriParamIdentityId, fmt.Sprintf("Invalid value for path parameter '%v': '%s': %s", UriParamIdentityId, log.SafeString(id), "empty list of identity ids"))
		}

		logger := log.From(req.logger.With().Str(logAccountId, accountId).Array(UriParamIdentityId, log.SafeStringArray(ids)))
		ctx := req.ctx.WithLogger(logger)

		notDeleted, sessionState, state, lang, jerr := g.jmap.DeleteIdentity(accountId, ids, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if len(notDeleted) == 0 {
			return req.noContent(accountId, sessionState, IdentityResponseObjectType, state)
		} else {
			logger.Error().Msgf("failed to delete %d identities", len(notDeleted))
			return req.errorS(accountId, req.apiError(&ErrorFailedToDeleteSomeIdentities), sessionState)
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
		ctx := req.ctx.WithLogger(logger)
		changes, sessionState, state, lang, jerr := g.jmap.GetIdentityChanges(accountId, sinceState, maxChanges, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		var body jmap.IdentityChanges = changes

		return req.respond(accountId, body, sessionState, IdentityResponseObjectType, state)
	})
}
