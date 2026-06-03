package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Create a new {{.Name}} using the JSON payload in the body of the `{{.Verb}}` operation.
// @api:response 200:T returns the {{.Name}} that was just created
// @api:body CHANGE the {{.Name}} to create
func create[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	bodyFunc func(r Request, accountId string, body *CHANGE, ctx jmap.Context) (bool, Response),
	createFunc func(accountId string, change CHANGE, ctx jmap.Context) (jmap.Result[*T], error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		if notok, resp := req.unsupportedQueryParams(single(accountId), noSupportedQueryParams); notok {
			return resp
		}

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

		if result, err := createFunc(accountId, create, ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			return req.respond(accountId, result.Payload, o.responseType, result)
		}
	})
}

// Retrieve all the {{.Name}}.
// @api:response 200:[]T returns all the {{.Names}}
func getall[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], RESP jmap.GetResponse[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	getFunc func(accountId string, ids []string, ctx jmap.Context) (jmap.Result[RESP], error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		if notok, resp := req.unsupportedQueryParams(single(accountId), noSupportedQueryParams); notok {
			return resp
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		if result, err := getFunc(accountId, []string{}, ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			return req.respond(accountId, result.Payload, o.responseType, result)
		}
	})
}

// var paginationQueryParams = toSupportedQueryParams(QueryParamPosition, QueryParamAnchor, QueryParamAnchorOffset, QueryParamLimit)
var firstQueryParams = toSupportedQueryParams(QueryParamLimit)
var nextQueryParams = toSupportedQueryParams(QueryParamNext, QueryParamLimit)
var noQueryParams = toSupportedQueryParams()

// Retrieve all the {{.Name}} with support for paging using the {{.QueryParam.QueryParamPosition.Name}} and {{.QueryParam.QueryParamLimit.Name}} query parameters.
// @api:response 200:SEARCHRESULTS returns the {{.Names}} within the requested range, as well as the total amount of {{.Names}}
func getallpaged[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], FILTER any, COMP any, SEARCHRESULTS jmap.SearchResults[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	withContainerId bool,
	filterFunc func(containerId string) FILTER,
	sortBy []COMP,
	queryFunc func(req Request, accountIds []string, qps QueryParamsSupplier, limit *uint, filter FILTER, sortBy []COMP, ctx jmap.Context) (jmap.Result[SEARCHRESULTS], NextToken, error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		var limit *uint = nil
		{
			v, ok, err := req.parseUIntParam(QueryParamLimit, uint(0))
			if err != nil {
				return req.error(accountId, err)
			}
			if ok {
				l = l.Uint(QueryParamLimit, v)
				limit = &v
			}
		}

		var qps QueryParamsSupplier
		var supportedQueryParams = noQueryParams
		{
			if s, ok := req.getStringParam(QueryParamNext, ""); ok {
				supportedQueryParams = nextQueryParams
				if n, err := unnext(NextToken(s)); err != nil {
					errorId := req.errorId()
					return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(err.Error()))) // TODO replace ErrorGeneric
				} else {
					qps = n
				}
			} else {
				supportedQueryParams = firstQueryParams
				qps = FirstQueryParamsSupplier{}

				/*
					position, ok, err := req.parseIntParam(QueryParamPosition, 0)
					if err != nil {
						return req.error(accountId, err)
					}
					if ok {
						l = l.Int(QueryParamPosition, position)
					}

					anchor, ok := req.getStringParam(QueryParamAnchor, "")
					if ok {
						l = l.Str(QueryParamAnchor, log.SafeString(anchor))
					}

					var anchorOffset *int = nil
					{
						v, ok, err := req.parseIntParam(QueryParamAnchorOffset, 0)
						if err != nil {
							return req.error(accountId, err)
						}
						if ok {
							l = l.Int(QueryParamAnchorOffset, v)
							anchorOffset = &v
						}
					}
				*/

			}
		}

		containerId := ""
		if withContainerId && o.containerUriParamName != "" {
			var err *Error
			containerId, err = req.PathParam(o.containerUriParamName)
			if err != nil {
				return req.error(accountId, err)
			}
			l = l.Str(o.containerUriParamName, log.SafeString(containerId))
		}

		if notok, resp := req.unsupportedQueryParams(single(accountId), supportedQueryParams); notok {
			return resp
		}

		filter := filterFunc(containerId)

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		if result, nextNextToken, err := queryFunc(req, single(accountId), qps, limit, filter, sortBy, ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			return req.respondNext(accountId, result.Payload, o.responseType, result, nextNextToken)
		}
	})
}

