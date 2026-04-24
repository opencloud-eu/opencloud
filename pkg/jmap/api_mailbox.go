package jmap

import (
	"slices"

	"github.com/opencloud-eu/opencloud/pkg/structs"
)

var NS_MAILBOX = ns(JmapMail)

func (j *Client) GetMailbox(accountId string, ids []string, ctx Context) (MailboxGetResponse, SessionState, State, Language, Error) {
	return get(j, "GetMailbox", MailboxType,
		func(accountId string, ids []string) MailboxGetCommand {
			return MailboxGetCommand{AccountId: accountId, Ids: ids}
		},
		MailboxGetResponse{},
		identity1,
		accountId, ids,
		ctx,
	)
}

func (j *Client) GetAllMailboxes(accountIds []string, ctx Context) (map[string][]Mailbox, SessionState, State, Language, Error) {
	return getAN(j, "GetAllMailboxes", MailboxType,
		func(accountId string, ids []string) MailboxGetCommand {
			return MailboxGetCommand{AccountId: accountId}
		},
		MailboxGetResponse{},
		identity1,
		accountIds, []string{},
		ctx,
	)
}

func (j *Client) SearchMailboxes(accountIds []string, filter MailboxFilterElement, ctx Context) (map[string][]Mailbox, SessionState, State, Language, Error) {
	logger := j.logger("SearchMailboxes", ctx)
	ctx = ctx.WithLogger(logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	invocations := make([]Invocation, len(uniqueAccountIds)*2)
	for i, accountId := range uniqueAccountIds {
		invocations[i*2+0] = invocation(MailboxQueryCommand{AccountId: accountId, Filter: filter}, mcid(accountId, "0"))
		invocations[i*2+1] = invocation(MailboxGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				Name:     CommandMailboxQuery,
				Path:     "/ids/*",
				ResultOf: mcid(accountId, "0"),
			},
		}, mcid(accountId, "1"))
	}
	cmd, err := j.request(ctx, NS_MAILBOX, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (map[string][]Mailbox, State, Error) {
		resp := map[string][]Mailbox{}
		stateByAccountid := map[string]State{}
		for _, accountId := range uniqueAccountIds {
			var response MailboxGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandMailboxGet, mcid(accountId, "1"), &response)
			if err != nil {
				return nil, "", err
			}

			resp[accountId] = response.List
			stateByAccountid[accountId] = response.State
		}
		return resp, squashState(stateByAccountid), nil
	})
}

