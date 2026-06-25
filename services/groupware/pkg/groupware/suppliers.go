package groupware

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type SupplierId string

const EmptySupplierId = ""

const DefaultSupplierId = SupplierId("jmap")

type Supplier[T jmap.Foo] interface {
	GetId() SupplierId
	IsMine(id string) bool
}

type ListSupplier[T jmap.Foo, G jmap.GetResponse[T]] interface {
	GetAll(accountId jmap.AccountId, ids []string, ctx jmap.Context) (jmap.Result[G], error)
	Supplier[T]
}

type QuerySupplier[T jmap.Foo, R jmap.SearchResults[T], F jmap.FilterElement[T], C jmap.Comparator[T]] interface {
	Query(accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter F, sortBy []C, calculateTotal bool, ctx jmap.Context) (jmap.Result[map[jmap.AccountId]R], error)
	Supplier[T]
}

type ChangesSupplier[T jmap.Foo, C jmap.Changes[T]] interface {
	GetChanges(accountId jmap.AccountId, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (jmap.Result[C], error)
	Supplier[T]
}

// queryFunc func(req Request, accountIds []string, qps QueryParamsSupplier, limit *uint, filter FILTER, sortBy []COMP, ctx jmap.Context) (jmap.Result[SEARCHRESULTS], NextToken, error),
func curryQueryFunc[SRES jmap.SearchResults[T], T jmap.Foo, FILTER any, COMP any](
	f func(accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter FILTER, sortBy []COMP, calculateTotal bool, ctx jmap.Context) (jmap.Result[SRES], NextToken, error),
) func(req Request, accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter FILTER, sortBy []COMP, ctx jmap.Context) (jmap.Result[SRES], NextToken, error) {
	return func(_ Request, accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter FILTER, sortBy []COMP, ctx jmap.Context) (jmap.Result[SRES], NextToken, error) { //NOSONAR
		result, next, err := f(accountIds, qps, limit, filter, sortBy, true, ctx)
		if err != nil {
			return jmap.ZeroResult[SRES](result.Durations), next, err
		} else {
			return result, next, err
		}
	}
}

func agg[T jmap.Idable, R jmap.GetResponse[T]](accountId jmap.AccountId, supplierIds []SupplierId, responses []*R, //NOSONAR
	ctor func(accountId jmap.AccountId, state jmap.State, notFound []string, list []T) R) (R, error) {
	if len(responses) < 1 {
		var zero R
		return zero, fmt.Errorf("requires at least one response")
	}
	lists := structs.Concat(structs.Map(responses, func(e *R) []T {
		if e != nil {
			return (*e).GetList()
		} else {
			var zero []T
			return zero
		}
	})...)
	states, err := structs.MeshMap(supplierIds, responses, func(id SupplierId, e *R) (SupplierId, jmap.State, bool) {
		if e != nil {
			state := (*e).GetState()
			if state != jmap.EmptyState {
				return id, state, true
			} else {
				return id, jmap.EmptyState, false
			}
		} else {
			return "", jmap.EmptyState, false
		}
	})
	if err != nil {
		var zero R
		return zero, err
	}
	state, err := combineState(states)
	if err != nil {
		var zero R
		return zero, err
	}
	notFounds := structs.Concat(structs.Map(responses, func(e *R) []string {
		if e != nil {
			return (*e).GetNotFound()
		} else {
			return []string{}
		}
	})...)
	return ctor(accountId, state, notFounds, lists), nil
}

func aggChanges[T jmap.Idable, R jmap.Changes[T]](accountId jmap.AccountId, supplierIds []SupplierId, responses []*R, //NOSONAR
	ctor func(accountId jmap.AccountId, oldState jmap.State, newState jmap.State, created []T, updated []T, destroyed []string, hasMoreChanges bool) R,
) (R, error) {
	if len(responses) < 1 {
		var zero R
		return zero, fmt.Errorf("requires at least one response")
	}
	created := structs.Concat(structs.Map(responses, func(e *R) []T {
		if e != nil {
			return (*e).GetCreated()
		} else {
			return []T{}
		}
	})...)
	updated := structs.Concat(structs.Map(responses, func(e *R) []T {
		if e != nil {
			return (*e).GetUpdated()
		} else {
			return []T{}
		}
	})...)
	destroyed := structs.Concat(structs.Map(responses, func(e *R) []string {
		if e != nil {
			return (*e).GetDestroyed()
		} else {
			return []string{}
		}
	})...)
	oldStates, err := structs.MeshMap(supplierIds, responses, func(id SupplierId, e *R) (SupplierId, jmap.State, bool) {
		if e != nil {
			state := (*e).GetOldState()
			if state != jmap.EmptyState {
				return id, state, true
			} else {
				return id, jmap.EmptyState, false
			}
		} else {
			return "", jmap.EmptyState, false
		}
	})
	if err != nil {
		var zero R
		return zero, err
	}
	oldState, err := combineState(oldStates)
	if err != nil {
		var zero R
		return zero, err
	}
	newStates, err := structs.MeshMap(supplierIds, responses, func(id SupplierId, e *R) (SupplierId, jmap.State, bool) {
		if e != nil {
			state := (*e).GetNewState()
			if state != jmap.EmptyState {
				return id, state, true
			} else {
				return id, jmap.EmptyState, false
			}
		} else {
			return "", jmap.EmptyState, false
		}
	})
	if err != nil {
		var zero R
		return zero, err
	}
	newState, err := combineState(newStates)
	if err != nil {
		var zero R
		return zero, err
	}
	hasMoreChanges := structs.AllMatch(structs.Map(responses, func(e *R) bool {
		if e != nil {
			return (*e).GetHasMoreChanges()
		} else {
			return false
		}
	}), func(b bool) bool { return b })
	return ctor(accountId, oldState, newState, created, updated, destroyed, hasMoreChanges), nil
}

func slist[T jmap.Idable, G jmap.GetResponse[T], S ListSupplier[T, G]](suppliers []S, accountId jmap.AccountId, ids []string, ctx jmap.Context, //NOSONAR
	ctor func(accountId jmap.AccountId, state jmap.State, notFound []string, list []T) G) (jmap.Result[G], error) {
	switch len(suppliers) {
	case 0:
		return jmap.ZeroResultV[G](), nil
	case 1:
		return suppliers[0].GetAll(accountId, ids, ctx)
	default:
		results := make([]*jmap.Result[G], len(suppliers))
		supplierIds := make([]SupplierId, len(suppliers))
		for i, supplier := range suppliers {
			supplierIds[i] = supplier.GetId()
			localIds := []string{}
			if len(ids) > 0 {
				localIds = structs.Filter(ids, func(id string) bool { return supplier.IsMine(id) })
				if len(localIds) == 0 {
					results[i] = nil
					continue
				}
			}
			// we are not injecting id prefixes here for all the objects, as each supplier is responsible for doing that if necessary
			if result, err := supplier.GetAll(accountId, localIds, ctx); err != nil {
				return result, err
			} else {
				results[i] = &result
			}
		}

		return smeld[T](supplierIds, results, func(payloads []*G) (G, error) {
			return agg(accountId, supplierIds, payloads, ctor)
		}, func(resp G) jmap.State {
			return resp.GetState()
		})

		/*

			return jmap.RefineResultSlice(results, func(payloads []*G, sessionStates []*jmap.SessionState, states []*jmap.State, langs []*jmap.Language) (G, jmap.SessionState, jmap.State, jmap.Language, error) {
				resp, err := agg(accountId, supplierIds, payloads, ctor)
				if err != nil {
					return resp, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, err
				}
				m, err := structs.MeshMap(supplierIds, sessionStates, func(id SupplierId, state *jmap.SessionState) (SupplierId, jmap.SessionState, bool) {
					if state != nil && *state != jmap.EmptySessionState {
						return id, *state, true
					} else {
						return id, jmap.EmptySessionState, false
					}
				})
				if err != nil {
					return resp, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, err
				}
				sessionState, err := combineState(m)
				if err != nil {
					return resp, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, err
				}
				lang := jmap.NoLanguage
				if f, ok := structs.First(langs, func(l *jmap.Language) bool { return l != nil && *l != jmap.NoLanguage }); ok {
					lang = *f
				}
				return resp, sessionState, resp.GetState(), lang, nil
			})
		*/
	}
}

func schanges[T jmap.Idable, C jmap.Changes[T], S ChangesSupplier[T, C]](suppliers []S, accountId jmap.AccountId, sinceState jmap.State, maxChanges uint, ctx jmap.Context, //NOSONAR
	ctor func(accountId jmap.AccountId, oldState jmap.State, newState jmap.State, created []T, updated []T, destroyed []string, hasMoreChanges bool) C,
) (jmap.Result[C], error) {
	switch len(suppliers) {
	case 0:
		return jmap.ZeroResultV[C](), nil
	case 1:
		return suppliers[0].GetChanges(accountId, sinceState, maxChanges, ctx)
	default:
		results := make([]*jmap.Result[C], len(suppliers))
		supplierIds := make([]SupplierId, len(suppliers))
		stateBySupplierId, err := splitState(sinceState, DefaultSupplierId)
		if err != nil {
			return jmap.ZeroResultV[C](), err
		}
		for i, supplier := range suppliers {
			supplierIds[i] = supplier.GetId()
			state := jmap.EmptyState
			if v, ok := stateBySupplierId[supplier.GetId()]; ok {
				state = v
			}
			// we are not injecting id prefixes here for all the objects, as each supplier is responsible for doing that if necessary
			if result, err := supplier.GetChanges(accountId, state, maxChanges, ctx); err != nil {
				return result, err
			} else {
				results[i] = &result
			}
		}

		// TODO need to reduce the maxChanges? this is quite complex to implement, if even possible, not doing that for now

		return smeld[T](supplierIds, results, func(payloads []*C) (C, error) {
			return aggChanges(accountId, supplierIds, payloads, ctor)
		}, func(resp C) jmap.State {
			return resp.GetNewState()
		})

		/*
			return jmap.RefineResultSlice(results, func(payloads []*C, sessionStates []*jmap.SessionState, states []*jmap.State, langs []*jmap.Language) (C, jmap.SessionState, jmap.State, jmap.Language, error) {
				resp, err := aggChanges(accountId, supplierIds, payloads, ctor)
				if err != nil {
					return resp, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, err
				}
				m, err := structs.MeshMap(supplierIds, sessionStates, func(id SupplierId, state *jmap.SessionState) (SupplierId, jmap.SessionState, bool) {
					if state != nil && *state != jmap.EmptySessionState {
						return id, *state, true
					} else {
						return id, jmap.EmptySessionState, false
					}
				})
				if err != nil {
					return resp, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, err
				}
				sessionState, err := combineState(m)
				if err != nil {
					return resp, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, err
				}
				lang := jmap.NoLanguage
				if f, ok := structs.First(langs, func(l *jmap.Language) bool { return l != nil && *l != jmap.NoLanguage }); ok {
					lang = *f
				}
				return resp, sessionState, resp.GetNewState(), lang, nil
			})
		*/
	}
}

func smeld[T jmap.Idable, C any](
	supplierIds []SupplierId,
	results []*jmap.Result[C],
	aggregator func(payloads []*C) (C, error),
	stateSupplier func(resp C) jmap.State,
) (jmap.Result[C], error) {
	return jmap.RefineResultSlice(results, func(payloads []*C, sessionStates []*jmap.SessionState, states []*jmap.State, langs []*jmap.Language) (C, jmap.SessionState, jmap.State, jmap.Language, error) {
		resp, err := aggregator(payloads)
		if err != nil {
			return resp, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, err
		}
		m, err := structs.MeshMap(supplierIds, sessionStates, func(id SupplierId, state *jmap.SessionState) (SupplierId, jmap.SessionState, bool) {
			if state != nil && *state != jmap.EmptySessionState {
				return id, *state, true
			} else {
				return id, jmap.EmptySessionState, false
			}
		})
		if err != nil {
			return resp, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, err
		}
		sessionState, err := combineState(m)
		if err != nil {
			return resp, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, err
		}
		lang := jmap.NoLanguage
		if f, ok := structs.First(langs, func(l *jmap.Language) bool { return l != nil && *l != jmap.NoLanguage }); ok {
			lang = *f
		}
		return resp, sessionState, stateSupplier(resp), lang, nil
	})
}

func fillMissingAccounts(qps QueryParamsSupplier, supplierId SupplierId, accountIds []jmap.AccountId, n map[jmap.AccountId]jmap.QueryParams) error {
	for _, accountId := range accountIds {
		if _, ok := n[accountId]; !ok {
			// no result item was kept for this accountId
			// => the next page must be either the same as what was requested for this one
			// or, if not specified, it should be the first page, which is assumed when there
			// is no entry in the "next" map for that accountId, so no need to store anything
			// in the "next" map in that case
			if qp, ok, err := qps.ForSupplier(supplierId, accountId); err != nil {
				// unlikely to happen here, since that should already have been requested
				// by the query function, but let's handle it gracefully here as well
				return err
			} else if ok {
				// just use the same query parameters for that accountId the next time
				n[accountId] = qp
			} else {
				// we didn't find any query parameters for this supplier and accountId,
				// which means that the first page is requested, and thus we don't need
				// to store anything in the "next" map either
			}
		}
	}
	return nil
}

func squery[T jmap.Idable, R jmap.SearchResults[T], S QuerySupplier[T, R, F, C], F jmap.FilterElement[T], C jmap.Comparator[T]]( //NOSONAR
	suppliers []S, accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter F, sortBy []C,
	calculateTotal bool, ctx jmap.Context,
	filterSupplierPredicate func(supplier S, filter F) bool,
	sorter func(T, T) int,
	searchResultCtor func(canCalculateChanges jmap.ChangeCalculation, position *uint, limit *uint, total *uint, results []T) R) (
	jmap.Result[R], NextToken, error,
) {
	s := len(suppliers)
	if s == 0 {
		return jmap.ZeroResultV[R](), NoNextToken, nil
	}

	switch s {
	case 1:
		supplier := suppliers[0]
		if result, err := supplier.Query(accountIds, qps, limit, filter, sortBy, calculateTotal, ctx); err != nil {
			return jmap.ZeroResult[R](result.Durations), NoNextToken, err
		} else if len(accountIds) == 0 {
			return jmap.ZeroResult[R](result.Durations), NoNextToken, nil
		} else if len(accountIds) == 1 {
			accountId := accountIds[0]
			// use anchor and anchorOffset and limit:
			// the anchor for the next query is the ID of the last element in the results for this query
			// using an anchor offset of +1
			n := map[jmap.AccountId]jmap.QueryParams{}
			for accountId, payload := range result.Payload {
				items := payload.GetResults()
				last := items[len(items)-1]
				n[accountId] = jmap.QueryParams{Anchor: last.GetId(), AnchorOffset: ptr(1)}
			}
			if nextToken, err := nextSingle(n); err != nil {
				return jmap.ZeroResult[R](result.Durations), NoNextToken, err
			} else {
				single, err := jmap.RefineResultPayload(result, func(m map[jmap.AccountId]R) (R, bool, error) {
					if r, ok := m[accountId]; ok {
						return r, true, nil
					} else {
						return r, false, nil
					}
				})
				return single, nextToken, err
			}
		} else {
			// multiple accountIds => we need to merge/flatten the results, and we must use anchor and offset for the next page
			payloads := []T{}
			originals := map[jmap.AccountId][]T{}
			cc := true
			total := uint(0)
			for accountId, searchResult := range result.Payload {
				if !searchResult.GetCanCalculateChanges() {
					cc = false
				}
				to := searchResult.GetTotal()
				if to != nil {
					total += *to
				}
				originals[accountId] = searchResult.GetResults()
				payloads = append(payloads, searchResult.GetResults()...)
			}

			// we have to sort everything in memory in order to be able to cut off if the limit is exceeded,
			// but only if necessary
			var r R
			{
				l := len(payloads)
				if limit != nil && l > int(*limit) {
					// the amount of items in payload should not exceed the requested limit
					// but in order to determine which items to keep and which to discard, we need to sort
					// them in memory first, to have a stable order from which to cut off and drop the
					// intermingled search results that are beyond the limit
					// e.g. if, with a limit of 10, supplier A gives us 10 results and supplier B gives us 5,
					// we end up with 15 elements of which we may only return the first 10, but in order to
					// reliably and repeatedly determine which those first 10 are, we need to sort them in
					// memory first
					slices.SortFunc(payloads, sorter)
					var shrunk []T
					shrunk = make([]T, int(*limit))
					copy(shrunk, payloads)
					payloads = shrunk
				}
				r = searchResultCtor(jmap.ChangeCalculation(cc), nil, limit, &total, payloads) // TODO can we determine the position here, instead of nil?
			}

			lastIdByAccountId := map[jmap.AccountId]string{}
			// not amazing, but since the accountId information is not attached to every single
			// search result element (e.g. a ContactCard), we have to iterate over the original results that we
			// have by accountId to find them again, in order to determine the "last ID"
			// for each accountId, since we will have to persist that into the "next page"
			// token in order to perform the query for each supplier and accountId
			for _, item := range payloads {
				for accountId, searchResult := range result.Payload {
					if slices.IndexFunc(searchResult.GetResults(), func(t T) bool { return t.GetId() == item.GetId() }) >= 0 {
						lastIdByAccountId[accountId] = item.GetId()
					}
				}
			}

			n := map[jmap.AccountId]jmap.QueryParams{}
			for accountId, lastId := range lastIdByAccountId {
				// the anchor for the next query is the ID of the last element in the results for this query
				// using an anchor offset of +1
				n[accountId] = jmap.QueryParams{Anchor: lastId, AnchorOffset: ptr(1)}
			}
			if err := fillMissingAccounts(qps, supplier.GetId(), accountIds, n); err != nil {
				return jmap.ZeroResult[R](result.Durations), NoNextToken, err
			}

			nextBySupplier := map[SupplierId]map[jmap.AccountId]jmap.QueryParams{supplier.GetId(): n}
			if nextToken, err := nextMulti(nextBySupplier); err != nil {
				return jmap.ZeroResult[R](result.Durations), NoNextToken, err
			} else {
				if refined, err := jmap.RefineResultPayload(result, func(a map[jmap.AccountId]R) (R, bool, error) { return r, true, nil }); err != nil {
					return jmap.ZeroResult[R](result.Durations), NoNextToken, err
				} else {
					return refined, nextToken, nil
				}
			}
		}
	default:
		payloads := []T{}
		originals := map[SupplierId]map[jmap.AccountId][]T{}
		cc := true
		total := uint(0)
		sessionState := jmap.EmptySessionState
		states := map[SupplierId]jmap.State{}
		lang := jmap.NoLanguage
		durations := make([][]time.Duration, len(suppliers))
		for i, supplier := range suppliers {
			if !filterSupplierPredicate(supplier, filter) {
				// the filter is not applicable for this supplier => skip
				continue
			}

			// we are not injecting id prefixes here for all the objects, as each supplier is responsible for doing that if necessary
			if result, err := supplier.Query(accountIds, qps, limit, filter, sortBy, calculateTotal, ctx); err != nil {
				return jmap.ZeroResult[R](result.Durations), NoNextToken, err
			} else {
				durations[i] = result.Durations
				if result.GetSessionState() != jmap.EmptySessionState {
					sessionState = result.GetSessionState()
				}
				states[supplier.GetId()] = result.GetState()
				if result.GetLanguage() != jmap.NoLanguage {
					lang = result.GetLanguage()
				}
				// iterate over results by accountId and flatten everything into the 'payloads' array
				o := map[jmap.AccountId][]T{}
				for accountId, searchResult := range result.Payload {
					o[accountId] = searchResult.GetResults()
					if !searchResult.GetCanCalculateChanges() {
						cc = false
					}
					to := searchResult.GetTotal()
					if to != nil {
						total += *to
					}
					payloads = append(payloads, searchResult.GetResults()...)
				}
				originals[supplier.GetId()] = o
			}
		}

		// we have to sort everything in memory in order to be able to cut off if the limit is exceeded,
		// but only if necessary
		var r R
		{
			l := len(payloads)
			slices.SortFunc(payloads, sorter)
			if limit != nil && l > int(*limit) {
				// the amount of items in payload should not exceed the requested limit
				// but in order to determine which items to keep and which to discard, we need to sort
				// them in memory first, to have a stable order from which to cut off and drop the
				// intermingled search results that are beyond the limit
				// e.g. if, with a limit of 10, supplier A gives us 10 results and supplier B gives us 5,
				// we end up with 15 elements of which we may only return the first 10, but in order to
				// reliably and repeatedly determine which those first 10 are, we need to sort them in
				// memory first
				var shrunk []T
				shrunk = make([]T, int(*limit))
				copy(shrunk, payloads)
				payloads = shrunk
			}
			r = searchResultCtor(jmap.ChangeCalculation(cc), nil, limit, &total, payloads) // TODO cen we provide the position here instead of nil?
		}

		lastIdBySupplierByAccountId := map[SupplierId]map[jmap.AccountId]string{}
		// not amazing, but since the accountId and supplier information is not attached to every single
		// search result element (e.g. a ContactCard), we have to iterate over the original results that we
		// kept by supplier and by accountId to find them again, in order to determine the "last ID"
		// for each supplier and for each accountId, since we will have to persist that into the "next page"
		// token in order to perform the query for each supplier and accountId
		for _, item := range payloads {
			for supplierId, o := range originals {
				for accountId, items := range o {
					if slices.IndexFunc(items, func(t T) bool { return t.GetId() == item.GetId() }) >= 0 {
						if _, ok := lastIdBySupplierByAccountId[supplierId]; !ok {
							lastIdBySupplierByAccountId[supplierId] = map[jmap.AccountId]string{}
						}
						lastIdBySupplierByAccountId[supplierId][accountId] = item.GetId()
					}
				}
			}
		}

		nextBySupplier := map[SupplierId]map[jmap.AccountId]jmap.QueryParams{}
		for supplierId, m := range lastIdBySupplierByAccountId {
			n := map[jmap.AccountId]jmap.QueryParams{}
			for accountId, lastId := range m {
				// the anchor for the next query is the ID of the last element in the results for this query
				// using an anchor offset of +1
				n[accountId] = jmap.QueryParams{Anchor: lastId, AnchorOffset: ptr(1)}
			}
			if err := fillMissingAccounts(qps, supplierId, accountIds, n); err != nil {
				return jmap.ZeroResult[R](structs.Flatten(durations)), NoNextToken, err
			}
			nextBySupplier[supplierId] = n
		}

		if nextToken, err := nextMulti(nextBySupplier); err != nil {
			return jmap.ZeroResult[R](structs.Flatten(durations)), NoNextToken, err
		} else {
			if state, err := combineState(states); err != nil {
				return jmap.ZeroResult[R](structs.Flatten(durations)), NoNextToken, err
			} else {
				return jmap.NewResult(r, sessionState, state, lang, structs.Flatten(durations)), nextToken, nil
			}
		}
	}
}

const combinedStateEncodingPrefix = "="

func combineState[S jmap.State | jmap.SessionState](m map[SupplierId]S) (S, error) {
	if b, err := json.Marshal(m); err != nil {
		return "", err
	} else {
		s := combinedStateEncodingPrefix + base64.RawURLEncoding.EncodeToString(b)
		return S(s), nil
	}
}

func splitState[S jmap.State | jmap.SessionState](state S, defaultSupplierId SupplierId) (map[SupplierId]S, error) {
	s := string(state)
	if strings.HasPrefix(s, combinedStateEncodingPrefix) {
		if b, err := base64.RawURLEncoding.DecodeString(s[len(combinedStateEncodingPrefix):]); err != nil {
			return nil, err
		} else {
			m := map[SupplierId]S{}
			if err := json.Unmarshal(b, &m); err != nil {
				return nil, err
			} else {
				return m, nil
			}
		}
	} else {
		return map[SupplierId]S{defaultSupplierId: state}, nil
	}
}
