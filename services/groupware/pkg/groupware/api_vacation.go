package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

// Get vacation notice information.
//
// A vacation response sends an automatic reply when a message is delivered to the mail store, informing the original
// sender that their message may not be read for some time.
//
// The VacationResponse object represents the state of vacation-response-related settings for an account.
func (g *Groupware) GetVacation(w http.ResponseWriter, r *http.Request) {
	get(VacationResponse, w, r, g, func(accountId string, ids []string, ctx jmap.Context) (jmap.Result[jmap.VacationResponseGetResponse], jmap.Error) {
		return g.jmap.GetVacationResponse(accountId, ctx)
	})
}

// Set the vacation notice information.
//
// A vacation response sends an automatic reply when a message is delivered to the mail store, informing the original
// sender that their message may not be read for some time.
func (g *Groupware) SetVacation(w http.ResponseWriter, r *http.Request) {
	modify(VacationResponse, w, r, g, func(accountId string, id string, change jmap.VacationResponseChange, ctx jmap.Context) (jmap.Result[jmap.VacationResponse], jmap.Error) {
		return g.jmap.SetVacationResponse(accountId, change, ctx)
	})
}