func (j *Client) SearchMailboxIdsPerRole(accountIds []string, roles []string, ctx Context) (map[string]map[string]string, SessionState, State, Language, Error) { //NOSONAR
	logger := j.logger("SearchMailboxIdsPerRole", ctx)
	ctx = ctx.WithLogger(logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	invocations := make([]Invocation, len(uniqueAccountIds)*len(roles))
	for i, accountId := range uniqueAccountIds {
		for j, role := range roles {
			invocations[i*len(roles)+j] = invocation(MailboxQueryCommand{AccountId: accountId, Filter: MailboxFilterCondition{Role: role}}, mcid(accountId, role))
		}
	}
	cmd, err := j.request(ctx, NS_MAILBOX, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (map[string]map[string]string, State, Error) {
		resp := map[string]map[string]string{}
		stateByAccountid := map[string]State{}
		for _, accountId := range uniqueAccountIds {
			mailboxIdsByRole := map[string]string{}
			for _, role := range roles {
				var response MailboxQueryResponse
				err = retrieveResponseMatchParameters(ctx, body, CommandMailboxQuery, mcid(accountId, role), &response)
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

type MailboxChanges ChangesTemplate[Mailbox]

var _ Changes[Mailbox] = MailboxChanges{}

func (c MailboxChanges) GetHasMoreChanges() bool { return c.HasMoreChanges }
func (c MailboxChanges) GetOldState() State      { return c.OldState }
func (c MailboxChanges) GetNewState() State      { return c.NewState }
func (c MailboxChanges) GetCreated() []Mailbox   { return c.Created }
func (c MailboxChanges) GetUpdated() []Mailbox   { return c.Updated }
func (c MailboxChanges) GetDestroyed() []string  { return c.Destroyed }

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
func (j *Client) GetMailboxChanges(accountId string, sinceState State, maxChanges uint,
	ctx Context) (MailboxChanges, SessionState, State, Language, Error) {
	return changesA(j, "GetMailboxChanges", MailboxType,
		func() MailboxChangesCommand {
			return MailboxChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: uintPtr(maxChanges)}
		},
		MailboxChangesResponse{},
		MailboxGetResponse{},
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
		newMailboxChanges,
		ctx,
	)
}

// Retrieve Mailbox changes of multiple Accounts.
// @api:tags email,changes
func (j *Client) GetMailboxChangesForMultipleAccounts(accountIds []string, //NOSONAR
	sinceStateMap map[string]State, maxChanges uint,
	ctx Context) (map[string]MailboxChanges, SessionState, State, Language, Error) {
	return changesN(j, "GetMailboxChangesForMultipleAccounts", MailboxType,
		accountIds, sinceStateMap,
		func(accountId string, state State) MailboxChangesCommand {
			return MailboxChangesCommand{AccountId: accountId, SinceState: state, MaxChanges: uintPtr(maxChanges)}
		},
		MailboxChangesResponse{},
		func(accountId string, path string, ref string) MailboxGetRefCommand {
			return MailboxGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: CommandMailboxChanges, Path: path, ResultOf: ref}}
		},
		func(resp MailboxGetResponse) []Mailbox { return resp.List },
		newMailboxChanges,
		identity1,
		func(resp MailboxGetResponse) State { return resp.State },
		ctx,
	)
}

func (j *Client) GetMailboxRolesForMultipleAccounts(accountIds []string, ctx Context) (map[string]*[]string, SessionState, State, Language, Error) {
	return queryN(j, "GetMailboxRolesForMultipleAccounts", MailboxType,
		[]MailboxComparator{{Property: MailboxPropertySortOrder, IsAscending: true}},
		func(accountId string, filter MailboxFilterCondition, sortBy []MailboxComparator, _ int, _ *uint) MailboxQueryCommand {
			return MailboxQueryCommand{AccountId: accountId, Filter: filter, Sort: sortBy, SortAsTree: false, FilterAsTree: false, Position: 0, Limit: nil, CalculateTotal: false}
		},
		func(accountId string, cmd Command, path, rof string) MailboxGetRefCommand {
			return MailboxGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: cmd, Path: path, ResultOf: rof}}
		},
		func(_ MailboxQueryResponse, get MailboxGetResponse) *[]string {
			roles := structs.Map(get.List, func(m Mailbox) string { return m.Role })
			slices.Sort(roles)
			return &roles
		},
		accountIds, MailboxFilterCondition{HasAnyRole: truep}, nil, nil, 0,
		ctx,
	)
}

func (j *Client) GetInboxNameForMultipleAccounts(accountIds []string, ctx Context) (map[string]string, SessionState, State, Language, Error) {
	logger := j.logger("GetInboxNameForMultipleAccounts", ctx)
	ctx = ctx.WithLogger(logger)

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return nil, "", "", "", nil
	}

	invocations := make([]Invocation, n*2)
	for i, accountId := range uniqueAccountIds {
		invocations[i*2+0] = invocation(MailboxQueryCommand{
			AccountId: accountId,
			Filter: MailboxFilterCondition{
				Role: JmapMailboxRoleInbox,
			},
		}, mcid(accountId, "0"))
	}

	cmd, err := j.request(ctx, NS_MAILBOX, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (map[string]string, State, Error) {
		resp := make(map[string]string, n)
		stateByAccountId := make(map[string]State, n)
		for _, accountId := range uniqueAccountIds {
			var r MailboxQueryResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandMailboxGet, mcid(accountId, "0"), &r)
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

func (j *Client) UpdateMailbox(accountId string, mailboxId string, change MailboxChange, //NOSONAR
	ctx Context) (Mailbox, SessionState, State, Language, Error) {
	return update(j, "UpdateMailbox", MailboxType,
		func(update map[string]PatchObject) MailboxSetCommand {
			return MailboxSetCommand{AccountId: accountId, Update: update}
		},
		func(id string) MailboxGetCommand {
			return MailboxGetCommand{AccountId: accountId, Ids: []string{id}}
		},
		func(resp MailboxSetResponse) map[string]SetError { return resp.NotUpdated },
		func(resp MailboxGetResponse) Mailbox { return resp.List[0] },
		mailboxId, change,
		ctx,
	)
}

func (j *Client) CreateMailbox(accountId string, mailbox MailboxChange, ctx Context) (*Mailbox, SessionState, State, Language, Error) {
	return create(j, "CreateMailbox", MailboxType,
		func(accountId string, create map[string]MailboxChange) MailboxSetCommand {
			return MailboxSetCommand{AccountId: accountId, Create: create}
		},
		func(accountId string, ids string) MailboxGetCommand {
			return MailboxGetCommand{AccountId: accountId, Ids: []string{ids}}
		},
		func(resp MailboxSetResponse) map[string]*Mailbox {
			return resp.Created
		},
		func(resp MailboxGetResponse) []Mailbox {
			return resp.List
		},
		accountId, mailbox,
		ctx,
	)
}

func (j *Client) DeleteMailboxes(accountId string, destroyIds []string, ctx Context) (map[string]SetError, SessionState, State, Language, Error) {
	return destroy(j, "DeleteMailboxes", MailboxType,
		func(accountId string, destroy []string) MailboxSetCommand {
			return MailboxSetCommand{AccountId: accountId, Destroy: destroyIds}
		},
		MailboxSetResponse{},
		accountId, destroyIds,
		ctx,
	)
}
