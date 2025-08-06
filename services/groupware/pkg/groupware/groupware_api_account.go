package groupware

import (
	"net/http"
)

func (g Groupware) GetAccount(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		account, err := req.GetAccount()
		if err != nil {
			return errorResponse(err)
		}
		return response(account, req.session.State)
	})
}

func (g Groupware) GetAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		return response(req.session.Accounts, req.session.State)
	})
}
