package groupware

import (
	"net/http"
)

func (g Groupware) GetAccount(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) (any, string, *Error) {
		account, ok := req.GetAccount()
		if !ok {
			return nil, "", nil
		}
		return account, req.session.State, nil
	})
}
