package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type MailboxesResponse struct {
	Mailboxes    []Mailbox `json:"mailboxes"`
	NotFound     []any     `json:"notFound"`
	State        string    `json:"state"`
	SessionState string    `json:"sessionState"`
}

// https://jmap.io/spec-mail.html#mailboxget
func (j *Client) GetMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, ids []string) (MailboxesResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetMailbox", session, logger)
	cmd, err := request(invocation(MailboxGet, MailboxGetCommand{AccountId: aid, Ids: ids}, "0"))
	if err != nil {
		return MailboxesResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (MailboxesResponse, Error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(body, MailboxGet, "0", &response)
		if err != nil {
			return MailboxesResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		return MailboxesResponse{
			Mailboxes:    response.List,
			NotFound:     response.NotFound,
			State:        response.State,
			SessionState: body.SessionState,
		}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

type AllMailboxesResponse struct {
	Mailboxes    []Mailbox `json:"mailboxes"`
	State        string    `json:"state"`
	SessionState string    `json:"sessionState"`
}

func (j *Client) GetAllMailboxes(accountId string, session *Session, ctx context.Context, logger *log.Logger) (AllMailboxesResponse, Error) {
	resp, err := j.GetMailbox(accountId, session, ctx, logger, nil)
	if err != nil {
		return AllMailboxesResponse{}, err
	}
	return AllMailboxesResponse{
		Mailboxes:    resp.Mailboxes,
		State:        resp.State,
		SessionState: resp.SessionState,
	}, nil
}

type Mailboxes struct {
	// The list of mailboxes that were found using the specified search criteria.
	Mailboxes []Mailbox `json:"mailboxes,omitempty"`
	// The state of the search.
	State string `json:"state,omitempty"`
	// The state of the Session.
	SessionState string `json:"sessionState,omitempty"`
}

func (j *Client) SearchMailboxes(accountId string, session *Session, ctx context.Context, logger *log.Logger, filter MailboxFilterElement) (Mailboxes, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "SearchMailboxes", session, logger)

	cmd, err := request(
		invocation(MailboxQuery, MailboxQueryCommand{AccountId: aid, Filter: filter}, "0"),
		invocation(MailboxGet, MailboxGetRefCommand{
			AccountId: aid,
			IdRef:     &ResultReference{Name: MailboxQuery, Path: "/ids/*", ResultOf: "0"},
		}, "1"),
	)
	if err != nil {
		return Mailboxes{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (Mailboxes, Error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(body, MailboxGet, "1", &response)
		if err != nil {
			return Mailboxes{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}
		return Mailboxes{Mailboxes: response.List, State: response.State, SessionState: body.SessionState}, nil
	})
}
