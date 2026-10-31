package jmap

import (
	"time"
)

var NS_VACATION = ns(JmapVacationResponse)

const (
	vacationResponseId = "singleton"
)

func (j *Client) GetVacationResponse(accountId AccountId, ctx Context) (Result[VacationResponseGetResponse], error) {
	return get(j, "GetVacationResponse", VacationResponseType,
		func(accountId AccountId, ids []string) VacationResponseGetCommand {
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
	IsEnabled *bool `json:"isEnabled,omitempty"`
	// If "isEnabled" is true, messages that arrive on or after this date-time (but before the "toDate" if defined) should receive the
	// user's vacation response. If null, the vacation response is effective immediately.
	FromDate *time.Time `json:"fromDate,omitzero"`
	// If "isEnabled" is true, messages that arrive before this date-time but on or after the "fromDate" if defined) should receive the
	// user's vacation response.  If null, the vacation response is effective indefinitely.
	ToDate *time.Time `json:"toDate,omitzero"`
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

var _ Change[VacationResponse] = VacationResponseChange{}

func (m VacationResponseChange) AsPatch() (PatchObject, error) {
	return toPatchObject(m)
}

func (m VacationResponseChange) GetMarker() VacationResponse {
	return VacationResponse{}
}

type VacationResponseChanges ChangesTemplate[VacationResponse]

var _ Changes[VacationResponse] = VacationResponseChanges{}

func (c VacationResponseChanges) GetHasMoreChanges() bool        { return c.HasMoreChanges }
func (c VacationResponseChanges) GetOldState() State             { return c.OldState }
func (c VacationResponseChanges) GetNewState() State             { return c.NewState }
func (c VacationResponseChanges) GetCreated() []VacationResponse { return c.Created }
func (c VacationResponseChanges) GetUpdated() []VacationResponse { return c.Updated }
func (c VacationResponseChanges) GetDestroyed() []string         { return c.Destroyed }

func (j *Client) SetVacationResponse(accountId AccountId, change VacationResponseChange,
	ctx Context) (Result[VacationResponse], error) {
	return update(j, "SetVacationResponse", VacationResponseType,
		func(update map[string]PatchObject) VacationResponseSetCommand {
			return VacationResponseSetCommand{AccountId: accountId, Update: update}
		},
		func(_ string) VacationResponseGetCommand {
			return VacationResponseGetCommand{AccountId: accountId}
		},
		func(resp VacationResponseSetResponse) map[string]SetError { return resp.NotUpdated },
		func(resp VacationResponseGetResponse) VacationResponse { return resp.List[0] },
		vacationResponseId, change,
		ctx,
	)
}
