package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// https://jmap.io/spec-mail.html#mailboxget
func (j *Client) GetMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, ids []string) (MailboxGetResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetMailbox", session, logger)
	cmd, err := request(invocation(MailboxGet, MailboxGetCommand{AccountId: aid, Ids: ids}, "0"))
	if err != nil {
		return MailboxGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (MailboxGetResponse, Error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(body, MailboxGet, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

func (j *Client) GetAllMailboxes(accountId string, session *Session, ctx context.Context, logger *log.Logger) (MailboxGetResponse, Error) {
	return j.GetMailbox(accountId, session, ctx, logger, nil)
}

// https://jmap.io/spec-mail.html#mailboxquery
func (j *Client) QueryMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, filter MailboxFilterElement) (MailboxQueryResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "QueryMailbox", session, logger)
	cmd, err := request(invocation(MailboxQuery, MailboxQueryCommand{AccountId: aid, Filter: filter}, "0"))
	if err != nil {
		return MailboxQueryResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (MailboxQueryResponse, Error) {
		var response MailboxQueryResponse
		err = retrieveResponseMatchParameters(body, MailboxQuery, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

type Mailboxes struct {
	Mailboxes []Mailbox `json:"mailboxes,omitempty"`
	State     string    `json:"state,omitempty"`
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
		return Mailboxes{Mailboxes: response.List, State: body.SessionState}, nil
	})
}
