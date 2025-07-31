package groupware

import (
	"net/http"
)

func (g Groupware) GetAccount(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) (any, string, *Error) {
		account, err := req.GetAccount()
		if err != nil {
			return nil, "", err
		}
		return account, req.session.State, nil
	})
}

func (g Groupware) GetAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) (any, string, *Error) {
		return req.session.Accounts, req.session.State, nil
	})
}
