package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var NS_PRINCIPALS = ns(JmapPrincipals)

type PrincipalsResponse struct {
	Principals []Principal `json:"principals"`
	NotFound   []string    `json:"notFound,omitempty"`
}

func (j *Client) GetPrincipals(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (PrincipalsResponse, SessionState, State, Language, Error) {
	logger = j.logger("GetPrincipals", session, logger)

	cmd, err := j.request(session, logger, NS_PRINCIPALS,
		invocation(CommandPrincipalGet, PrincipalGetCommand{AccountId: accountId, Ids: ids}, "0"),
	)
	if err != nil {
		return PrincipalsResponse{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (PrincipalsResponse, State, Error) {
		var response PrincipalGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandPrincipalGet, "0", &response)
		if err != nil {
			return PrincipalsResponse{}, response.State, err
		}
		return PrincipalsResponse{
			Principals: response.List,
			NotFound:   response.NotFound,
		}, response.State, nil
	})
}
