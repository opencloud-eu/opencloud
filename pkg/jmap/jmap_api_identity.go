package jmap

import (
	"context"
	"strconv"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
)

// https://jmap.io/spec-mail.html#identityget
func (j *Client) GetIdentity(accountId string, session *Session, ctx context.Context, logger *log.Logger) (IdentityGetResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetIdentity", session, logger)
	cmd, err := request(invocation(IdentityGet, IdentityGetCommand{AccountId: aid}, "0"))
	if err != nil {
		return IdentityGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (IdentityGetResponse, Error) {
		var response IdentityGetResponse
		err = retrieveResponseMatchParameters(body, IdentityGet, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

type IdentitiesGetResponse struct {
	State      string                `json:"state"`
	Identities map[string][]Identity `json:"identities,omitempty"`
	NotFound   []string              `json:"notFound,omitempty"`
}

func (j *Client) GetIdentities(accountIds []string, session *Session, ctx context.Context, logger *log.Logger) (IdentitiesGetResponse, Error) {
	uniqueAccountIds := uniq(accountIds)

	logger = j.loggerParams("", "GetIdentities", session, logger, func(l zerolog.Context) zerolog.Context {
		return l.Array(logAccountId, logstrarray(uniqueAccountIds))
	})

	calls := make([]Invocation, len(uniqueAccountIds))
	for i, accountId := range uniqueAccountIds {
		calls[i] = invocation(IdentityGet, IdentityGetCommand{AccountId: accountId}, strconv.Itoa(i))
	}

	cmd, err := request(calls...)
	if err != nil {
		return IdentitiesGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (IdentitiesGetResponse, Error) {
		identities := make(map[string][]Identity, len(uniqueAccountIds))
		lastState := ""
		notFound := []string{}
		for i, accountId := range uniqueAccountIds {
			var response IdentityGetResponse
			err = retrieveResponseMatchParameters(body, IdentityGet, strconv.Itoa(i), &response)
			if err != nil {
				return IdentitiesGetResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			} else {
				identities[accountId] = response.List
			}
			lastState = response.State
			notFound = append(notFound, response.NotFound...)
		}

		return IdentitiesGetResponse{
			Identities: identities,
			State:      lastState,
			NotFound:   uniq(notFound),
		}, nil
	})
}