// Query all the {{.Name}} with support for paging using the {{.QueryParam.QueryParamPosition.Name}} and {{.QueryParam.QueryParamLimit.Name}} query parameters.
// @api:response 200:SEARCHRESULTS returns the {{.Names}} that match the filter, within the requested range, as well as the total amount of matches
func query[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], SEARCHRESULTS jmap.SearchResults[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	defaultLimit *uint,
	queryFunc func(req Request, accountId string, containerId string, qp jmap.QueryParams, limit *uint, ctx jmap.Context) (jmap.Result[SEARCHRESULTS], error),
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

		var limit *uint = nil
		{
			v, ok, err := req.parseUIntParam(QueryParamLimit, 0)
			if err != nil {
				return req.error(accountId, err)
			}
			if ok {
				l = l.Uint(QueryParamLimit, v)
				limit = &v
			} else {
				limit = defaultLimit
			}
		}

		var qp jmap.QueryParams
		var supportedQueryParams = noQueryParams
		{
			if s, ok := req.getStringParam(QueryParamNext, ""); ok {
				supportedQueryParams = nextQueryParams
				if n, err := unnext(NextToken(s)); err != nil {
					errorId := req.errorId()
					return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(err.Error()))) // TODO replace ErrorGeneric
				} else {
					if q, ok, err := n.ForAccountId(accountId); err != nil {
						errorId := req.errorId()
						return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(err.Error()))) // TODO replace ErrorGeneric
					} else if ok {
						qp = q
					} else {
						qp = jmap.NullQueryParams
					}
				}
			} else {
				supportedQueryParams = firstQueryParams
				qp = jmap.NullQueryParams
			}
		}

		/*
			position, ok, err := req.parseIntParam(QueryParamPosition, 0)
			if err != nil {
				return req.error(accountId, err)
			}
			if ok {
				l = l.Int(QueryParamPosition, position)
			}

			anchor, ok := req.getStringParam(QueryParamAnchor, "")
			if ok {
				l = l.Str(QueryParamAnchor, log.SafeString(anchor))
			}

			var anchorOffset *int = nil
			{
				v, ok, err := req.parseIntParam(QueryParamAnchorOffset, 0)
				if err != nil {
					return req.error(accountId, err)
				}
				if ok {
					l = l.Int(QueryParamAnchorOffset, v)
					anchorOffset = &v
				}
			}

			var limit *uint = nil
			{
				v, ok, err := req.parseUIntParam(QueryParamLimit, defaultLimit)
				if err != nil {
					return req.error(accountId, err)
				}
				if ok {
					l = l.Uint(QueryParamLimit, v)
					limit = &v
				} else if defaultLimit > 0 {
					limit = &defaultLimit
				}
			}
		*/

		if notok, resp := req.unsupportedQueryParams(single(accountId), supportedQueryParams); notok {
			return resp
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)

		/*
			jmaplimit := limit
			if limit != nil && *limit == 0 {
				jmaplimit = UintPtrOne
			}
		*/

		if result, err := queryFunc(req, accountId, containerId, qp, limit, ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			/*
				if limit != nil && *limit == 0 {
					result.Payload.RemoveResults()
					result.Payload.SetLimit(UintPtrZero)
				}
				if anchor != "" && result.Payload.GetPosition() != nil && *result.Payload.GetPosition() == 0 {
					result.Payload.SetPosition(nil)
				}
			*/

			return req.respond(accountId, result.Payload, o.responseType, result)
		}
	})
}

// Retrieve a specific {{.Name}} referenced by its unique identifier as specified in the path parameter `{{.UriParamName}}` in the path `{{.Path}}`
// @api:response 200:T returns the {{.Name}} that matches the requested identifier, if it exists
// @api:response 404 when there is no {{.Name}} for the requested identifier
func get[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], RESP jmap.GetResponse[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	getFunc func(accountId string, ids []string, ctx jmap.Context) (jmap.Result[RESP], error),
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

		if notok, resp := req.unsupportedQueryParams(single(accountId), noSupportedQueryParams); notok {
			return resp
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		if result, err := getFunc(accountId, ids, ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			n := len(result.Payload.GetList())
			switch n {
			case 0:
				return req.notFound(accountId, ContactResponseObjectType, result)
			case 1:
				return req.respond(accountId, result.Payload.GetList()[0], ContactResponseObjectType, result)
			default:
				logger.Error().Msgf("found %d %s matching '%s' instead of 1", n, o.responseType, ids)
				return req.errorS(accountId, req.apiError(&ErrorMultipleIdMatches), result)
			}
		}
	})
}

