package groupware

import (
	"net/http"

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

// swagger:route GET /accounts/{accountid}/identities identities identities
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
		accountId, err := req.GetAccountIdWithoutFallback()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))
		res, sessionState, jerr := g.jmap.GetIdentity(accountId, req.session, req.ctx, logger)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		return etagResponse(res, sessionState, res.State)
	})
}
