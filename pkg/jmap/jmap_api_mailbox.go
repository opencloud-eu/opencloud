package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
)

type MailboxesResponse struct {
	Mailboxes []Mailbox `json:"mailboxes"`
	NotFound  []any     `json:"notFound"`
	State     State     `json:"state"`
}

// https://jmap.io/spec-mail.html#mailboxget
func (j *Client) GetMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, ids []string) (MailboxesResponse, SessionState, Error) {
	logger = j.logger(accountId, "GetMailbox", session, logger)
	cmd, err := request(invocation(CommandMailboxGet, MailboxGetCommand{AccountId: accountId, Ids: ids}, "0"))
	if err != nil {
		logger.Error().Err(err)
		return MailboxesResponse{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (MailboxesResponse, Error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(body, CommandMailboxGet, "0", &response)
		if err != nil {
			logger.Error().Err(err)
			return MailboxesResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		return MailboxesResponse{
			Mailboxes: response.List,
			NotFound:  response.NotFound,
			State:     response.State,
		}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

type AllMailboxesResponse struct {
	Mailboxes []Mailbox `json:"mailboxes"`
	State     State     `json:"state"`
}

func (j *Client) GetAllMailboxes(accountId string, session *Session, ctx context.Context, logger *log.Logger) (AllMailboxesResponse, SessionState, Error) {
	resp, sessionState, err := j.GetMailbox(accountId, session, ctx, logger, nil)
	if err != nil {
		return AllMailboxesResponse{}, sessionState, err
	}
	return AllMailboxesResponse{
		Mailboxes: resp.Mailboxes,
		State:     resp.State,
	}, sessionState, nil
}

type Mailboxes struct {
	// The list of mailboxes that were found using the specified search criteria.
	Mailboxes []Mailbox `json:"mailboxes,omitempty"`
	// The state of the search.
	State State `json:"state,omitempty"`
}

func (j *Client) SearchMailboxes(accountId string, session *Session, ctx context.Context, logger *log.Logger, filter MailboxFilterElement) (Mailboxes, SessionState, Error) {
	logger = j.logger(accountId, "SearchMailboxes", session, logger)

	cmd, err := request(
		invocation(CommandMailboxQuery, MailboxQueryCommand{AccountId: accountId, Filter: filter}, "0"),
		invocation(CommandMailboxGet, MailboxGetRefCommand{
			AccountId: accountId,
			IdRef:     &ResultReference{Name: CommandMailboxQuery, Path: "/ids/*", ResultOf: "0"},
		}, "1"),
	)
	if err != nil {
		logger.Error().Err(err)
		return Mailboxes{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (Mailboxes, Error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(body, CommandMailboxGet, "1", &response)
		if err != nil {
			logger.Error().Err(err)
			return Mailboxes{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		return Mailboxes{Mailboxes: response.List, State: response.State}, nil
	})
}

type MailboxChanges struct {
	Destroyed      []string `json:"destroyed,omitzero"`
	HasMoreChanges bool     `json:"hasMoreChanges,omitzero"`
	NewState       State    `json:"newState"`
	Created        []Email  `json:"created,omitempty"`
	Updated        []Email  `json:"updated,omitempty"`
	State          State    `json:"state,omitempty"`
}

// Retrieve Email changes in a given Mailbox since a given state.
func (j *Client) GetMailboxChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger, mailboxId string, sinceState string, fetchBodies bool, maxBodyValueBytes uint, maxChanges uint) (MailboxChanges, SessionState, Error) {
	logger = j.loggerParams(accountId, "GetMailboxChanges", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Str(logSinceState, sinceState)
	})

	changes := MailboxChangesCommand{
		AccountId:  accountId,
		SinceState: sinceState,
	}
	if maxChanges > 0 {
		changes.MaxChanges = maxChanges
	}

	getCreated := EmailGetRefCommand{
		AccountId:          accountId,
		FetchAllBodyValues: fetchBodies,
		IdRef:              &ResultReference{Name: CommandMailboxChanges, Path: "/created", ResultOf: "0"},
	}
	if maxBodyValueBytes > 0 {
		getCreated.MaxBodyValueBytes = maxBodyValueBytes
	}
	getUpdated := EmailGetRefCommand{
		AccountId:          accountId,
		FetchAllBodyValues: fetchBodies,
		IdRef:              &ResultReference{Name: CommandMailboxChanges, Path: "/updated", ResultOf: "0"},
	}
	if maxBodyValueBytes > 0 {
		getUpdated.MaxBodyValueBytes = maxBodyValueBytes
	}

	cmd, err := request(
		invocation(CommandMailboxChanges, changes, "0"),
		invocation(CommandEmailGet, getCreated, "1"),
		invocation(CommandEmailGet, getUpdated, "2"),
	)
	if err != nil {
		logger.Error().Err(err)
		return MailboxChanges{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (MailboxChanges, Error) {
		var mailboxResponse MailboxChangesResponse
		err = retrieveResponseMatchParameters(body, CommandMailboxChanges, "0", &mailboxResponse)
		if err != nil {
			logger.Error().Err(err)
			return MailboxChanges{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		var createdResponse EmailGetResponse
		err = retrieveResponseMatchParameters(body, CommandEmailGet, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err)
			return MailboxChanges{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		var updatedResponse EmailGetResponse
		err = retrieveResponseMatchParameters(body, CommandEmailGet, "2", &updatedResponse)
		if err != nil {
			logger.Error().Err(err)
			return MailboxChanges{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		return MailboxChanges{
			Destroyed:      mailboxResponse.Destroyed,
			HasMoreChanges: mailboxResponse.HasMoreChanges,
			NewState:       mailboxResponse.NewState,
			Created:        createdResponse.List,
			Updated:        createdResponse.List,
			State:          createdResponse.State,
		}, nil
	})
}