// Retrieve a specific {{.Name}} referenced by its unique identifier as specified in the path parameter `{{.UriParamName}}` in the path `{{.Path}}`
// @api:response 200:T returns the {{.Name}} that matches the requested identifier, if it exists
// @api:response 404 when there is no {{.Name}} for the requested identifier
func getFromMap[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T], RESP jmap.GetResponse[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	getFunc func(accountIds []string, ids []string, ctx jmap.Context) (jmap.Result[map[string]RESP], error),
) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := o.accountFunc(&req)
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))
		id, err := req.PathParamDoc(o.uriParamName, "The unique identifier of the object to retrieve")
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(o.uriParamName, log.SafeString(id))

		if notok, resp := req.unsupportedQueryParams(single(accountId), noSupportedQueryParams); notok {
			return resp
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		if result, err := getFunc(single(accountId), single(id), ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			if objs, ok := result.Payload[accountId]; ok {
				n := len(objs.GetList())
				switch n {
				case 0:
					return req.notFound(accountId, ContactResponseObjectType, result)
				case 1:
					return req.respond(accountId, objs.GetList()[0], ContactResponseObjectType, result)
				default:
					logger.Error().Msgf("found %d %s matching '%s' instead of 1", n, o.responseType, id)
					return req.errorS(accountId, req.apiError(&ErrorMultipleIdMatches), result)
				}
			} else {
				return req.notFound(accountId, ContactResponseObjectType, result)
			}
		}
	})
}

var changesSupportedQueryParams = toSupportedQueryParams(QueryParamMaxChanges)

// Retrieve the changes that occured for {{.Name}}, optionally since an opaque state specified using the header `{{.HeaderParam.HeaderParamSince}}`,
// optionally bounded by the query parameter `{{.QueryParam.QueryParamMaxChanges}}`.
// @api:response 200:CHANGES returns the changes to {{.Names}}: created, updated, and identifiers of destroyed {{.Names}}
func changes[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	changesFunc func(accountId string, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (jmap.Result[CHANGES], error),
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

		if notok, resp := req.unsupportedQueryParams(single(accountId), changesSupportedQueryParams); notok {
			return resp
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		if result, err := changesFunc(accountId, sinceState, maxChanges, ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			return req.respond(accountId, result.Payload, o.responseType, result)
		}
	})
}

// Delete a specific {{.Name}} referenced by its unique identifier as specified in the path parameter `{{.UriParamName}}` in the path `{{.Path}}`
// @api:success 204
// @api:response 204 when the referenced {{.Name}} has been deleted successfully
// @api:response 404 when there is no {{.Name}} for the requested identifier
func delete[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	deleteFunc func(accountId string, ids []string, ctx jmap.Context) (jmap.Result[map[string]jmap.SetError], error),
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

		if notok, resp := req.unsupportedQueryParams(single(accountId), noSupportedQueryParams); notok {
			return resp
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		if result, err := deleteFunc(accountId, single(id), ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			for _, e := range result.Payload {
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
			return req.noContent(accountId, o.responseType, result)
		}
	})
}

var deleteManySupportedQueryParams = toSupportedQueryParams(QueryParamId)

// Delete several {{.Name}} objects referenced by their unique identifiers as specified as an array in the body,
// or using the query parameter `{{.QueryParam.QueryParamId}}`.
// @api:response 204 when the referenced {{.Names}} have all been deleted successfully
// @api:body ?[]string an array of identifiers of {{.Names}} to delete
func deleteMany[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]]( //NOSONAR
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	deleteFunc func(accountId string, ids []string, ctx jmap.Context) (jmap.Result[map[string]jmap.SetError], error),
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

		if notok, resp := req.unsupportedQueryParams(single(accountId), deleteManySupportedQueryParams); notok {
			return resp
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		if result, err := deleteFunc(accountId, ids, ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			for _, e := range result.Payload {
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
			return req.noContent(accountId, o.responseType, result)
		}
	})
}

// Modify the specified {{.Name}} referenced its unique identifier, changes to attributes being specified as a JSON map in the request body.
// @api:response 200:T the modified {{.Name}}
func modify[T jmap.Foo, CHANGE jmap.Change, CHANGES jmap.Changes[T]](
	o ObjectType[T, CHANGE, CHANGES],
	w http.ResponseWriter, r *http.Request,
	g *Groupware,
	updateFunc func(accountId string, id string, change CHANGE, ctx jmap.Context) (jmap.Result[T], error),
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

		if notok, resp := req.unsupportedQueryParams(single(accountId), noSupportedQueryParams); notok {
			return resp
		}

		var change CHANGE
		err = req.body(&change)
		if err != nil {
			return req.error(accountId, err)
		}

		logger := log.From(l)
		ctx := req.ctx.WithLogger(logger)
		if result, err := updateFunc(accountId, id, change, ctx); err != nil {
			switch e := err.(type) {
			case jmap.Error:
				return req.jmapError(accountId, e, result)
			case GroupwareError:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, e))
			default:
				errorId := req.errorId()
				return req.error(accountId, apiError(errorId, ErrorGeneric, withDetail(e.Error())))
			}
		} else {
			return req.respond(accountId, result.Payload, o.responseType, result)
		}
	})
}
