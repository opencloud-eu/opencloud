package jmap

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

const (
	vacationResponseId = "singleton"
)

// https://jmap.io/spec-mail.html#vacationresponseget
func (j *Client) GetVacationResponse(accountId string, session *Session, ctx context.Context, logger *log.Logger) (VacationResponseGetResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetVacationResponse", session, logger)
	cmd, err := request(invocation(VacationResponseGet, VacationResponseGetCommand{AccountId: aid}, "0"))
	if err != nil {
		return VacationResponseGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (VacationResponseGetResponse, Error) {
		var response VacationResponseGetResponse
		err = retrieveResponseMatchParameters(body, VacationResponseGet, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

type VacationResponseStatusChange struct {
	VacationResponse VacationResponse `json:"vacationResponse"`
	ResponseState    string           `json:"state"`
	SessionState     string           `json:"sessionState"`
}

func (j *Client) SetVacationResponseStatus(accountId string, enabled bool, session *Session, ctx context.Context, logger *log.Logger) (VacationResponseStatusChange, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "EnableVacationResponse", session, logger)

	cmd, err := request(invocation(VacationResponseSet, VacationResponseSetRequest{
		AccountId: aid,
		Update: map[string]PatchObject{
			"u": {
				"/isEnabled": enabled,
			},
		},
	}, "0"))

	if err != nil {
		return VacationResponseStatusChange{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (VacationResponseStatusChange, Error) {
		var response VacationResponseSetResponse
		err = retrieveResponseMatchParameters(body, VacationResponseSet, "0", &response)
		if err != nil {
			return VacationResponseStatusChange{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		updated, ok := response.Updated["u"]
		if !ok {
			// TODO implement error when not updated
		}

		return VacationResponseStatusChange{
			VacationResponse: updated,
			ResponseState:    response.NewState,
			SessionState:     response.State,
		}, nil
	})
}

type VacationResponseBody struct {
	// Should a vacation response be sent if a message arrives between the "fromDate" and "toDate"?
	IsEnabled bool `json:"isEnabled"`
	// If "isEnabled" is true, messages that arrive on or after this date-time (but before the "toDate" if defined) should receive the
	// user's vacation response. If null, the vacation response is effective immediately.
	FromDate time.Time `json:"fromDate,omitzero"`
	// If "isEnabled" is true, messages that arrive before this date-time but on or after the "fromDate" if defined) should receive the
	// user's vacation response.  If null, the vacation response is effective indefinitely.
	ToDate time.Time `json:"toDate,omitzero"`
	// The subject that will be used by the message sent in response to messages when the vacation response is enabled.
	// If null, an appropriate subject SHOULD be set by the server.
	Subject string `json:"subject,omitempty"`
	// The plaintext body to send in response to messages when the vacation response is enabled.
	// If this is null, the server SHOULD generate a plaintext body part from the "htmlBody" when sending vacation responses
	// but MAY choose to send the response as HTML only.  If both "textBody" and "htmlBody" are null, an appropriate default
	// body SHOULD be generated for responses by the server.
	TextBody string `json:"textBody,omitempty"`
	// The HTML body to send in response to messages when the vacation response is enabled.
	// If this is null, the server MAY choose to generate an HTML body part from the "textBody" when sending vacation responses
	// or MAY choose to send the response as plaintext only.
	HtmlBody string `json:"htmlBody,omitempty"`
}

type VacationResponseChange struct {
	VacationResponse VacationResponse `json:"vacationResponse"`
	ResponseState    string           `json:"state"`
	SessionState     string           `json:"sessionState"`
}

func (j *Client) SetVacationResponse(accountId string, vacation VacationResponseBody, session *Session, ctx context.Context, logger *log.Logger) (VacationResponseChange, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "SetVacationResponse", session, logger)

	set := VacationResponseSetRequest{
		AccountId: aid,
		Create: map[string]VacationResponse{
			vacationResponseId: {
				IsEnabled: vacation.IsEnabled,
				FromDate:  vacation.FromDate,
				ToDate:    vacation.ToDate,
				Subject:   vacation.Subject,
				TextBody:  vacation.TextBody,
				HtmlBody:  vacation.HtmlBody,
			},
		},
	}

	cmd, err := request(invocation(VacationResponseSet, set, "0"))
	if err != nil {
		return VacationResponseChange{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (VacationResponseChange, Error) {
		var response VacationResponseSetResponse
		err = retrieveResponseMatchParameters(body, VacationResponseSet, "0", &response)
		if err != nil {
			return VacationResponseChange{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		created, ok := response.Created[vacationResponseId]
		if !ok {
			// TODO handle case where created is missing
		}

		return VacationResponseChange{
			VacationResponse: created,
			ResponseState:    response.NewState,
			SessionState:     response.State,
		}, nil
	})
}
