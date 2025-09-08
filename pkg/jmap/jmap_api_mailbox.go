package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/rs/zerolog"
)

type MailboxesResponse struct {
	Mailboxes []Mailbox `json:"mailboxes"`
	NotFound  []any     `json:"notFound"`
	State     State     `json:"state"`
}

// https://jmap.io/spec-mail.html#mailboxget
func (j *Client) GetMailbox(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, ids []string) (map[string]MailboxesResponse, SessionState, Error) {
	logger = j.logger("GetMailbox", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)
	if len(uniqueAccountIds) < 1 {
		return map[string]MailboxesResponse{}, "", nil
	}

	invocations := make([]Invocation, len(uniqueAccountIds))
	for i, accountId := range uniqueAccountIds {
		invocations[i] = invocation(CommandMailboxGet, MailboxGetCommand{AccountId: accountId, Ids: ids}, accountId)
	}

	cmd, err := request(invocations...)
	if err != nil {
		logger.Error().Err(err)
		return map[string]MailboxesResponse{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (map[string]MailboxesResponse, Error) {
		resp := map[string]MailboxesResponse{}
		for _, accountId := range uniqueAccountIds {
			var response MailboxGetResponse
			err = retrieveResponseMatchParameters(body, CommandMailboxGet, "0", &response)
			if err != nil {
				logger.Error().Err(err)
				return map[string]MailboxesResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			}

			resp[accountId] = MailboxesResponse{
				Mailboxes: response.List,
				NotFound:  response.NotFound,
				State:     response.State,
			}
		}
		return resp, nil
	})
}

type AllMailboxesResponse struct {
	Mailboxes []Mailbox `json:"mailboxes"`
	State     State     `json:"state"`
}

func (j *Client) GetAllMailboxes(accountIds []string, session *Session, ctx context.Context, logger *log.Logger) (map[string]AllMailboxesResponse, SessionState, Error) {
	resp, sessionState, err := j.GetMailbox(accountIds, session, ctx, logger, nil)
	if err != nil {
		return map[string]AllMailboxesResponse{}, sessionState, err
	}

	mapped := make(map[string]AllMailboxesResponse, len(resp))
	for accountId, mailboxesResponse := range resp {
		mapped[accountId] = AllMailboxesResponse{
			Mailboxes: mailboxesResponse.Mailboxes,
			State:     mailboxesResponse.State,
		}
	}

	return mapped, sessionState, nil
}

type Mailboxes struct {
	// The list of mailboxes that were found using the specified search criteria.
	Mailboxes []Mailbox `json:"mailboxes,omitempty"`
	// The state of the search.
	State State `json:"state,omitempty"`
}

func (j *Client) SearchMailboxes(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, filter MailboxFilterElement) (map[string]Mailboxes, SessionState, Error) {
	logger = j.logger("SearchMailboxes", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	invocations := make([]Invocation, len(uniqueAccountIds)*2)
	for i, accountId := range uniqueAccountIds {
		baseId := accountId + ":"
		invocations[i*2+0] = invocation(CommandMailboxQuery, MailboxQueryCommand{AccountId: accountId, Filter: filter}, baseId+"0")
		invocations[i*2+1] = invocation(CommandMailboxGet, MailboxGetRefCommand{
			AccountId: accountId,
			IdRef:     &ResultReference{Name: CommandMailboxQuery, Path: "/ids/*", ResultOf: baseId + "0"},
		}, baseId+"1")
	}
	cmd, err := request(invocations...)
	if err != nil {
		logger.Error().Err(err)
		return map[string]Mailboxes{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (map[string]Mailboxes, Error) {
		resp := map[string]Mailboxes{}
		for _, accountId := range uniqueAccountIds {
			baseId := accountId + ":"

			var response MailboxGetResponse
			err = retrieveResponseMatchParameters(body, CommandMailboxGet, baseId+"1", &response)
			if err != nil {
				logger.Error().Err(err)
				return map[string]Mailboxes{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			}

			resp[accountId] = Mailboxes{Mailboxes: response.List, State: response.State}
		}
		return resp, nil
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
	logger = j.loggerParams("GetMailboxChanges", session, logger, func(z zerolog.Context) zerolog.Context {
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

// Retrieve Email changes in Mailboxes of multiple Accounts.
func (j *Client) GetMailboxChangesForMultipleAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, sinceStateMap map[string]string, fetchBodies bool, maxBodyValueBytes uint, maxChanges uint) (map[string]MailboxChanges, SessionState, Error) {
	logger = j.loggerParams("GetMailboxChangesForMultipleAccounts", session, logger, func(z zerolog.Context) zerolog.Context {
		sinceStateLogDict := zerolog.Dict()
		for k, v := range sinceStateMap {
			sinceStateLogDict.Str(log.SafeString(k), log.SafeString(v))
		}
		return z.Bool(logFetchBodies, fetchBodies).Dict(logSinceState, sinceStateLogDict)
	})

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return map[string]MailboxChanges{}, "", nil
	}

	invocations := make([]Invocation, n*3)
	for i, accountId := range uniqueAccountIds {
		changes := MailboxChangesCommand{
			AccountId: accountId,
		}

		sinceState, ok := sinceStateMap[accountId]
		if ok {
			changes.SinceState = sinceState
		}

		if maxChanges > 0 {
			changes.MaxChanges = maxChanges
		}

		baseId := accountId + ":"

		getCreated := EmailGetRefCommand{
			AccountId:          accountId,
			FetchAllBodyValues: fetchBodies,
			IdRef:              &ResultReference{Name: CommandMailboxChanges, Path: "/created", ResultOf: baseId + "0"},
		}
		if maxBodyValueBytes > 0 {
			getCreated.MaxBodyValueBytes = maxBodyValueBytes
		}
		getUpdated := EmailGetRefCommand{
			AccountId:          accountId,
			FetchAllBodyValues: fetchBodies,
			IdRef:              &ResultReference{Name: CommandMailboxChanges, Path: "/updated", ResultOf: baseId + "0"},
		}
		if maxBodyValueBytes > 0 {
			getUpdated.MaxBodyValueBytes = maxBodyValueBytes
		}

		invocations[i*3+0] = invocation(CommandMailboxChanges, changes, baseId+"0")
		invocations[i*3+1] = invocation(CommandEmailGet, getCreated, baseId+"1")
		invocations[i*3+2] = invocation(CommandEmailGet, getUpdated, baseId+"2")
	}

	cmd, err := request(invocations...)
	if err != nil {
		logger.Error().Err(err)
		return map[string]MailboxChanges{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (map[string]MailboxChanges, Error) {
		resp := make(map[string]MailboxChanges, n)
		for _, accountId := range uniqueAccountIds {
			baseId := accountId + ":"

			var mailboxResponse MailboxChangesResponse
			err = retrieveResponseMatchParameters(body, CommandMailboxChanges, baseId+"0", &mailboxResponse)
			if err != nil {
				logger.Error().Err(err)
				return map[string]MailboxChanges{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			}

			var createdResponse EmailGetResponse
			err = retrieveResponseMatchParameters(body, CommandEmailGet, baseId+"1", &createdResponse)
			if err != nil {
				logger.Error().Err(err)
				return map[string]MailboxChanges{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			}

			var updatedResponse EmailGetResponse
			err = retrieveResponseMatchParameters(body, CommandEmailGet, baseId+"2", &updatedResponse)
			if err != nil {
				logger.Error().Err(err)
				return map[string]MailboxChanges{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			}

			resp[accountId] = MailboxChanges{
				Destroyed:      mailboxResponse.Destroyed,
				HasMoreChanges: mailboxResponse.HasMoreChanges,
				NewState:       mailboxResponse.NewState,
				Created:        createdResponse.List,
				Updated:        createdResponse.List,
				State:          createdResponse.State,
			}
		}

		return resp, nil
	})
}
