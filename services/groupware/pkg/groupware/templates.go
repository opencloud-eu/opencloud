package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Create a new {{.Name}} using the JSON payload in the body if the `{{.Method}}` operation.
//
// When successful, it returns a `200 OK` with the {{.ObjType}} that was just created in the response.
func create[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]](
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	bodyFunc func(r Request, accountId string, body *CHANGE, ctx jmap.Context) (bool, Response),
	createFunc func(accountId string, change CHANGE, ctx jmap.Context) (*T, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		var create CHANGE
		err := req.body(&create)
		if err != nil {
			return req.error(accountId, err)
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)

		if bodyFunc != nil {
			if ok, resp := bodyFunc(req, accountId, &create, ctx); !ok {
				return resp
			}
		}

		created, sessionState, state, lang, jerr := createFunc(accountId, create, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, created, sessionState, o.responseType, state, lang)
	})
}

func getall[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], RESP jmap.GetResponse[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	getFunc func(accountId string, ids []string, ctx jmap.Context) (RESP, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		if notok, resp := req.unsupportedParams(single(accountId), QueryParamOffset, QueryParamLimit); notok {
			return resp
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		objs, sessionState, state, lang, jerr := getFunc(accountId, []string{}, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, objs, sessionState, o.responseType, state, lang)
	})
}

func getallpaged[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], RESP jmap.GetResponse[T], FILTER any, COMP any, SEARCHRESULTS jmap.SearchResults[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	getFunc func(accountId string, ids []string, ctx jmap.Context) (RESP, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
	filterFunc func(containerId string) FILTER,
	sortBy []COMP,
	queryFunc func(req Request, accountId string, filter FILTER, sortBy []COMP, offset int, limit uint, ctx jmap.Context) (SEARCHRESULTS, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		search := false
		offset, ok, err := req.parseIntParam(QueryParamOffset, 0)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			search = true
			l = l.Int(QueryParamOffset, offset)
		}

		limit, ok, err := req.parseUIntParam(QueryParamLimit, uint(0))
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			search = true
			l = l.Uint(QueryParamLimit, limit)
		}

		if search {
			containerId := ""
			if o.containerUriParamName != "" {
				var err *Error
				containerId, err = req.PathParam(o.containerUriParamName)
				if err != nil {
					return req.error(accountId, err)
				}
				l = l.Str(o.containerUriParamName, log.SafeString(containerId))
			}

			filter := filterFunc(containerId)

			logger := log.From(l)
			ctx := req.ctx.WithLogger(logger)
			results, sessionState, state, lang, jerr := queryFunc(req, accountId, filter, sortBy, offset, limit, ctx)
			if jerr != nil {
				return req.jmapError(accountId, jerr, sessionState, lang)
			}
			return req.respond(accountId, results, sessionState, o.responseType, state, lang)
		} else {
			logger := log.From(l)
			ctx := req.ctx.WithLogger(logger)
			objs, sessionState, state, lang, jerr := getFunc(accountId, []string{}, ctx)
			if jerr != nil {
				return req.jmapError(accountId, jerr, sessionState, lang)
			}
			return req.respond(accountId, objs, sessionState, o.responseType, state, lang)
		}
	})
}

func query[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], SEARCHRESULTS jmap.SearchResults[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	defaultLimit uint,
	queryFunc func(req Request, accountId string, containerId string, offset int, limit uint, ctx jmap.Context) (SEARCHRESULTS, jmap.SessionState, jmap.State, jmap.Language, *Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		containerId := ""
		if o.containerUriParamName != "" {
			var err *Error
			containerId, err = req.PathParam(o.containerUriParamName)
			if err != nil {
				return req.error(accountId, err)
			}
			l = l.Str(o.containerUriParamName, log.SafeString(containerId))
		}

		offset, ok, err := req.parseIntParam(QueryParamOffset, 0)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Int(QueryParamOffset, offset)
		}

		limit, ok, err := req.parseUIntParam(QueryParamLimit, defaultLimit)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Uint(QueryParamLimit, limit)
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)

		results, sessionState, state, lang, err := queryFunc(req, accountId, containerId, offset, limit, ctx)
		if err != nil {
			return req.error(accountId, err)
		}

		return req.respond(accountId, results, sessionState, o.responseType, state, lang)
	})
}

func get[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], RESP jmap.GetResponse[T]](
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	getFunc func(accountId string, ids []string, ctx jmap.Context) (RESP, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))
		ids := []string{}
		if o.uriParamName != "" {
			id, err := req.PathParamDoc(o.uriParamName, "The unique identifier of the object to retrieve")
			if err != nil {
				return req.error(accountId, err)
			}
			l.Str(o.uriParamName, log.SafeString(id))
			ids = single(id)
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		objs, sessionState, state, lang, jerr := getFunc(accountId, ids, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		n := len(objs.GetList())
		switch n {
		case 0:
			return req.notFound(accountId, sessionState, ContactResponseObjectType, state)
		case 1:
			return req.respond(accountId, objs.GetList()[0], sessionState, ContactResponseObjectType, state, lang)
		default:
			logger.Error().Msgf("found %d %s matching '%s' instead of 1", n, o.responseType, ids)
			return req.errorS(accountId, req.apiError(&ErrorMultipleIdMatches), sessionState)
		}
	})
}

