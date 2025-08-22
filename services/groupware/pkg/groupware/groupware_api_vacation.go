package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

// When the request succeeds.
// swagger:response GetVacationResponse200
type SwaggerGetVacationResponse200 struct {
	// in: body
	Body struct {
		*jmap.VacationResponseGetResponse
	}
}

// swagger:route GET /accounts/{account}/vacation vacation getvacation
// Get vacation notice information.
//
// A vacation response sends an automatic reply when a message is delivered to the mail store, informing the original
// sender that their message may not be read for some time.
//
// The VacationResponse object represents the state of vacation-response-related settings for an account.
//
// responses:
//
//	200: GetVacationResponse200
//	400: ErrorResponse400
//	500: ErrorResponse500
func (g *Groupware) GetVacation(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		res, err := g.jmap.GetVacationResponse(req.GetAccountId(), req.session, req.ctx, req.logger)
		if err != nil {
			return req.errorResponseFromJmap(err)
		}
		return response(res, res.State)
	})
}

// When the request succeeds.
// swagger:response SetVacationResponse200
type SwaggerSetVacationResponse200 struct {
	// in: body
	Body struct {
		*jmap.VacationResponseChange
	}
}

// swagger:route PUT /accounts/{account}/vacation vacation setvacation
// Set the vacation notice information.
//
// A vacation response sends an automatic reply when a message is delivered to the mail store, informing the original
// sender that their message may not be read for some time.
//
// responses:
//
//	200: SetVacationResponse200
//	400: ErrorResponse400
//	500: ErrorResponse500
func (g *Groupware) SetVacation(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		var body jmap.VacationResponsePayload
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		res, jerr := g.jmap.SetVacationResponse(req.GetAccountId(), body, req.session, req.ctx, req.logger)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		return response(res, res.SessionState)
	})
}
