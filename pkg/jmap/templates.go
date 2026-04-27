package jmap

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/rs/zerolog"
)

func get[T Foo, GETREQ GetCommand[T], GETRESP GetResponse[T], RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	getCommandFactory func(string, []string) GETREQ,
	_ GETRESP,
	mapper func(GETRESP) RESP,
	accountId string, ids []string, ctx Context) (RESP, SessionState, State, Language, Error) {
	ctx = ctx.WithLogger(client.logger(name, ctx))

	var zero RESP

	get := getCommandFactory(accountId, ids)
	cmd, err := client.request(ctx, objType.Namespaces, invocation(get, "0"))
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		var response GETRESP
		err = retrieveGet(ctx, body, get, "0", &response)
		if err != nil {
			return zero, "", err
		}

		return mapper(response), response.GetState(), nil
	})
}

func getAN[T Foo, GETREQ GetCommand[T], GETRESP GetResponse[T], RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	getCommandFactory func(string, []string) GETREQ,
	resp GETRESP,
	respMapper func(map[string][]T) RESP,
	accountIds []string, ids []string, ctx Context) (RESP, SessionState, State, Language, Error) {
	return getN(client, name, objType, getCommandFactory, resp,
		func(r GETRESP) []T { return r.GetList() },
		respMapper,
		accountIds, ids,
		ctx,
	)
}

func getN[T Foo, ITEM any, GETREQ GetCommand[T], GETRESP GetResponse[T], RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	getCommandFactory func(string, []string) GETREQ,
	_ GETRESP,
	itemMapper func(GETRESP) ITEM,
	respMapper func(map[string]ITEM) RESP,
	accountIds []string, ids []string, ctx Context) (RESP, SessionState, State, Language, Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	var zero RESP

	uniqueAccountIds := structs.Uniq(accountIds)

	invocations := make([]Invocation, len(uniqueAccountIds))
	var c Command
	for i, accountId := range uniqueAccountIds {
		get := getCommandFactory(accountId, ids)
		c = get.GetCommand()
		invocations[i] = invocation(get, mcid(accountId, "0"))
	}

	cmd, err := client.request(ctx, objType.Namespaces, invocations...)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		result := map[string]ITEM{}
		responses := map[string]GETRESP{}
		for _, accountId := range uniqueAccountIds {
			var resp GETRESP
			err = retrieveResponseMatchParameters(ctx, body, c, mcid(accountId, "0"), &resp)
			if err != nil {
				return zero, "", err
			}
			responses[accountId] = resp
			result[accountId] = itemMapper(resp)
		}
		return respMapper(result), squashStateFunc(responses, func(r GETRESP) State { return r.GetState() }), nil
	})
}

