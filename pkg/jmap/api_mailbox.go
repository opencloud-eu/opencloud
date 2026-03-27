package jmap

import (
	"context"
	"fmt"
	"slices"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

var NS_MAILBOX = ns(JmapMail)

type MailboxesResponse struct {
	Mailboxes []Mailbox `json:"mailboxes"`
	NotFound  []any     `json:"notFound"`
}

func (j *Client) GetMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (MailboxesResponse, SessionState, State, Language, Error) {
	return getTemplate(j, "GetMailbox", NS_MAILBOX, CommandCalendarGet,
		func(accountId string, ids []string) MailboxGetCommand {
			return MailboxGetCommand{AccountId: accountId, Ids: ids}
		},
		func(resp MailboxGetResponse) MailboxesResponse {
			return MailboxesResponse{
				Mailboxes: resp.List,
				NotFound:  resp.NotFound,
			}
		},
		func(resp MailboxGetResponse) State { return resp.State },
		accountId, session, ctx, logger, acceptLanguage, ids,
	)
}

func (j *Client) GetAllMailboxes(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string][]Mailbox, SessionState, State, Language, Error) {
	return getTemplateN(j, "GetAllMailboxes", NS_MAILBOX, CommandCalendarGet,
		func(accountId string, ids []string) MailboxGetCommand {
			return MailboxGetCommand{AccountId: accountId}
		},
		func(resp MailboxGetResponse) []Mailbox { return resp.List },
		identity1,
		func(resp MailboxGetResponse) State { return resp.State },
		accountIds, session, ctx, logger, acceptLanguage, []string{},
	)
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
	cmd, err := j.request(session, logger, NS_MAILBOX, invocations...)
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

func (j *Client) SearchMailboxIdsPerRole(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, roles []string) (map[string]map[string]string, SessionState, State, Language, Error) { //NOSONAR
	logger = j.logger("SearchMailboxIdsPerRole", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	invocations := make([]Invocation, len(uniqueAccountIds)*len(roles))
	for i, accountId := range uniqueAccountIds {
		for j, role := range roles {
			invocations[i*len(roles)+j] = invocation(CommandMailboxQuery, MailboxQueryCommand{AccountId: accountId, Filter: MailboxFilterCondition{Role: role}}, mcid(accountId, role))
		}
	}
	cmd, err := j.request(session, logger, NS_MAILBOX, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]map[string]string, State, Error) {
		resp := map[string]map[string]string{}
		stateByAccountid := map[string]State{}
		for _, accountId := range uniqueAccountIds {
			mailboxIdsByRole := map[string]string{}
			for _, role := range roles {
				var response MailboxQueryResponse
				err = retrieveResponseMatchParameters(logger, body, CommandMailboxQuery, mcid(accountId, role), &response)
				if err != nil {
					return nil, "", err
				}
				if len(response.Ids) == 1 {
					mailboxIdsByRole[role] = response.Ids[0]
				}
				if _, ok := stateByAccountid[accountId]; !ok {
					stateByAccountid[accountId] = response.QueryState
				}
			}
			resp[accountId] = mailboxIdsByRole
		}
		return resp, squashState(stateByAccountid), nil
	})
}

type MailboxChanges struct {
	HasMoreChanges bool      `json:"hasMoreChanges"`
	OldState       State     `json:"oldState,omitempty"`
	NewState       State     `json:"newState"`
	Created        []Mailbox `json:"created,omitempty"`
	Updated        []Mailbox `json:"updated,omitempty"`
	Destroyed      []string  `json:"destroyed,omitempty"`
}

func newMailboxChanges(oldState, newState State, hasMoreChanges bool, created, updated []Mailbox, destroyed []string) MailboxChanges {
	return MailboxChanges{
		OldState:       oldState,
		NewState:       newState,
		HasMoreChanges: hasMoreChanges,
		Created:        created,
		Updated:        updated,
		Destroyed:      destroyed,
	}
}

// Retrieve Mailbox changes since a given state.
// @apidoc mailboxes,changes
func (j *Client) GetMailboxChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, sinceState State, maxChanges uint) (MailboxChanges, SessionState, State, Language, Error) {
	return changesTemplate(j, "GetMailboxChanges", NS_MAILBOX,
		CommandMailboxChanges, CommandMailboxGet,
		func() MailboxChangesCommand {
			return MailboxChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
		func(path string, rof string) MailboxGetRefCommand {
			return MailboxGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandMailboxChanges,
					Path:     path,
					ResultOf: rof,
				},
			}
		},
		func(resp MailboxChangesResponse) (State, State, bool, []string) {
			return resp.OldState, resp.NewState, resp.HasMoreChanges, resp.Destroyed
		},
		func(resp MailboxGetResponse) []Mailbox { return resp.List },
		newMailboxChanges,
		func(resp MailboxGetResponse) State { return resp.State },
		session, ctx, logger, acceptLanguage,
	)
}

// Retrieve Mailbox changes of multiple Accounts.
func (j *Client) GetMailboxChangesForMultipleAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, sinceStateMap map[string]State, maxChanges uint) (map[string]MailboxChanges, SessionState, State, Language, Error) { //NOSONAR
	return changesTemplateN(j, "GetMailboxChangesForMultipleAccounts", NS_MAILBOX,
		accountIds, sinceStateMap, CommandMailboxChanges, CommandMailboxGet,
		func(accountId string, state State) MailboxChangesCommand {
			return MailboxChangesCommand{AccountId: accountId, SinceState: state, MaxChanges: posUIntPtr(maxChanges)}
		},
		func(accountId string, path string, ref string) MailboxGetRefCommand {
			return MailboxGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: CommandMailboxChanges, Path: path, ResultOf: ref}}
		},
		func(resp MailboxChangesResponse) (State, State, bool, []string) {
			return resp.OldState, resp.NewState, resp.HasMoreChanges, resp.Destroyed
		},
		func(resp MailboxGetResponse) []Mailbox { return resp.List },
		newMailboxChanges,
		identity1,
		func(resp MailboxGetResponse) State { return resp.State },
		session, ctx, logger, acceptLanguage,
	)
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

	cmd, err := j.request(session, logger, NS_MAILBOX, invocations...)
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

	cmd, err := j.request(session, logger, NS_MAILBOX, invocations...)
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

func (j *Client) UpdateMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, mailboxId string, ifInState string, update MailboxChange) (Mailbox, SessionState, State, Language, Error) { //NOSONAR
	logger = j.logger("UpdateMailbox", session, logger)
	cmd, err := j.request(session, logger, NS_MAILBOX, invocation(CommandMailboxSet, MailboxSetCommand{
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
	cmd, err := j.request(session, logger, NS_MAILBOX, invocation(CommandMailboxSet, MailboxSetCommand{
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
			return Mailbox{}, "", jmapError(fmt.Errorf("failed to find created %T in response", Mailbox{}), JmapErrorMissingCreatedObject)
		}
	})
}

func (j *Client) DeleteMailboxes(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ifInState string, mailboxIds []string) ([]string, SessionState, State, Language, Error) {
	logger = j.logger("DeleteMailbox", session, logger)
	cmd, err := j.request(session, logger, NS_MAILBOX, invocation(CommandMailboxSet, MailboxSetCommand{
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
