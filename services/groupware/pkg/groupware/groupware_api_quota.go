package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// When the request succeeds.
// swagger:response GetQuotaResponse200
type SwaggerGetQuotaResponse200 struct {
	// in: body
	Body []jmap.Quota
}

// swagger:route GET /groupware/accounts/{account}/quota quota getquota
// Get quota limits.
//
// responses:
//
//	200: GetQuotaResponse200
//	400: ErrorResponse400
//	500: ErrorResponse500
func (g *Groupware) GetQuota(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForQuota()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		res, sessionState, lang, jerr := g.jmap.GetQuotas(accountId, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		return etagResponse(res.List, sessionState, res.State, lang)
	})
}
