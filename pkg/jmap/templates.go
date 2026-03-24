package jmap

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

func getTemplate[GETREQ any, GETRESP any, RESP any]( //NOSONAR
	client *Client, name string, getCommand Command,
	getCommandFactory func(string, []string) GETREQ,
	mapper func(GETRESP) RESP,
	stateMapper func(GETRESP) State,
	accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (RESP, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	var zero RESP

	cmd, err := client.request(session, logger,
		invocation(getCommand, getCommandFactory(accountId, ids), "0"),
	)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		var response GETRESP
		err = retrieveResponseMatchParameters(logger, body, getCommand, "0", &response)
		if err != nil {
			return zero, "", err
		}

		return mapper(response), stateMapper(response), nil
	})
}

func getTemplateN[GETREQ any, GETRESP any, RESP any]( //NOSONAR
	client *Client, name string, getCommand Command,
	getCommandFactory func(string, []string) GETREQ,
	mapper func(map[string]GETRESP) RESP,
	stateMapper func(GETRESP) State,
	accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (RESP, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	var zero RESP

	uniqueAccountIds := structs.Uniq(accountIds)

	invocations := make([]Invocation, len(uniqueAccountIds))
	for i, accountId := range uniqueAccountIds {
		invocations[i] = invocation(getCommand, getCommandFactory(accountId, ids), mcid(accountId, "0"))
	}

	cmd, err := client.request(session, logger, invocations...)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		result := map[string]GETRESP{}
		for _, accountId := range uniqueAccountIds {
			var response GETRESP
			err = retrieveResponseMatchParameters(logger, body, getCommand, mcid(accountId, "0"), &response)
			if err != nil {
				return zero, "", err
			}
			result[accountId] = response
		}
		return mapper(result), squashStateFunc(result, stateMapper), nil
	})
}

func createTemplate[T any, SETREQ any, GETREQ any, SETRESP any, GETRESP any]( //NOSONAR
	client *Client, name string, t ObjectType, setCommand Command, getCommand Command,
	setCommandFactory func(string, map[string]T) SETREQ,
	getCommandFactory func(string, string) GETREQ,
	createdMapper func(SETRESP) map[string]*T,
	notCreatedMapper func(SETRESP) map[string]SetError,
	listMapper func(GETRESP) []T,
	stateMapper func(SETRESP) State,
	accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, create T) (*T, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	createMap := map[string]T{"c": create}
	cmd, err := client.request(session, logger,
		invocation(setCommand, setCommandFactory(accountId, createMap), "0"),
		invocation(getCommand, getCommandFactory(accountId, "#c"), "1"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (*T, State, Error) {
		var setResponse SETRESP
		err = retrieveResponseMatchParameters(logger, body, setCommand, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}

		notCreatedMap := notCreatedMapper(setResponse)
		setErr, notok := notCreatedMap["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", setResponse, setErr)
			return nil, "", setErrorError(setErr, t)
		}

		createdMap := createdMapper(setResponse)
		if created, ok := createdMap["c"]; !ok || created == nil {
			berr := fmt.Errorf("failed to find %s in %s response", string(t), string(setCommand))
			logger.Error().Err(berr)
			return nil, "", simpleError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		var getResponse GETRESP
		err = retrieveResponseMatchParameters(logger, body, getCommand, "1", &getResponse)
		if err != nil {
			return nil, "", err
		}

		list := listMapper(getResponse)

		if len(list) < 1 {
			berr := fmt.Errorf("failed to find %s in %s response", string(t), string(getCommand))
			logger.Error().Err(berr)
			return nil, "", simpleError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		return &list[0], stateMapper(setResponse), nil
	})
}

func deleteTemplate[REQ any, RESP any](client *Client, name string, c Command, //NOSONAR
	commandFactory func(string, []string) REQ,
	notDestroyedMapper func(RESP) map[string]SetError,
	stateMapper func(RESP) State,
	accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]SetError, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	cmd, err := client.request(session, logger,
		invocation(c, commandFactory(accountId, destroy), "0"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]SetError, State, Error) {
		var setResponse RESP
		err = retrieveResponseMatchParameters(logger, body, c, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}
		return notDestroyedMapper(setResponse), stateMapper(setResponse), nil
	})
}
