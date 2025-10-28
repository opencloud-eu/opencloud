package jmap

import (
	"context"
	"fmt"
	"slices"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/rs/zerolog"
)

type MailboxesResponse struct {
	Mailboxes []Mailbox `json:"mailboxes"`
	NotFound  []any     `json:"notFound"`
}

// https://jmap.io/spec-mail.html#mailboxget
func (j *Client) GetMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (MailboxesResponse, SessionState, State, Language, Error) {
	logger = j.logger("GetMailbox", session, logger)

	cmd, err := j.request(session, logger,
		invocation(CommandMailboxGet, MailboxGetCommand{AccountId: accountId, Ids: ids}, "0"),
	)
	if err != nil {
		return MailboxesResponse{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (MailboxesResponse, State, Error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, "0", &response)
		if err != nil {
			return MailboxesResponse{}, "", err
		}
		return MailboxesResponse{
			Mailboxes: response.List,
			NotFound:  response.NotFound,
		}, response.State, nil
	})
}

func (j *Client) GetAllMailboxes(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string][]Mailbox, SessionState, State, Language, Error) {
	logger = j.logger("GetAllMailboxes", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return nil, "", "", "", nil
	}

	invocations := make([]Invocation, n)
	for i, accountId := range uniqueAccountIds {
		invocations[i] = invocation(CommandMailboxGet, MailboxGetCommand{AccountId: accountId}, mcid(accountId, "0"))
	}

	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string][]Mailbox, State, Error) {
		resp := map[string][]Mailbox{}
		stateByAccountid := map[string]State{}
		for _, accountId := range uniqueAccountIds {
			var response MailboxGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, mcid(accountId, "0"), &response)
			if err != nil {
				return nil, "", err
			}

			resp[accountId] = response.List
			stateByAccountid[accountId] = response.State
		}
		return resp, squashState(stateByAccountid), nil
	})
}

func (j *Client) SearchMailboxes(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, filter MailboxFilterElement) (map[string][]Mailbox, SessionState, State, Language, Error) {
	logger = j.logger("SearchMailboxes", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	invocations := make([]Invocation, len(uniqueAccountIds)*2)
	for i, accountId := range uniqueAccountIds {
		invocations[i*2+0] = invocation(CommandMailboxQuery, MailboxQueryCommand{AccountId: accountId, Filter: filter}, mcid(accountId, "0"))
		invocations[i*2+1] = invocation(CommandMailboxGet, MailboxGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				Name:     CommandMailboxQuery,
				Path:     "/ids/*",
				ResultOf: mcid(accountId, "0"),
			},
		}, mcid(accountId, "1"))
	}
	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string][]Mailbox, State, Error) {
		resp := map[string][]Mailbox{}
		stateByAccountid := map[string]State{}
		for _, accountId := range uniqueAccountIds {
			var response MailboxGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, mcid(accountId, "1"), &response)
			if err != nil {
				return nil, "", err
			}

			resp[accountId] = response.List
			stateByAccountid[accountId] = response.State
		}
		return resp, squashState(stateByAccountid), nil
	})
}

type MailboxChanges struct {
	Destroyed      []string `json:"destroyed,omitzero"`
	HasMoreChanges bool     `json:"hasMoreChanges,omitzero"`
	NewState       State    `json:"newState"`
	Created        []Email  `json:"created,omitempty"`
	Updated        []Email  `json:"updated,omitempty"`
}

// Retrieve Email changes in a given Mailbox since a given state.
func (j *Client) GetMailboxChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, mailboxId string, sinceState string, fetchBodies bool, maxBodyValueBytes uint, maxChanges uint) (MailboxChanges, SessionState, State, Language, Error) {
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
		IdsRef:             &ResultReference{Name: CommandMailboxChanges, Path: "/created", ResultOf: "0"},
	}
	if maxBodyValueBytes > 0 {
		getCreated.MaxBodyValueBytes = maxBodyValueBytes
	}
	getUpdated := EmailGetRefCommand{
		AccountId:          accountId,
		FetchAllBodyValues: fetchBodies,
		IdsRef:             &ResultReference{Name: CommandMailboxChanges, Path: "/updated", ResultOf: "0"},
	}
	if maxBodyValueBytes > 0 {
		getUpdated.MaxBodyValueBytes = maxBodyValueBytes
	}

	cmd, err := j.request(session, logger,
		invocation(CommandMailboxChanges, changes, "0"),
		invocation(CommandEmailGet, getCreated, "1"),
		invocation(CommandEmailGet, getUpdated, "2"),
	)
	if err != nil {
		return MailboxChanges{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (MailboxChanges, State, Error) {
		var mailboxResponse MailboxChangesResponse
		err = retrieveResponseMatchParameters(logger, body, CommandMailboxChanges, "0", &mailboxResponse)
		if err != nil {
			return MailboxChanges{}, "", err
		}

		var createdResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return MailboxChanges{}, "", err
		}

		var updatedResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "2", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return MailboxChanges{}, "", err
		}

		return MailboxChanges{
			Destroyed:      mailboxResponse.Destroyed,
			HasMoreChanges: mailboxResponse.HasMoreChanges,
			NewState:       mailboxResponse.NewState,
			Created:        createdResponse.List,
			Updated:        createdResponse.List,
		}, createdResponse.State, nil
	})
}

