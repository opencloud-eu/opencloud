package groupware

import (
	"net/http"
)

func (g Groupware) GetIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) (any, string, *Error) {
		res, err := g.jmap.GetIdentity(req.session, req.ctx, req.logger)
		return res, res.State, apiErrorFromJmap(err)
	})
}
