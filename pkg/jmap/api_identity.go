package jmap

import (
	"strconv"

	"github.com/opencloud-eu/opencloud/pkg/structs"
)

var NS_IDENTITY = ns(JmapMail)

func (j *Client) GetAllIdentities(accountId string, ctx Context) ([]Identity, SessionState, State, Language, Error) {
	return getA(j, "GetAllIdentities", IdentityType,
		func(accountId string, ids []string) IdentityGetCommand {
			return IdentityGetCommand{AccountId: accountId}
		},
		IdentityGetResponse{},
		accountId, []string{},
		ctx,
	)
}

func (j *Client) GetIdentities(accountId string, identityIds []string, ctx Context) ([]Identity, SessionState, State, Language, Error) {
	return getA(j, "GetIdentities", IdentityType,
		func(accountId string, ids []string) IdentityGetCommand {
			return IdentityGetCommand{AccountId: accountId, Ids: ids}
		},
		IdentityGetResponse{},
		accountId, identityIds,
		ctx,
	)
}

func (j *Client) GetIdentitiesForAllAccounts(accountIds []string, ctx Context) (map[string][]Identity, SessionState, State, Language, Error) {
	return getN(j, "GetIdentitiesForAllAccounts", IdentityType,
		func(accountId string, ids []string) IdentityGetCommand {
			return IdentityGetCommand{AccountId: accountId}
		},
		IdentityGetResponse{},
		func(resp IdentityGetResponse) []Identity { return resp.List },
		identity1,
		accountIds, []string{},
		ctx,
	)
}

type IdentitiesAndMailboxesGetResponse struct {
	Identities map[string][]Identity `json:"identities,omitempty"`
	NotFound   []string              `json:"notFound,omitempty"`
	Mailboxes  []Mailbox             `json:"mailboxes"`
}

func (j *Client) GetIdentitiesAndMailboxes(mailboxAccountId string, accountIds []string, ctx Context) (IdentitiesAndMailboxesGetResponse, SessionState, State, Language, Error) {
	uniqueAccountIds := structs.Uniq(accountIds)

	logger := j.logger("GetIdentitiesAndMailboxes", ctx)
	ctx = ctx.WithLogger(logger)

	calls := make([]Invocation, len(uniqueAccountIds)+1)
	calls[0] = invocation(MailboxGetCommand{AccountId: mailboxAccountId}, "0")
	for i, accountId := range uniqueAccountIds {
		calls[i+1] = invocation(IdentityGetCommand{AccountId: accountId}, strconv.Itoa(i+1))
	}

	cmd, err := j.request(ctx, NS_IDENTITY, calls...)
	if err != nil {
		return IdentitiesAndMailboxesGetResponse{}, "", "", "", err
	}
	return command(j, ctx, cmd, func(body *Response) (IdentitiesAndMailboxesGetResponse, State, Error) {
		identities := make(map[string][]Identity, len(uniqueAccountIds))
		stateByAccountId := make(map[string]State, len(uniqueAccountIds))
		notFound := []string{}
		for i, accountId := range uniqueAccountIds {
			var response IdentityGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandIdentityGet, strconv.Itoa(i+1), &response)
			if err != nil {
				return IdentitiesAndMailboxesGetResponse{}, "", err
			} else {
				identities[accountId] = response.List
			}
			stateByAccountId[accountId] = response.State
			notFound = append(notFound, response.NotFound...)
		}

		var mailboxResponse MailboxGetResponse
		err = retrieveResponseMatchParameters(ctx, body, CommandMailboxGet, "0", &mailboxResponse)
		if err != nil {
			return IdentitiesAndMailboxesGetResponse{}, "", err
		}

		return IdentitiesAndMailboxesGetResponse{
			Identities: identities,
			NotFound:   structs.Uniq(notFound),
			Mailboxes:  mailboxResponse.List,
		}, squashState(stateByAccountId), nil
	})
}

func (j *Client) CreateIdentity(accountId string, identity IdentityChange, ctx Context) (*Identity, SessionState, State, Language, Error) {
	return create(j, "CreateIdentity", IdentityType,
		func(accountId string, create map[string]IdentityChange) IdentitySetCommand {
			return IdentitySetCommand{AccountId: accountId, Create: create}
		},
		func(accountId string, ids string) IdentityGetCommand {
			return IdentityGetCommand{AccountId: accountId, Ids: []string{ids}}
		},
		func(resp IdentitySetResponse) map[string]*Identity {
			return resp.Created
		},
		func(resp IdentityGetResponse) []Identity {
			return resp.List
		},
		accountId, identity,
		ctx,
	)
}

func (j *Client) UpdateIdentity(accountId string, id string, changes IdentityChange, ctx Context) (Identity, SessionState, State, Language, Error) {
	return update(j, "UpdateIdentity", IdentityType,
		func(update map[string]PatchObject) IdentitySetCommand {
			return IdentitySetCommand{AccountId: accountId, Update: update}
		},
		func(id string) IdentityGetCommand {
			return IdentityGetCommand{AccountId: accountId, Ids: []string{id}}
		},
		func(resp IdentitySetResponse) map[string]SetError { return resp.NotUpdated },
		func(resp IdentityGetResponse) Identity { return resp.List[0] },
		id, changes,
		ctx,
	)
}

func (j *Client) DeleteIdentity(accountId string, destroyIds []string, ctx Context) (map[string]SetError, SessionState, State, Language, Error) {
	return destroy(j, "DeleteIdentity", IdentityType,
		func(accountId string, destroy []string) IdentitySetCommand {
			return IdentitySetCommand{AccountId: accountId, Destroy: destroyIds}
		},
		IdentitySetResponse{},
		accountId, destroyIds,
		ctx,
	)
}

type IdentityChanges = ChangesTemplate[Identity]

// Retrieve the changes in Email Identities since a given State.
// @api:tags email,changes
func (j *Client) GetIdentityChanges(accountId string, sinceState State, maxChanges uint,
	ctx Context) (IdentityChanges, SessionState, State, Language, Error) {
	return changes(j, "GetIdentityChanges", IdentityType,
		func() IdentityChangesCommand {
			return IdentityChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: uintPtr(maxChanges)}
		},
		IdentityChangesResponse{},
		func(path string, rof string) IdentityGetRefCommand {
			return IdentityGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandIdentityChanges,
					Path:     path,
					ResultOf: rof,
				},
			}
		},
		func(resp IdentityGetResponse) []Identity { return resp.List },
		func(oldState, newState State, hasMoreChanges bool, created, updated []Identity, destroyed []string) IdentityChanges {
			return IdentityChanges{
				OldState:       oldState,
				NewState:       newState,
				HasMoreChanges: hasMoreChanges,
				Created:        created,
				Updated:        updated,
				Destroyed:      destroyed,
			}
		},
		ctx,
	)
}