func create[T Foo, C any, SETREQ SetCommand[T], GETREQ GetCommand[T], SETRESP SetResponse[T], GETRESP GetResponse[T]]( //NOSONAR
	client *Client, name string, objType ObjectType,
	setCommandFactory func(string, map[string]C) SETREQ,
	getCommandFactory func(string, string) GETREQ,
	createdMapper func(SETRESP) map[string]*T,
	listMapper func(GETRESP) []T,
	accountId string, create C,
	ctx Context) (*T, SessionState, State, Language, Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	createMap := map[string]C{"c": create}
	get := getCommandFactory(accountId, "#c")
	set := setCommandFactory(accountId, createMap)
	cmd, err := client.request(ctx, objType.Namespaces,
		invocation(set, "0"),
		invocation(get, "1"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (*T, State, Error) {
		var setResponse SETRESP
		err = retrieveSet(ctx, body, set, "0", &setResponse)
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
		err = retrieveGet(ctx, body, get, "1", &getResponse)
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

func destroy[T Foo, REQ SetCommand[T], RESP SetResponse[T]](client *Client, name string, objType ObjectType, //NOSONAR
	setCommandFactory func(string, []string) REQ, _ RESP,
	accountId string, destroy []string, ctx Context) (map[string]SetError, SessionState, State, Language, Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	set := setCommandFactory(accountId, destroy)
	cmd, err := client.request(ctx, objType.Namespaces,
		invocation(set, "0"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (map[string]SetError, State, Error) {
		var setResponse RESP
		err = retrieveSet(ctx, body, set, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}
		return setResponse.GetNotDestroyed(), setResponse.GetNewState(), nil
	})
}

func changesA[T Foo, CHANGESREQ ChangesCommand[T], GETREQ GetCommand[T], CHANGESRESP ChangesResponse[T], GETRESP GetResponse[T], RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	changesCommandFactory func() CHANGESREQ,
	changesResp CHANGESRESP,
	_ GETRESP,
	getCommandFactory func(string, string) GETREQ,
	respMapper func(State, State, bool, []T, []T, []string) RESP,
	ctx Context) (RESP, SessionState, State, Language, Error) {

	return changes(client, name, objType, changesCommandFactory, changesResp, getCommandFactory,
		func(r GETRESP) []T { return r.GetList() },
		respMapper,
		ctx,
	)
}

func changes[T Foo, CHANGESREQ ChangesCommand[T], GETREQ GetCommand[T], CHANGESRESP ChangesResponse[T], GETRESP GetResponse[T], ITEM any, RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	changesCommandFactory func() CHANGESREQ,
	_ CHANGESRESP,
	getCommandFactory func(string, string) GETREQ,
	getMapper func(GETRESP) []ITEM,
	respMapper func(State, State, bool, []ITEM, []ITEM, []string) RESP,
	ctx Context) (RESP, SessionState, State, Language, Error) {
	logger := client.logger(name, ctx)
	var zero RESP

	changes := changesCommandFactory()
	getCreated := getCommandFactory("/created", "0") //NOSONAR
	getUpdated := getCommandFactory("/updated", "0") //NOSONAR

	cmd, err := client.request(ctx.WithLogger(logger), objType.Namespaces,
		invocation(changes, "0"),
		invocation(getCreated, "1"),
		invocation(getUpdated, "2"),
	)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		var changesResponse CHANGESRESP
		err = retrieveChanges(ctx, body, changes, "0", &changesResponse)
		if err != nil {
			return zero, "", err
		}

		var createdResponse GETRESP
		err = retrieveGet(ctx, body, getCreated, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return zero, "", err
		}

		var updatedResponse GETRESP
		err = retrieveGet(ctx, body, getUpdated, "2", &updatedResponse)
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

func changesN[T Foo, CHANGESREQ ChangesCommand[T], GETREQ GetCommand[T], CHANGESRESP ChangesResponse[T], GETRESP GetResponse[T], ITEM any, CHANGESITEM any, RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	accountIds []string, sinceStateMap map[string]State,
	changesCommandFactory func(string, State) CHANGESREQ,
	_ CHANGESRESP,
	getCommandFactory func(string, string, string) GETREQ,
	getMapper func(GETRESP) []ITEM,
	changesItemMapper func(State, State, bool, []ITEM, []ITEM, []string) CHANGESITEM,
	respMapper func(map[string]CHANGESITEM) RESP,
	stateMapper func(GETRESP) State,
	ctx Context) (RESP, SessionState, State, Language, Error) {
	logger := client.loggerParams(name, ctx, func(z zerolog.Context) zerolog.Context {
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
	var ch CHANGESREQ
	var gc GETREQ
	var gu GETREQ
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

		ch = changes
		gc = getCreated
		gu = getUpdated
	}

	ctx = ctx.WithLogger(logger)

	cmd, err := client.request(ctx, objType.Namespaces, invocations...)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		changesItemByAccount := make(map[string]CHANGESITEM, n)
		stateByAccountId := make(map[string]State, n)
		for _, accountId := range uniqueAccountIds {
			var changesResponse CHANGESRESP
			err = retrieveChanges(ctx, body, ch, mcid(accountId, "0"), &changesResponse)
			if err != nil {
				return zero, "", err
			}

			var createdResponse GETRESP
			err = retrieveGet(ctx, body, gc, mcid(accountId, "1"), &createdResponse)
			if err != nil {
				return zero, "", err
			}

			var updatedResponse GETRESP
			err = retrieveGet(ctx, body, gu, mcid(accountId, "2"), &updatedResponse)
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

func updates[T Foo, CHANGESREQ ChangesCommand[T], GETREQ GetCommand[T], CHANGESRESP ChangesResponse[T], GETRESP GetResponse[T], ITEM any, RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	changesCommandFactory func() CHANGESREQ,
	_ CHANGESRESP,
	getCommandFactory func(string, string) GETREQ,
	getMapper func(GETRESP) []ITEM,
	respMapper func(State, State, bool, []ITEM) RESP,
	ctx Context) (RESP, SessionState, State, Language, Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)
	var zero RESP

	changes := changesCommandFactory()
	getUpdated := getCommandFactory("/updated", "0") //NOSONAR
	cmd, err := client.request(ctx, objType.Namespaces,
		invocation(changes, "0"),
		invocation(getUpdated, "1"),
	)
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		var changesResponse CHANGESRESP
		err = retrieveChanges(ctx, body, changes, "0", &changesResponse)
		if err != nil {
			return zero, "", err
		}

		var updatedResponse GETRESP
		err = retrieveGet(ctx, body, getUpdated, "1", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return zero, "", err
		}

		updated := getMapper(updatedResponse)
		result := respMapper(changesResponse.GetOldState(), changesResponse.GetNewState(), changesResponse.GetHasMoreChanges(), updated)

		return result, changesResponse.GetNewState(), nil
	})
}

func update[T Foo, CHANGES Change, SET SetCommand[T], GET GetCommand[T], RESP any, SETRESP SetResponse[T], GETRESP GetResponse[T]]( //NOSONAR
	client *Client, name string, objType ObjectType,
	setCommandFactory func(map[string]PatchObject) SET,
	getCommandFactory func(string) GET,
	notUpdatedExtractor func(SETRESP) map[string]SetError,
	objExtractor func(GETRESP) RESP,
	id string, changes CHANGES,
	ctx Context) (RESP, SessionState, State, Language, Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	var zero RESP

	var update SET
	{
		patch, err := changes.AsPatch()
		if err != nil {
			return zero, "", "", "", jmapError(err, JmapPatchObjectSerialization)
		}
		update = setCommandFactory(map[string]PatchObject{id: patch})
	}
	get := getCommandFactory(id)
	cmd, err := client.request(ctx, objType.Namespaces, invocation(update, "0"), invocation(get, "1"))
	if err != nil {
		return zero, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		var setResponse SETRESP
		err = retrieveSet(ctx, body, update, "0", &setResponse)
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
		err = retrieveGet(ctx, body, get, "1", &getResponse)
		if err != nil {
			return zero, setResponse.GetNewState(), err
		}
		return objExtractor(getResponse), setResponse.GetNewState(), nil
	})
}

func query[T Foo, FILTER any, SORT any, QUERY QueryCommand[T], GET GetCommand[T], QUERYRESP QueryResponse[T], GETRESP GetResponse[T], RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	defaultSortBy []SORT,
	queryCommandFactory func(filter FILTER, sortBy []SORT, position int, anchor string, anchorOffset *int, limit *uint) QUERY,
	getCommandFactory func(cmd Command, path string, rof string) GET,
	respMapper func(query QUERYRESP, get GETRESP) *RESP,
	filter FILTER, sortBy []SORT, position int, anchor string, anchorOffset *int, limit *uint,
	ctx Context) (*RESP, SessionState, State, Language, Error) {

	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	if sortBy == nil {
		sortBy = defaultSortBy
	}

	query := queryCommandFactory(filter, sortBy, position, anchor, anchorOffset, limit)
	get := getCommandFactory(query.GetCommand(), "/ids/*", "0")

	cmd, err := client.request(ctx, objType.Namespaces, invocation(query, "0"), invocation(get, "1"))
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (*RESP, State, Error) {
		var queryResponse QUERYRESP
		err = retrieveQuery(ctx, body, query, "0", &queryResponse)
		if err != nil {
			return nil, EmptyState, err
		}
		var getResponse GETRESP
		err = retrieveGet(ctx, body, get, "1", &getResponse)
		if err != nil {
			return nil, EmptyState, err
		}
		return respMapper(queryResponse, getResponse), queryResponse.GetQueryState(), nil
	})
}

func queryN[T Foo, FILTER any, SORT any, QUERY QueryCommand[T], GET GetCommand[T], QUERYRESP QueryResponse[T], GETRESP GetResponse[T], RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	defaultSortBy []SORT,
	queryCommandFactory func(accountId string, filter FILTER, sortBy []SORT, position int, anchor string, anchorOffset *int, imit *uint) QUERY,
	getCommandFactory func(accountId string, cmd Command, path string, rof string) GET,
	respMapper func(query QUERYRESP, get GETRESP) *RESP,
	accountIds []string,
	filter FILTER, sortBy []SORT, position int, anchor string, anchorOffset *int, limit *uint,
	ctx Context) (map[string]*RESP, SessionState, State, Language, Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	if sortBy == nil {
		sortBy = defaultSortBy
	}

	invocations := make([]Invocation, len(uniqueAccountIds)*2)
	var g GET
	var q QUERY
	for i, accountId := range uniqueAccountIds {
		query := queryCommandFactory(accountId, filter, sortBy, position, anchor, anchorOffset, limit)
		get := getCommandFactory(accountId, query.GetCommand(), "/ids/*", mcid(accountId, "0"))
		invocations[i*2+0] = invocation(query, mcid(accountId, "0"))
		invocations[i*2+1] = invocation(get, mcid(accountId, "1"))
		q = query
		g = get
	}

	cmd, err := client.request(ctx, objType.Namespaces, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(client, ctx, cmd, func(body *Response) (map[string]*RESP, State, Error) {
		resp := map[string]*RESP{}
		stateByAccountId := map[string]State{}
		for _, accountId := range uniqueAccountIds {
			var queryResponse QUERYRESP
			err = retrieveQuery(ctx, body, q, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, "", err
			}
			var getResponse GETRESP
			err = retrieveGet(ctx, body, g, mcid(accountId, "1"), &getResponse)
			if err != nil {
				return nil, "", err
			}
			if len(getResponse.GetNotFound()) > 0 {
				// TODO what to do when there are not-found calendarevents here? potentially nothing, they could have been deleted between query and get?
			}
			resp[accountId] = respMapper(queryResponse, getResponse)
			stateByAccountId[accountId] = getResponse.GetState()
		}
		return resp, squashState(stateByAccountId), nil
	})
}
