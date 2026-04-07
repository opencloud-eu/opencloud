package jmap

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/rs/zerolog"
)

func get[GETREQ GetCommand, GETRESP GetResponse, RESP any]( //NOSONAR
	client *Client, name string, using []JmapNamespace,
	getCommandFactory func(string, []string) GETREQ,
	_ GETRESP,
	mapper func(GETRESP) RESP,
	accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (RESP, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	var zero RESP

	get := getCommandFactory(accountId, ids)
	cmd, err := client.request(session, logger, using,
		invocation(get, "0"),
	)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		var response GETRESP
		err = retrieveGet(logger, body, get, "0", &response)
		if err != nil {
			return zero, "", err
		}

		return mapper(response), response.GetState(), nil
	})
}

func getN[GETREQ GetCommand, GETRESP GetResponse, ITEM any, RESP any]( //NOSONAR
	client *Client, name string, using []JmapNamespace,
	getCommandFactory func(string, []string) GETREQ,
	_ GETRESP,
	itemMapper func(GETRESP) ITEM,
	respMapper func(map[string]ITEM) RESP,
	accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (RESP, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	var zero RESP

	uniqueAccountIds := structs.Uniq(accountIds)

	invocations := make([]Invocation, len(uniqueAccountIds))
	var c Command
	for i, accountId := range uniqueAccountIds {
		get := getCommandFactory(accountId, ids)
		c = get.GetCommand()
		invocations[i] = invocation(get, mcid(accountId, "0"))
	}

	cmd, err := client.request(session, logger, using, invocations...)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		result := map[string]ITEM{}
		responses := map[string]GETRESP{}
		for _, accountId := range uniqueAccountIds {
			var resp GETRESP
			err = retrieveResponseMatchParameters(logger, body, c, mcid(accountId, "0"), &resp)
			if err != nil {
				return zero, "", err
			}
			responses[accountId] = resp
			result[accountId] = itemMapper(resp)
		}
		return respMapper(result), squashStateFunc(responses, func(r GETRESP) State { return r.GetState() }), nil
	})
}

func create[T any, C any, SETREQ SetCommand, GETREQ GetCommand, SETRESP SetResponse, GETRESP GetResponse]( //NOSONAR
	client *Client, name string, using []JmapNamespace,
	setCommandFactory func(string, map[string]C) SETREQ,
	getCommandFactory func(string, string) GETREQ,
	createdMapper func(SETRESP) map[string]*T,
	listMapper func(GETRESP) []T,
	accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, create C) (*T, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	createMap := map[string]C{"c": create}
	get := getCommandFactory(accountId, "#c")
	set := setCommandFactory(accountId, createMap)
	cmd, err := client.request(session, logger, using,
		invocation(set, "0"),
		invocation(get, "1"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (*T, State, Error) {
		var setResponse SETRESP
		err = retrieveSet(logger, body, set, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}

		notCreatedMap := setResponse.GetNotCreated()
		setErr, notok := notCreatedMap["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", setResponse, setErr)
			return nil, "", setErrorError(setErr, set.GetObjectType())
		}

		createdMap := createdMapper(setResponse)
		if created, ok := createdMap["c"]; !ok || created == nil {
			berr := fmt.Errorf("failed to find %s in %s response", set.GetObjectType(), set.GetCommand())
			logger.Error().Err(berr)
			return nil, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		var getResponse GETRESP
		err = retrieveGet(logger, body, get, "1", &getResponse)
		if err != nil {
			return nil, "", err
		}

		list := listMapper(getResponse)

		if len(list) < 1 {
			berr := fmt.Errorf("failed to find %s in %s response", get.GetObjectType(), get.GetCommand())
			logger.Error().Err(berr)
			return nil, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		return &list[0], setResponse.GetNewState(), nil
	})
}

func destroy[REQ SetCommand, RESP SetResponse](client *Client, name string, using []JmapNamespace, //NOSONAR
	setCommandFactory func(string, []string) REQ, _ RESP,
	accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]SetError, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)

	set := setCommandFactory(accountId, destroy)
	cmd, err := client.request(session, logger, using,
		invocation(set, "0"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]SetError, State, Error) {
		var setResponse RESP
		err = retrieveSet(logger, body, set, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}
		return setResponse.GetNotDestroyed(), setResponse.GetNewState(), nil
	})
}

func changes[CHANGESREQ ChangesCommand, GETREQ GetCommand, CHANGESRESP ChangesResponse, GETRESP GetResponse, ITEM any, RESP any]( //NOSONAR
	client *Client, name string, using []JmapNamespace,
	changesCommandFactory func() CHANGESREQ,
	_ CHANGESRESP,
	getCommandFactory func(string, string) GETREQ,
	getMapper func(GETRESP) []ITEM,
	respMapper func(State, State, bool, []ITEM, []ITEM, []string) RESP,
	session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (RESP, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)
	var zero RESP

	changes := changesCommandFactory()
	getCreated := getCommandFactory("/created", "0") //NOSONAR
	getUpdated := getCommandFactory("/updated", "0") //NOSONAR

	cmd, err := client.request(session, logger, using,
		invocation(changes, "0"),
		invocation(getCreated, "1"),
		invocation(getUpdated, "2"),
	)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		var changesResponse CHANGESRESP
		err = retrieveChanges(logger, body, changes, "0", &changesResponse)
		if err != nil {
			return zero, "", err
		}

		var createdResponse GETRESP
		err = retrieveGet(logger, body, getCreated, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return zero, "", err
		}

		var updatedResponse GETRESP
		err = retrieveGet(logger, body, getUpdated, "2", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return zero, "", err
		}

		created := getMapper(createdResponse)
		updated := getMapper(updatedResponse)

		result := respMapper(changesResponse.GetOldState(), changesResponse.GetNewState(), changesResponse.GetHasMoreChanges(), created, updated, changesResponse.GetDestroyed())

		return result, changesResponse.GetNewState(), nil
	})
}

