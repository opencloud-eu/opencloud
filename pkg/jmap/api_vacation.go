package jmap

import (
	"fmt"
	"time"
)

var NS_VACATION = ns(JmapVacationResponse)

const (
	vacationResponseId = "singleton"
)

func (j *Client) GetVacationResponse(accountId string, ctx Context) (VacationResponseGetResponse, SessionState, State, Language, Error) {
	return get(j, "GetVacationResponse", VacationResponseType,
		func(accountId string, ids []string) VacationResponseGetCommand {
			return VacationResponseGetCommand{AccountId: accountId}
		},
		VacationResponseGetResponse{},
		identity1,
		accountId, []string{},
		ctx,
	)
}

// Same as VacationResponse but without the id.
type VacationResponseChange struct {
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

var _ Change = VacationResponseChange{}

func (m VacationResponseChange) AsPatch() (PatchObject, error) {
	return toPatchObject(m)
}

type VacationResponseChanges ChangesTemplate[VacationResponse]

var _ Changes[VacationResponse] = VacationResponseChanges{}

func (c VacationResponseChanges) GetHasMoreChanges() bool        { return c.HasMoreChanges }
func (c VacationResponseChanges) GetOldState() State             { return c.OldState }
func (c VacationResponseChanges) GetNewState() State             { return c.NewState }
func (c VacationResponseChanges) GetCreated() []VacationResponse { return c.Created }
func (c VacationResponseChanges) GetUpdated() []VacationResponse { return c.Updated }
func (c VacationResponseChanges) GetDestroyed() []string         { return c.Destroyed }

func (j *Client) SetVacationResponse(accountId string, vacation VacationResponseChange,
	ctx Context) (VacationResponse, SessionState, State, Language, Error) {
	logger := j.logger("SetVacationResponse", ctx)
	ctx = ctx.WithLogger(logger)

	set := VacationResponseSetCommand{
		AccountId: accountId,
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

	get := VacationResponseGetCommand{AccountId: accountId}

	cmd, err := j.request(ctx, NS_VACATION,
		invocation(set, "0"),
		// chain a second request to get the current complete VacationResponse object
		// after performing the changes, as that makes for a better API
		invocation(get, "1"),
	)
	if err != nil {
		return VacationResponse{}, "", "", "", err
	}
	return command(j, ctx, cmd, func(body *Response) (VacationResponse, State, Error) {
		var setResponse VacationResponseSetResponse
		err = retrieveSet(ctx, body, set, "0", &setResponse)
		if err != nil {
			return VacationResponse{}, "", err
		}

		setErr, notok := setResponse.NotCreated[vacationResponseId]
		if notok {
			// this means that the VacationResponse was not updated
			logger.Error().Msgf("%T.NotCreated contains an error: %v", setResponse, setErr)
			return VacationResponse{}, "", setErrorError(setErr, VacationResponseType)
		}

		var getResponse VacationResponseGetResponse
		err = retrieveGet(ctx, body, get, "1", &getResponse)
		if err != nil {
			return VacationResponse{}, "", err
		}

		if len(getResponse.List) != 1 {
			berr := fmt.Errorf("failed to find %s in %s response", VacationResponseType, string(CommandVacationResponseGet))
			logger.Error().Msg(berr.Error())
			return VacationResponse{}, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		return getResponse.List[0], setResponse.NewState, nil
	})
}
