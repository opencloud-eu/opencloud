package jmap

import (
	"context"
	"slices"

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
func (j *Client) GetMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (MailboxesResponse, SessionState, Language, Error) {
	logger = j.logger("GetMailbox", session, logger)

	cmd, err := j.request(session, logger,
		invocation(CommandMailboxGet, MailboxGetCommand{AccountId: accountId, Ids: ids}, "0"),
	)
	if err != nil {
		return MailboxesResponse{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (MailboxesResponse, Error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, "0", &response)
		if err != nil {
			return MailboxesResponse{}, err
		}
		return MailboxesResponse{
			Mailboxes: response.List,
			NotFound:  response.NotFound,
			State:     response.State,
		}, nil
	})
}

type AllMailboxesResponse struct {
	Mailboxes []Mailbox `json:"mailboxes"`
	State     State     `json:"state"`
}

func (j *Client) GetAllMailboxes(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]AllMailboxesResponse, SessionState, Language, Error) {
	logger = j.logger("GetAllMailboxes", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return map[string]AllMailboxesResponse{}, "", "", nil
	}

	invocations := make([]Invocation, n)
	for i, accountId := range uniqueAccountIds {
		invocations[i] = invocation(CommandMailboxGet, MailboxGetCommand{AccountId: accountId}, mcid(accountId, "0"))
	}

	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return map[string]AllMailboxesResponse{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]AllMailboxesResponse, Error) {
		resp := map[string]AllMailboxesResponse{}
		for _, accountId := range uniqueAccountIds {
			var response MailboxGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, mcid(accountId, "0"), &response)
			if err != nil {
				return map[string]AllMailboxesResponse{}, err
			}

			resp[accountId] = AllMailboxesResponse{
				Mailboxes: response.List,
				State:     response.State,
			}
		}
		return resp, nil
	})
}

type Mailboxes struct {
	// The list of mailboxes that were found using the specified search criteria.
	Mailboxes []Mailbox `json:"mailboxes,omitempty"`
	// The state of the search.
	State State `json:"state,omitempty"`
}

func (j *Client) SearchMailboxes(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, filter MailboxFilterElement) (map[string]Mailboxes, SessionState, Language, Error) {
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
		return map[string]Mailboxes{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]Mailboxes, Error) {
		resp := map[string]Mailboxes{}
		for _, accountId := range uniqueAccountIds {
			var response MailboxGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, mcid(accountId, "1"), &response)
			if err != nil {
				return map[string]Mailboxes{}, err
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
func (j *Client) GetMailboxChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, mailboxId string, sinceState string, fetchBodies bool, maxBodyValueBytes uint, maxChanges uint) (MailboxChanges, SessionState, Language, Error) {
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
		return MailboxChanges{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (MailboxChanges, Error) {
		var mailboxResponse MailboxChangesResponse
		err = retrieveResponseMatchParameters(logger, body, CommandMailboxChanges, "0", &mailboxResponse)
		if err != nil {
			return MailboxChanges{}, err
		}

		var createdResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return MailboxChanges{}, err
		}

		var updatedResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "2", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return MailboxChanges{}, err
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
func (j *Client) GetMailboxChangesForMultipleAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, sinceStateMap map[string]string, fetchBodies bool, maxBodyValueBytes uint, maxChanges uint) (map[string]MailboxChanges, SessionState, Language, Error) {
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
		return map[string]MailboxChanges{}, "", "", nil
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
		return map[string]MailboxChanges{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]MailboxChanges, Error) {
		resp := make(map[string]MailboxChanges, n)
		for _, accountId := range uniqueAccountIds {
			var mailboxResponse MailboxChangesResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxChanges, mcid(accountId, "0"), &mailboxResponse)
			if err != nil {
				return map[string]MailboxChanges{}, err
			}

			var createdResponse EmailGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, mcid(accountId, "1"), &createdResponse)
			if err != nil {
				return map[string]MailboxChanges{}, err
			}

			var updatedResponse EmailGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, mcid(accountId, "2"), &updatedResponse)
			if err != nil {
				return map[string]MailboxChanges{}, err
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

func (j *Client) GetMailboxRolesForMultipleAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string][]string, SessionState, Language, Error) {
	logger = j.logger("GetMailboxRolesForMultipleAccounts", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return map[string][]string{}, "", "", nil
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
		return map[string][]string{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string][]string, Error) {
		resp := make(map[string][]string, n)
		for _, accountId := range uniqueAccountIds {
			var getResponse MailboxGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, mcid(accountId, "1"), &getResponse)
			if err != nil {
				return map[string][]string{}, err
			}
			roles := make([]string, len(getResponse.List))
			for i, mailbox := range getResponse.List {
				roles[i] = mailbox.Role
			}
			slices.Sort(roles)
			resp[accountId] = roles
		}
		return resp, nil
	})
}

func (j *Client) GetInboxNameForMultipleAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]string, SessionState, Language, Error) {
	logger = j.logger("GetInboxNameForMultipleAccounts", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return nil, "", "", nil
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
		return nil, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]string, Error) {
		resp := make(map[string]string, n)
		for _, accountId := range uniqueAccountIds {
			var r MailboxQueryResponse
			err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, mcid(accountId, "0"), &r)
			if err != nil {
				return nil, err
			}
			switch len(r.Ids) {
			case 0:
				// skip: account has no inbox?
			case 1:
				resp[accountId] = r.Ids[0]
			default:
				logger.Warn().Msgf("multiple ids for mailbox role='%v' for accountId='%v'", JmapMailboxRoleInbox, accountId)
				resp[accountId] = r.Ids[0]
			}
		}
		return resp, nil
	})
}