func getFromMap[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], RESP jmap.GetResponse[T]](
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	getFunc func(accountIds []string, ids []string, ctx jmap.Context) (map[string]RESP, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))
		id, err := req.PathParamDoc(o.uriParamName, "The unique identifier of the object to retrieve")
		// TODO add id splitting
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(o.uriParamName, log.SafeString(id))

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		objMap, sessionState, state, lang, jerr := getFunc(single(accountId), single(id), ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if objs, ok := objMap[accountId]; ok {
			n := len(objs.GetList())
			switch n {
			case 0:
				return req.notFound(accountId, sessionState, ContactResponseObjectType, state)
			case 1:
				return req.respond(accountId, objs.GetList()[0], sessionState, ContactResponseObjectType, state, lang)
			default:
				logger.Error().Msgf("found %d %s matching '%s' instead of 1", n, o.responseType, id)
				return req.errorS(accountId, req.apiError(&ErrorMultipleIdMatches), sessionState)
			}
		} else {
			return req.notFound(accountId, sessionState, ContactResponseObjectType, state)
		}
	})
}

func changes[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]](
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	changesFunc func(accountId string, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (CHANGES, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		maxChanges, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}

		sinceState := jmap.State(req.OptHeaderParamDoc(HeaderParamSince, "Optionally specifies the state identifier from which on to list changes"))
		if sinceState != "" {
			l = l.Str(HeaderParamSince, log.SafeString(string(sinceState)))
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		changes, sessionState, state, lang, jerr := changesFunc(accountId, sinceState, maxChanges, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, changes, sessionState, o.responseType, state, lang)
	})
}

func delete[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	deleteFunc func(accountId string, ids []string, ctx jmap.Context) (map[string]jmap.SetError, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))
		id, err := req.PathParamDoc(o.uriParamName, "The unique identifier of the object to delete")
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(o.uriParamName, log.SafeString(id))

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		setErrors, sessionState, state, lang, jerr := deleteFunc(accountId, single(id), ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		for _, e := range setErrors {
			desc := e.Description
			if desc != "" {
				return req.error(accountId, apiError(
					req.errorId(),
					o.failedToDeleteError,
					withDetail(e.Description),
				))
			} else {
				return req.error(accountId, apiError(
					req.errorId(),
					o.failedToDeleteError,
				))
			}
		}
		return req.noContent(accountId, sessionState, o.responseType, state)
	})
}

func deleteMany[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	deleteFunc func(accountId string, ids []string, ctx jmap.Context) (map[string]jmap.SetError, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		ids := []string{}
		if o.uriParamName != "" {
			pathId, err := req.PathParam(o.uriParamName)
			if err != nil {
				return req.error(accountId, err)
			}
			if ok {
				ids = append(ids, pathId)
			}
		}
		{
			queryIds, ok, err := req.parseOptStringListParam(QueryParamId)
			if err != nil {
				return req.error(accountId, err)
			}
			if ok {
				ids = append(ids, queryIds...)
			}
		}
		{
			var bodyIds []string
			err := req.body(&bodyIds)
			if err != nil {
				return req.error(accountId, err)
			}
			ids = append(ids, bodyIds...)
		}
		switch len(ids) {
		case 0:
			return req.noop(accountId)
		case 1:
			l.Str("id", log.SafeString(ids[0]))
		default:
			l.Array("ids", log.SafeStringArray(ids))
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		setErrors, sessionState, state, lang, jerr := deleteFunc(accountId, ids, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		for _, e := range setErrors {
			desc := e.Description
			if desc != "" {
				return req.error(accountId, apiError(
					req.errorId(),
					o.failedToDeleteError,
					withDetail(e.Description),
				))
			} else {
				return req.error(accountId, apiError(
					req.errorId(),
					o.failedToDeleteError,
				))
			}
		}
		return req.noContent(accountId, sessionState, o.responseType, state)
	})
}

func modify[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]](
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	updateFunc func(accountId string, id string, change CHANGE, ctx jmap.Context) (T, jmap.SessionState, jmap.State, jmap.Language, jmap.Error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))
		id, err := req.PathParamDoc(o.uriParamName, "The unique identifier of the object to modify")
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(o.uriParamName, log.SafeString(id))

		var change CHANGE
		err = req.body(&change)
		if err != nil {
			return req.error(accountId, err)
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		updated, sessionState, state, lang, jerr := updateFunc(accountId, id, change, ctx)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, updated, sessionState, o.responseType, state, lang)
	})
}
