package groupware

import (
	"net/http"
)

func (g Groupware) GetIdentities(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		res, err := g.jmap.GetIdentity(req.GetAccountId(), req.session, req.ctx, req.logger)
		if err != nil {
			return req.errorResponseFromJmap(err)
		}
		return response(res, res.State)
	})
}
