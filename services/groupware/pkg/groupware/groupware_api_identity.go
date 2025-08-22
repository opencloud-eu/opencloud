package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
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
		res, err := g.jmap.GetIdentity(req.GetAccountId(), req.session, req.ctx, req.logger)
		if err != nil {
			return req.errorResponseFromJmap(err)
		}
		return response(res, res.State)
	})
}