// Retrieve Email changes in Mailboxes of multiple Accounts.
func (j *Client) GetMailboxChangesForMultipleAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, sinceStateMap map[string]string, fetchBodies bool, maxBodyValueBytes uint, maxChanges uint) (map[string]MailboxChanges, SessionState, State, Language, Error) {
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
		return map[string]MailboxChanges{}, "", "", "", nil
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

		getCreated := EmailGetRefCommand{
			AccountId:          accountId,
			FetchAllBodyValues: fetchBodies,
			IdsRef:             &ResultReference{Name: CommandMailboxChanges, Path: "/created", ResultOf: mcid(accountId, "0")},
		}
		if maxBodyValueBytes > 0 {
			getCreated.MaxBodyValueBytes = maxBodyValueBytes
		}
		getUpdated := EmailGetRefCommand{
			AccountId:          accountId,
			FetchAllBodyValues: fetchBodies,
			IdsRef:             &ResultReference{Name: CommandMailboxChanges, Path: "/updated", ResultOf: mcid(accountId, "0")},
		}
		if maxBodyValueBytes > 0 {
			getUpdated.MaxBodyValueBytes = maxBodyValueBytes
		}

		invocations[i*3+0] = invocation(CommandMailboxChanges, changes, mcid(accountId, "0"))
		invocations[i*3+1] = invocation(CommandEmailGet, getCreated, mcid(accountId, "1"))
		invocations[i*3+2] = invocation(CommandEmailGet, getUpdated, mcid(accountId, "2"))
	}

	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]MailboxChanges, State, Error) {
		resp := make(map[string]MailboxChanges, n)
		stateByAccountId := make(map[string]State, n)
		for _, accountId := range uniqueAccountIds {
			var mailboxResponse MailboxChangesResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxChanges, mcid(accountId, "0"), &mailboxResponse)
			if err != nil {
				return nil, "", err
			}

			var createdResponse EmailGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, mcid(accountId, "1"), &createdResponse)
			if err != nil {
				return nil, "", err
			}

			var updatedResponse EmailGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, mcid(accountId, "2"), &updatedResponse)
			if err != nil {
				return nil, "", err
			}

			resp[accountId] = MailboxChanges{
				Destroyed:      mailboxResponse.Destroyed,
				HasMoreChanges: mailboxResponse.HasMoreChanges,
				NewState:       mailboxResponse.NewState,
				Created:        createdResponse.List,
				Updated:        createdResponse.List,
			}
			stateByAccountId[accountId] = createdResponse.State
		}

		return resp, squashState(stateByAccountId), nil
	})
}

func (j *Client) GetMailboxRolesForMultipleAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string][]string, SessionState, State, Language, Error) {
	logger = j.logger("GetMailboxRolesForMultipleAccounts", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return nil, "", "", "", nil
	}

	t := true

	invocations := make([]Invocation, n*2)
	for i, accountId := range uniqueAccountIds {
		invocations[i*2+0] = invocation(CommandMailboxQuery, MailboxQueryCommand{
			AccountId: accountId,
			Filter: MailboxFilterCondition{
				HasAnyRole: &t,
			},
		}, mcid(accountId, "0"))
		invocations[i*2+1] = invocation(CommandMailboxGet, MailboxGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: mcid(accountId, "0"),
				Name:     CommandMailboxQuery,
				Path:     "/ids",
			},
		}, mcid(accountId, "1"))
	}

	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string][]string, State, Error) {
		resp := make(map[string][]string, n)
		stateByAccountId := make(map[string]State, n)
		for _, accountId := range uniqueAccountIds {
			var getResponse MailboxGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, mcid(accountId, "1"), &getResponse)
			if err != nil {
				return nil, "", err
			}
			roles := make([]string, len(getResponse.List))
			for i, mailbox := range getResponse.List {
				roles[i] = mailbox.Role
			}
			slices.Sort(roles)
			resp[accountId] = roles
			stateByAccountId[accountId] = getResponse.State
		}
		return resp, squashState(stateByAccountId), nil
	})
}

func (j *Client) GetInboxNameForMultipleAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]string, SessionState, State, Language, Error) {
	logger = j.logger("GetInboxNameForMultipleAccounts", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return nil, "", "", "", nil
	}

	invocations := make([]Invocation, n*2)
	for i, accountId := range uniqueAccountIds {
		invocations[i*2+0] = invocation(CommandMailboxQuery, MailboxQueryCommand{
			AccountId: accountId,
			Filter: MailboxFilterCondition{
				Role: JmapMailboxRoleInbox,
			},
		}, mcid(accountId, "0"))
	}

	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]string, State, Error) {
		resp := make(map[string]string, n)
		stateByAccountId := make(map[string]State, n)
		for _, accountId := range uniqueAccountIds {
			var r MailboxQueryResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, mcid(accountId, "0"), &r)
			if err != nil {
				return nil, "", err
			}
			switch len(r.Ids) {
			case 0:
				// skip: account has no inbox?
			case 1:
				resp[accountId] = r.Ids[0]
				stateByAccountId[accountId] = r.QueryState
			default:
				logger.Warn().Msgf("multiple ids for mailbox role='%v' for accountId='%v'", JmapMailboxRoleInbox, accountId)
				resp[accountId] = r.Ids[0]
				stateByAccountId[accountId] = r.QueryState
			}
		}
		return resp, squashState(stateByAccountId), nil
	})
}

func (j *Client) UpdateMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, mailboxId string, ifInState string, update MailboxChange) (Mailbox, SessionState, State, Language, Error) {
	logger = j.logger("UpdateMailbox", session, logger)
	cmd, err := j.request(session, logger, invocation(CommandMailboxSet, MailboxSetCommand{
		AccountId: accountId,
		IfInState: ifInState,
		Update: map[string]PatchObject{
			mailboxId: update.AsPatch(),
		},
	}, "0"))
	if err != nil {
		return Mailbox{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (Mailbox, State, Error) {
		var setResp MailboxSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandMailboxSet, "0", &setResp)
		if err != nil {
			return Mailbox{}, "", err
		}
		setErr, notok := setResp.NotUpdated["u"]
		if notok {
			logger.Error().Msgf("%T.NotUpdated returned an error %v", setResp, setErr)
			return Mailbox{}, "", setErrorError(setErr, MailboxType)
		}
		return setResp.Updated["c"], setResp.NewState, nil
	})
}

func (j *Client) CreateMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ifInState string, create MailboxChange) (Mailbox, SessionState, State, Language, Error) {
	logger = j.logger("CreateMailbox", session, logger)
	cmd, err := j.request(session, logger, invocation(CommandMailboxSet, MailboxSetCommand{
		AccountId: accountId,
		IfInState: ifInState,
		Create: map[string]MailboxChange{
			"c": create,
		},
	}, "0"))
	if err != nil {
		return Mailbox{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (Mailbox, State, Error) {
		var setResp MailboxSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandMailboxSet, "0", &setResp)
		if err != nil {
			return Mailbox{}, "", err
		}
		setErr, notok := setResp.NotCreated["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", setResp, setErr)
			return Mailbox{}, "", setErrorError(setErr, MailboxType)
		}
		if mailbox, ok := setResp.Created["c"]; ok {
			return mailbox, setResp.NewState, nil
		} else {
			return Mailbox{}, "", simpleError(fmt.Errorf("failed to find created %T in response", Mailbox{}), JmapErrorMissingCreatedObject)
		}
	})
}

func (j *Client) DeleteMailboxes(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ifInState string, mailboxIds []string) ([]string, SessionState, State, Language, Error) {
	logger = j.logger("DeleteMailbox", session, logger)
	cmd, err := j.request(session, logger, invocation(CommandMailboxSet, MailboxSetCommand{
		AccountId: accountId,
		IfInState: ifInState,
		Destroy:   mailboxIds,
	}, "0"))
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) ([]string, State, Error) {
		var setResp MailboxSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandMailboxSet, "0", &setResp)
		if err != nil {
			return nil, "", err
		}
		setErr, notok := setResp.NotUpdated["u"]
		if notok {
			logger.Error().Msgf("%T.NotUpdated returned an error %v", setResp, setErr)
			return nil, "", setErrorError(setErr, MailboxType)
		}
		return setResp.Destroyed, setResp.NewState, nil
	})
}
