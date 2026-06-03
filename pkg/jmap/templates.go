package jmap

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/rs/zerolog"
)

func get[T Foo, GETREQ GetCommand[T], GETRESP GetResponse[T], ID any, RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	getCommandFactory func(AccountId, []ID) GETREQ,
	_ GETRESP,
	mapper func(GETRESP) RESP,
	accountId AccountId, ids []ID, ctx Context) (Result[RESP], Error) {
	ctx = ctx.WithLogger(client.logger(name, ctx))

	get := getCommandFactory(accountId, ids)
	cmd, err := client.request(ctx, objType.Namespaces, invocation(get, "0"))
	if err != nil {
		return ZeroResult[RESP](), err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		var response GETRESP
		err = retrieveGet(ctx, body, get, "0", &response)
		if err != nil {
			var zero RESP
			return zero, "", err
		}

		return mapper(response), response.GetState(), nil
	})
}

func getAN[T Foo, GETREQ GetCommand[T], GETRESP GetResponse[T], ID any, RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	getCommandFactory func(AccountId, []ID) GETREQ,
	resp GETRESP,
	respMapper func(map[AccountId][]T) RESP,
	accountIds []AccountId, ids []ID, ctx Context) (Result[RESP], Error) {
	return getN(client, name, objType, getCommandFactory, resp,
		func(r GETRESP) []T { return r.GetList() },
		respMapper,
		accountIds, ids,
		ctx,
	)
}

