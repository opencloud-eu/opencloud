package jmap

import (
	"context"
	"strconv"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/rs/zerolog"
)

type Identities struct {
	Identities   []Identity `json:"identities"`
	State        string     `json:"state"`
	SessionState string     `json:"sessionState"`
}

// https://jmap.io/spec-mail.html#identityget
func (j *Client) GetIdentity(accountId string, session *Session, ctx context.Context, logger *log.Logger) (Identities, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetIdentity", session, logger)
	cmd, err := request(invocation(CommandIdentityGet, IdentityGetCommand{AccountId: aid}, "0"))
	if err != nil {
		logger.Error().Err(err)
		return Identities{}, simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (Identities, Error) {
		var response IdentityGetResponse
		err = retrieveResponseMatchParameters(body, CommandIdentityGet, "0", &response)
		if err != nil {
			logger.Error().Err(err)
			return Identities{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		return Identities{
			Identities:   response.List,
			State:        response.State,
			SessionState: body.SessionState,
		}, nil
	})
}

type IdentitiesGetResponse struct {
	Identities   map[string][]Identity `json:"identities,omitempty"`
	NotFound     []string              `json:"notFound,omitempty"`
	State        string                `json:"state"`
	SessionState string                `json:"sessionState"`
}

func (j *Client) GetIdentities(accountIds []string, session *Session, ctx context.Context, logger *log.Logger) (IdentitiesGetResponse, Error) {
	uniqueAccountIds := structs.Uniq(accountIds)

	logger = j.loggerParams("", "GetIdentities", session, logger, func(l zerolog.Context) zerolog.Context {
		return l.Array(logAccountId, log.SafeStringArray(uniqueAccountIds))
	})

	calls := make([]Invocation, len(uniqueAccountIds))
	for i, accountId := range uniqueAccountIds {
		calls[i] = invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId}, strconv.Itoa(i))
	}

	cmd, err := request(calls...)
	if err != nil {
		logger.Error().Err(err)
		return IdentitiesGetResponse{}, simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (IdentitiesGetResponse, Error) {
		identities := make(map[string][]Identity, len(uniqueAccountIds))
		lastState := ""
		notFound := []string{}
		for i, accountId := range uniqueAccountIds {
			var response IdentityGetResponse
			err = retrieveResponseMatchParameters(body, CommandIdentityGet, strconv.Itoa(i), &response)
			if err != nil {
				logger.Error().Err(err)
				return IdentitiesGetResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			} else {
				identities[accountId] = response.List
			}
			lastState = response.State
			notFound = append(notFound, response.NotFound...)
		}

		return IdentitiesGetResponse{
			Identities:   identities,
			NotFound:     structs.Uniq(notFound),
			State:        lastState,
			SessionState: body.SessionState,
		}, nil
	})
}

type IdentitiesAndMailboxesGetResponse struct {
	Identities   map[string][]Identity `json:"identities,omitempty"`
	NotFound     []string              `json:"notFound,omitempty"`
	State        string                `json:"state"`
	SessionState string                `json:"sessionState"`
	Mailboxes    []Mailbox             `json:"mailboxes"`
}

func (j *Client) GetIdentitiesAndMailboxes(mailboxAccountId string, accountIds []string, session *Session, ctx context.Context, logger *log.Logger) (IdentitiesAndMailboxesGetResponse, Error) {
	uniqueAccountIds := structs.Uniq(accountIds)

	logger = j.loggerParams("", "GetIdentitiesAndMailboxes", session, logger, func(l zerolog.Context) zerolog.Context {
		return l.Array(logAccountId, log.SafeStringArray(uniqueAccountIds))
	})

	calls := make([]Invocation, len(uniqueAccountIds)+1)
	calls[0] = invocation(CommandMailboxGet, MailboxGetCommand{AccountId: mailboxAccountId}, "0")
	for i, accountId := range uniqueAccountIds {
		calls[i+1] = invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId}, strconv.Itoa(i+1))
	}

	cmd, err := request(calls...)
	if err != nil {
		logger.Error().Err(err)
		return IdentitiesAndMailboxesGetResponse{}, simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (IdentitiesAndMailboxesGetResponse, Error) {
		identities := make(map[string][]Identity, len(uniqueAccountIds))
		lastState := ""
		notFound := []string{}
		for i, accountId := range uniqueAccountIds {
			var response IdentityGetResponse
			err = retrieveResponseMatchParameters(body, CommandIdentityGet, strconv.Itoa(i+1), &response)
			if err != nil {
				logger.Error().Err(err)
				return IdentitiesAndMailboxesGetResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			} else {
				identities[accountId] = response.List
			}
			lastState = response.State
			notFound = append(notFound, response.NotFound...)
		}

		var mailboxResponse MailboxGetResponse
		err = retrieveResponseMatchParameters(body, CommandMailboxGet, "0", &mailboxResponse)
		if err != nil {
			logger.Error().Err(err)
			return IdentitiesAndMailboxesGetResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		return IdentitiesAndMailboxesGetResponse{
			Identities:   identities,
			NotFound:     structs.Uniq(notFound),
			State:        lastState,
			SessionState: body.SessionState,
			Mailboxes:    mailboxResponse.List,
		}, nil
	})
}
