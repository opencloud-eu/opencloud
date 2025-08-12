package jmap

import (
	"context"
	"fmt"
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
	cmd, err := request(invocation(CommandVacationResponseGet, VacationResponseGetCommand{AccountId: aid}, "0"))
	if err != nil {
		return VacationResponseGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (VacationResponseGetResponse, Error) {
		var response VacationResponseGetResponse
		err = retrieveResponseMatchParameters(body, CommandVacationResponseGet, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

// Same as VacationResponse but without the id.
type VacationResponsePayload struct {
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

func (j *Client) SetVacationResponse(accountId string, vacation VacationResponsePayload, session *Session, ctx context.Context, logger *log.Logger) (VacationResponseChange, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "SetVacationResponse", session, logger)

	cmd, err := request(
		invocation(CommandVacationResponseSet, VacationResponseSetCommand{
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
		}, "0"),
		// chain a second request to get the current complete VacationResponse object
		// after performing the changes, as that makes for a better API
		invocation(CommandVacationResponseGet, VacationResponseGetCommand{AccountId: aid}, "1"),
	)
	if err != nil {
		return VacationResponseChange{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (VacationResponseChange, Error) {
		var setResponse VacationResponseSetResponse
		err = retrieveResponseMatchParameters(body, CommandVacationResponseSet, "0", &setResponse)
		if err != nil {
			return VacationResponseChange{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		setErr, notok := setResponse.NotCreated[vacationResponseId]
		if notok {
			// this means that the VacationResponse was not updated
			return VacationResponseChange{}, setErrorError(setErr, VacationResponseType)
		}

		var getResponse VacationResponseGetResponse
		err = retrieveResponseMatchParameters(body, CommandVacationResponseGet, "1", &getResponse)
		if err != nil {
			return VacationResponseChange{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		if len(getResponse.List) != 1 {
			err = fmt.Errorf("failed to find %s in %s response", string(VacationResponseType), string(CommandVacationResponseGet))
			return VacationResponseChange{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		return VacationResponseChange{
			VacationResponse: getResponse.List[0],
			ResponseState:    setResponse.NewState,
			SessionState:     body.SessionState,
		}, nil
	})
}