func getN[T Foo, ITEM any, GETREQ GetCommand[T], GETRESP GetResponse[T], ID any, RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	getCommandFactory func(AccountId, []ID) GETREQ,
	_ GETRESP,
	itemMapper func(GETRESP) ITEM,
	respMapper func(map[AccountId]ITEM) RESP,
	accountIds []AccountId, ids []ID, ctx Context) (Result[RESP], Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

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
		return ZeroResult[RESP](), err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		result := map[AccountId]ITEM{}
		responses := map[AccountId]GETRESP{}
		for _, accountId := range uniqueAccountIds {
			var resp GETRESP
			err = retrieveResponseMatchParameters(ctx, body, c, mcid(accountId, "0"), &resp)
			if err != nil {
				var zero RESP
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
	setCommandFactory func(AccountId, map[string]C) SETREQ,
	getCommandFactory func(AccountId, string) GETREQ,
	createdMapper func(SETRESP) map[string]*T,
	listMapper func(GETRESP) []T,
	accountId AccountId, create C,
	ctx Context) (Result[*T], Error) {
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
		return ZeroResult[*T](), err
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
	setCommandFactory func(AccountId, []string) REQ, _ RESP,
	accountId AccountId, destroy []string, ctx Context) (Result[map[string]SetError], Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	set := setCommandFactory(accountId, destroy)
	cmd, err := client.request(ctx, objType.Namespaces,
		invocation(set, "0"),
	)
	if err != nil {
		return ZeroResult[map[string]SetError](), err
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
	ctx Context) (Result[RESP], Error) {
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
	ctx Context) (Result[RESP], Error) {
	logger := client.logger(name, ctx)

	changes := changesCommandFactory()
	getCreated := getCommandFactory("/created", "0") //NOSONAR
	getUpdated := getCommandFactory("/updated", "0") //NOSONAR

	cmd, err := client.request(ctx.WithLogger(logger), objType.Namespaces,
		invocation(changes, "0"),
		invocation(getCreated, "1"),
		invocation(getUpdated, "2"),
	)
	if err != nil {
		return ZeroResult[RESP](), err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		var changesResponse CHANGESRESP
		err = retrieveChanges(ctx, body, changes, "0", &changesResponse)
		if err != nil {
			var zero RESP
			return zero, "", err
		}

		var createdResponse GETRESP
		err = retrieveGet(ctx, body, getCreated, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			var zero RESP
			return zero, "", err
		}

		var updatedResponse GETRESP
		err = retrieveGet(ctx, body, getUpdated, "2", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			var zero RESP
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
	accountIds []AccountId, sinceStateMap map[AccountId]State,
	changesCommandFactory func(AccountId, State) CHANGESREQ,
	_ CHANGESRESP,
	getCommandFactory func(AccountId, string, string) GETREQ,
	getMapper func(GETRESP) []ITEM,
	changesItemMapper func(State, State, bool, []ITEM, []ITEM, []string) CHANGESITEM,
	respMapper func(map[AccountId]CHANGESITEM) RESP,
	stateMapper func(GETRESP) State,
	ctx Context) (Result[RESP], Error) {
	logger := client.loggerParams(name, ctx, func(z zerolog.Context) zerolog.Context {
		sinceStateLogDict := zerolog.Dict()
		for k, v := range sinceStateMap {
			sinceStateLogDict.Str(log.SafeString(string(k)), log.SafeString(string(v)))
		}
		return z.Dict(logSinceState, sinceStateLogDict)
	})

	uniqueAccountIds := structs.Uniq(accountIds)
	n := len(uniqueAccountIds)
	if n < 1 {
		return ZeroResult[RESP](), nil
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
		return ZeroResult[RESP](), err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		changesItemByAccount := make(map[AccountId]CHANGESITEM, n)
		stateByAccountId := make(map[AccountId]State, n)
		for _, accountId := range uniqueAccountIds {
			var changesResponse CHANGESRESP
			err = retrieveChanges(ctx, body, ch, mcid(accountId, "0"), &changesResponse)
			if err != nil {
				var zero RESP
				return zero, "", err
			}

			var createdResponse GETRESP
			err = retrieveGet(ctx, body, gc, mcid(accountId, "1"), &createdResponse)
			if err != nil {
				var zero RESP
				return zero, "", err
			}

			var updatedResponse GETRESP
			err = retrieveGet(ctx, body, gu, mcid(accountId, "2"), &updatedResponse)
			if err != nil {
				var zero RESP
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
	ctx Context) (Result[RESP], Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	changes := changesCommandFactory()
	getUpdated := getCommandFactory("/updated", "0") //NOSONAR
	cmd, err := client.request(ctx, objType.Namespaces,
		invocation(changes, "0"),
		invocation(getUpdated, "1"),
	)
	if err != nil {
		return ZeroResult[RESP](), err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		var changesResponse CHANGESRESP
		err = retrieveChanges(ctx, body, changes, "0", &changesResponse)
		if err != nil {
			var zero RESP
			return zero, "", err
		}

		var updatedResponse GETRESP
		err = retrieveGet(ctx, body, getUpdated, "1", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			var zero RESP
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
	ctx Context) (Result[RESP], Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	var update SET
	{
		patch, err := changes.AsPatch()
		if err != nil {
			return ZeroResult[RESP](), jmapError(err, JmapErrorPatchObjectSerialization)
		}
		update = setCommandFactory(map[string]PatchObject{id: patch})
	}
	get := getCommandFactory(id)
	cmd, err := client.request(ctx, objType.Namespaces, invocation(update, "0"), invocation(get, "1"))
	if err != nil {
		return ZeroResult[RESP](), err
	}

	return command(client, ctx, cmd, func(body *Response) (RESP, State, Error) {
		var setResponse SETRESP
		err = retrieveSet(ctx, body, update, "0", &setResponse)
		if err != nil {
			var zero RESP
			return zero, setResponse.GetNewState(), err
		}
		nc := notUpdatedExtractor(setResponse)
		setErr, notok := nc[id]
		if notok {
			logger.Error().Msgf("%T.NotUpdated returned an error %v", setResponse, setErr)
			var zero RESP
			return zero, "", setErrorError(setErr, update.GetObjectType())
		}
		var getResponse GETRESP
		err = retrieveGet(ctx, body, get, "1", &getResponse)
		if err != nil {
			var zero RESP
			return zero, setResponse.GetNewState(), err
		}
		return objExtractor(getResponse), setResponse.GetNewState(), nil
	})
}

/*
func query[T Foo, FILTER any, SORT any, QUERY QueryCommand[T], GET GetCommand[T], QUERYRESP QueryResponse[T], GETRESP GetResponse[T], RESP SearchResults[T]]( //NOSONAR
	client *Client, name string, objType ObjectType,
	defaultSortBy []SORT,
	queryCommandFactory func(filter FILTER, sortBy []SORT) QUERY,
	getCommandFactory func(cmd Command, path string, rof string) GET,
	respMapper0 func(query QUERYRESP) *RESP,
	respMapper func(query QUERYRESP, get GETRESP) *RESP,
	filter FILTER, sortBy []SORT,
	ctx Context) (Result[*RESP], Error) {

	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	if sortBy == nil {
		sortBy = defaultSortBy
	}

	query := queryCommandFactory(filter, sortBy)
	var get GET
	limit := query.GetLimit()
	if limit != nil && *limit == 0 {
		query.SetLimit(UintPtrOne)
	} else {
		get = getCommandFactory(query.GetCommand(), "/ids/*", "0")
	}

	cmd, err := client.request(ctx, objType.Namespaces, invocation(query, "0"), invocation(get, "1"))
	if err != nil {
		return ZeroResult[*RESP](), err
	}

	return command(client, ctx, cmd, func(body *Response) (*RESP, State, Error) {
		var queryResponse QUERYRESP
		err = retrieveQuery(ctx, body, query, "0", &queryResponse)
		if err != nil {
			return nil, EmptyState, err
		}
		if limit == nil && *limit == 0 {
			result := respMapper0(queryResponse)
			if query.GetAnchor() != "" && (*result).GetPosition() != nil && *(*result).GetPosition() == 0 {
				(*result).SetPosition(nil)
			}
			return result, queryResponse.GetQueryState(), nil
		} else {
			var getResponse GETRESP
			err = retrieveGet(ctx, body, get, "1", &getResponse)
			if err != nil {
				return nil, EmptyState, err
			}
			return respMapper(queryResponse, getResponse), queryResponse.GetQueryState(), nil
		}
	})
}
*/

func queryN[T Foo, FILTER any, SORT any, QUERY QueryCommand[T, QUERY], GET GetCommand[T], QUERYRESP QueryResponse[T], GETRESP GetResponse[T], RESP any]( //NOSONAR
	client *Client, name string, objType ObjectType,
	defaultSortBy []SORT,
	queryCommandFactory func(accountId AccountId, queryParams QueryParams, limit *uint, filter FILTER, sortBy []SORT) QUERY,
	getCommandFactory func(accountId AccountId, cmd Command, path string, rof string) GET,
	respMapper0 func(query QUERYRESP, queryParams QueryParams, limit *uint) *RESP,
	respMapper func(query QUERYRESP, get GETRESP, queryParams QueryParams, limit *uint) *RESP,
	accountIds map[AccountId]QueryParams,
	limit *uint, filter FILTER, sortBy []SORT,
	ctx Context) (Result[map[AccountId]*RESP], Error) {
	logger := client.logger(name, ctx)
	ctx = ctx.WithLogger(logger)

	if sortBy == nil {
		sortBy = defaultSortBy
	}

	invocations := make([]Invocation, len(accountIds)*2)
	var g GET
	var q QUERY
	{
		i := 0
		for accountId, queryParams := range accountIds {
			query := queryCommandFactory(accountId, queryParams, limit, filter, sortBy)
			q = query
			invocations[i*2+0] = invocation(query, mcid(accountId, "0"))
			if limit != nil && *limit == 0 {
				query = query.WithLimit(UintPtrOne)
				invocations[i*2+1] = skipInvocation()
			} else {
				get := getCommandFactory(accountId, query.GetCommand(), "/ids/*", mcid(accountId, "0"))
				invocations[i*2+1] = invocation(get, mcid(accountId, "1"))
				g = get
			}
			i++
		}
	}

	cmd, err := client.request(ctx, objType.Namespaces, invocations...)
	if err != nil {
		return ZeroResult[map[AccountId]*RESP](), err
	}

	return command(client, ctx, cmd, func(body *Response) (map[AccountId]*RESP, State, Error) {
		resp := map[AccountId]*RESP{}
		stateByAccountId := map[AccountId]State{}
		for accountId, queryParams := range accountIds {
			var queryResponse QUERYRESP
			err = retrieveQuery(ctx, body, q, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, EmptyState, err
			}
			if limit != nil && *limit == 0 {
				resp[accountId] = respMapper0(queryResponse, queryParams, limit)
				stateByAccountId[accountId] = queryResponse.GetQueryState()
			} else {
				var getResponse GETRESP
				err = retrieveGet(ctx, body, g, mcid(accountId, "1"), &getResponse)
				if err != nil {
					return nil, EmptyState, err
				}
				if len(getResponse.GetNotFound()) > 0 {
					// TODO what to do when there are not-found calendarevents here? potentially nothing, they could have been deleted between query and get?
				}
				resp[accountId] = respMapper(queryResponse, getResponse, queryParams, limit)
				stateByAccountId[accountId] = getResponse.GetState()
			}
		}
		return resp, squashState(stateByAccountId), nil
	})
}
