package jmap

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/rs/zerolog"
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

func getTemplateN[GETREQ any, GETRESP any, ITEM any, RESP any]( //NOSONAR
	client *Client, name string, getCommand Command,
	getCommandFactory func(string, []string) GETREQ,
	itemMapper func(GETRESP) ITEM,
	respMapper func(map[string]ITEM) RESP,
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
		result := map[string]ITEM{}
		responses := map[string]GETRESP{}
		for _, accountId := range uniqueAccountIds {
			var resp GETRESP
			err = retrieveResponseMatchParameters(logger, body, getCommand, mcid(accountId, "0"), &resp)
			if err != nil {
				return zero, "", err
			}
			responses[accountId] = resp
			result[accountId] = itemMapper(resp)
		}
		return respMapper(result), squashStateFunc(responses, stateMapper), nil
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
			return nil, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
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
			return nil, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
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

func changesTemplate[CHANGESREQ any, GETREQ any, CHANGESRESP any, GETRESP any, ITEM any, RESP any]( //NOSONAR
	client *Client, name string,
	changesCommand Command, getCommand Command,
	changesCommandFactory func() CHANGESREQ,
	getCommandFactory func(string, string) GETREQ,
	changesMapper func(CHANGESRESP) (State, State, bool, []string),
	getMapper func(GETRESP) []ITEM,
	respMapper func(State, State, bool, []ITEM, []ITEM, []string) RESP,
	stateMapper func(GETRESP) State,
	session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (RESP, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)
	var zero RESP

	changes := changesCommandFactory()
	getCreated := getCommandFactory("/created", "0") //NOSONAR
	getUpdated := getCommandFactory("/updated", "0") //NOSONAR
	cmd, err := client.request(session, logger,
		invocation(changesCommand, changes, "0"),
		invocation(getCommand, getCreated, "1"),
		invocation(getCommand, getUpdated, "2"),
	)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		var changesResponse CHANGESRESP
		err = retrieveResponseMatchParameters(logger, body, changesCommand, "0", &changesResponse)
		if err != nil {
			return zero, "", err
		}

		var createdResponse GETRESP
		err = retrieveResponseMatchParameters(logger, body, getCommand, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return zero, "", err
		}

		var updatedResponse GETRESP
		err = retrieveResponseMatchParameters(logger, body, getCommand, "2", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return zero, "", err
		}

		oldState, newState, hasMoreChanges, destroyed := changesMapper(changesResponse)
		created := getMapper(createdResponse)
		updated := getMapper(updatedResponse)

		result := respMapper(oldState, newState, hasMoreChanges, created, updated, destroyed)

		return result, stateMapper(createdResponse), nil
	})
}

func changesTemplateN[CHANGESREQ any, GETREQ any, CHANGESRESP any, GETRESP any, ITEM any, CHANGESITEM any, RESP any]( //NOSONAR
	client *Client, name string,
	accountIds []string, sinceStateMap map[string]State,
	changesCommand Command, getCommand Command,
	changesCommandFactory func(string, State) CHANGESREQ,
	getCommandFactory func(string, string, string) GETREQ,
	changesMapper func(CHANGESRESP) (State, State, bool, []string),
	getMapper func(GETRESP) []ITEM,
	changesItemMapper func(State, State, bool, []ITEM, []ITEM, []string) CHANGESITEM,
	respMapper func(map[string]CHANGESITEM) RESP,
	stateMapper func(GETRESP) State,
	session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (RESP, SessionState, State, Language, Error) {
	logger = client.loggerParams(name, session, logger, func(z zerolog.Context) zerolog.Context {
		sinceStateLogDict := zerolog.Dict()
		for k, v := range sinceStateMap {
			sinceStateLogDict.Str(log.SafeString(k), log.SafeString(string(v)))
		}
		return z.Dict(logSinceState, sinceStateLogDict)
	})

	var zero RESP

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return zero, "", "", "", nil
	}

	invocations := make([]Invocation, n*3)
	for i, accountId := range uniqueAccountIds {
		sinceState, ok := sinceStateMap[accountId]
		if !ok {
			sinceState = ""
		}
		changes := changesCommandFactory(accountId, sinceState)
		ref := mcid(accountId, "0")

		getCreated := getCommandFactory(accountId, "/created", ref)
		getUpdated := getCommandFactory(accountId, "/updated", ref)

		invocations[i*3+0] = invocation(changesCommand, changes, ref)
		invocations[i*3+1] = invocation(getCommand, getCreated, mcid(accountId, "1"))
		invocations[i*3+2] = invocation(getCommand, getUpdated, mcid(accountId, "2"))
	}

	cmd, err := client.request(session, logger, invocations...)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		changesItemByAccount := make(map[string]CHANGESITEM, n)
		stateByAccountId := make(map[string]State, n)
		for _, accountId := range uniqueAccountIds {
			var changesResponse CHANGESRESP
			err = retrieveResponseMatchParameters(logger, body, changesCommand, mcid(accountId, "0"), &changesResponse)
			if err != nil {
				return zero, "", err
			}

			var createdResponse GETRESP
			err = retrieveResponseMatchParameters(logger, body, getCommand, mcid(accountId, "1"), &createdResponse)
			if err != nil {
				return zero, "", err
			}

			var updatedResponse GETRESP
			err = retrieveResponseMatchParameters(logger, body, getCommand, mcid(accountId, "2"), &updatedResponse)
			if err != nil {
				return zero, "", err
			}

			oldState, newState, hasMoreChanges, destroyed := changesMapper(changesResponse)
			created := getMapper(createdResponse)
			updated := getMapper(updatedResponse)
			changesItemByAccount[accountId] = changesItemMapper(oldState, newState, hasMoreChanges, created, updated, destroyed)
			stateByAccountId[accountId] = stateMapper(createdResponse)
		}
		return respMapper(changesItemByAccount), squashState(stateByAccountId), nil
	})
}