func changesN[CHANGESREQ ChangesCommand, GETREQ GetCommand, CHANGESRESP ChangesResponse, GETRESP GetResponse, ITEM any, CHANGESITEM any, RESP any]( //NOSONAR
	client *Client, name string, using []JmapNamespace,
	accountIds []string, sinceStateMap map[string]State,
	changesCommandFactory func(string, State) CHANGESREQ,
	_ CHANGESRESP,
	getCommandFactory func(string, string, string) GETREQ,
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
	getCommand := Command("")
	changesCommand := Command("")
	for i, accountId := range uniqueAccountIds {
		sinceState, ok := sinceStateMap[accountId]
		if !ok {
			sinceState = ""
		}
		changes := changesCommandFactory(accountId, sinceState)
		ref := mcid(accountId, "0")

		getCreated := getCommandFactory(accountId, "/created", ref)
		getUpdated := getCommandFactory(accountId, "/updated", ref)

		invocations[i*3+0] = invocation(changes, ref)
		invocations[i*3+1] = invocation(getCreated, mcid(accountId, "1"))
		invocations[i*3+2] = invocation(getUpdated, mcid(accountId, "2"))

		changesCommand = changes.GetCommand()
		getCommand = getCreated.GetCommand()
	}

	cmd, err := client.request(session, logger, using, invocations...)
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

			created := getMapper(createdResponse)
			updated := getMapper(updatedResponse)
			changesItemByAccount[accountId] = changesItemMapper(changesResponse.GetOldState(), changesResponse.GetNewState(), changesResponse.GetHasMoreChanges(), created, updated, changesResponse.GetDestroyed())
			stateByAccountId[accountId] = stateMapper(createdResponse)
		}
		return respMapper(changesItemByAccount), squashState(stateByAccountId), nil
	})
}

func updates[CHANGESREQ ChangesCommand, GETREQ GetCommand, CHANGESRESP ChangesResponse, GETRESP GetResponse, ITEM any, RESP any]( //NOSONAR
	client *Client, name string, using []JmapNamespace,
	changesCommandFactory func() CHANGESREQ,
	_ CHANGESRESP,
	getCommandFactory func(string, string) GETREQ,
	getMapper func(GETRESP) []ITEM,
	respMapper func(State, State, bool, []ITEM) RESP,
	session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (RESP, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)
	var zero RESP

	changes := changesCommandFactory()
	getUpdated := getCommandFactory("/updated", "0") //NOSONAR
	cmd, err := client.request(session, logger, using,
		invocation(changes, "0"),
		invocation(getUpdated, "1"),
	)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		var changesResponse CHANGESRESP
		err = retrieveChanges(logger, body, changes, "0", &changesResponse)
		if err != nil {
			return zero, "", err
		}

		var updatedResponse GETRESP
		err = retrieveGet(logger, body, getUpdated, "1", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return zero, "", err
		}

		updated := getMapper(updatedResponse)
		result := respMapper(changesResponse.GetOldState(), changesResponse.GetNewState(), changesResponse.GetHasMoreChanges(), updated)

		return result, changesResponse.GetNewState(), nil
	})
}

func update[CHANGES Change, SET SetCommand, GET GetCommand, RESP any, SETRESP SetResponse, GETRESP GetResponse](client *Client, name string, using []JmapNamespace, //NOSONAR
	setCommandFactory func(map[string]PatchObject) SET,
	getCommandFactory func(string) GET,
	notUpdatedExtractor func(SETRESP) map[string]SetError,
	objExtractor func(GETRESP) RESP,
	id string, changes CHANGES,
	session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (RESP, SessionState, State, Language, Error) {
	logger = client.logger(name, session, logger)
	update := setCommandFactory(map[string]PatchObject{id: changes.AsPatch()})
	get := getCommandFactory(id)
	cmd, err := client.request(session, logger, using, invocation(update, "0"), invocation(get, "1"))
	var zero RESP
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client.api, logger, ctx, session, client.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (RESP, State, Error) {
		var setResponse SETRESP
		err = retrieveSet(logger, body, update, "0", &setResponse)
		if err != nil {
			return zero, setResponse.GetNewState(), err
		}
		nc := notUpdatedExtractor(setResponse)
		setErr, notok := nc[id]
		if notok {
			logger.Error().Msgf("%T.NotUpdated returned an error %v", setResponse, setErr)
			return zero, "", setErrorError(setErr, update.GetObjectType())
		}
		var getResponse GETRESP
		err = retrieveGet(logger, body, get, "1", &getResponse)
		if err != nil {
			return zero, setResponse.GetNewState(), err
		}
		return objExtractor(getResponse), setResponse.GetNewState(), nil
	})
}
