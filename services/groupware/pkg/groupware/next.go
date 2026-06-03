package groupware

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type NextToken string

const NoNextToken = NextToken("")

func nextSingle(m map[jmap.AccountId]jmap.QueryParams) (NextToken, error) {
	if b, err := json.Marshal(m); err != nil {
		return NoNextToken, err
	} else {
		return NextToken("S" + EncodeBytesToBase62(b)), nil
	}
}

func nextMulti(m map[SupplierId]map[jmap.AccountId]jmap.QueryParams) (NextToken, error) {
	if b, err := json.Marshal(m); err != nil {
		return NoNextToken, err
	} else {
		return NextToken("M" + EncodeBytesToBase62(b)), nil
	}
}

func unnext(n NextToken) (QueryParamsSupplier, error) {
	s := string(n)
	if len(s) < 1 {
		err := fmt.Errorf("invalid next token: is empty")
		return ErrorQueryParamsSupplier{err: err}, err
	}
	t := s[0:1]
	switch t {
	case "S":
		payload := s[1:]
		if b, err := DecodeBytesFromBase62(payload); err != nil {
			return ErrorQueryParamsSupplier{err: err}, err
		} else {
			var m map[jmap.AccountId]jmap.QueryParams
			if err := json.Unmarshal(b, &m); err != nil {
				return ErrorQueryParamsSupplier{err: err}, err
			} else {
				return SingleSupplierQueryParamsSupplier{m: m}, nil
			}
		}
	case "M":
		payload := s[1:]
		if b, err := DecodeBytesFromBase62(payload); err != nil {
			return ErrorQueryParamsSupplier{err: err}, err
		} else {
			var m map[SupplierId]map[jmap.AccountId]jmap.QueryParams
			if err := json.Unmarshal(b, &m); err != nil {
				return ErrorQueryParamsSupplier{err: err}, err
			} else {
				return MultiSupplierQueryParamsSupplier{m: m}, nil
			}
		}
	default:
		err := fmt.Errorf("invalid next token: unsupported type header '%s'", t)
		return ErrorQueryParamsSupplier{err: err}, err
	}
}

func curryNoNextMapQuery[SRES jmap.SearchResults[T], T jmap.Idable, FILTER any, COMP any](
	f func(accountIds map[jmap.AccountId]jmap.QueryParams, limit *uint, filter FILTER, sortBy []COMP, calculateTotal bool, ctx jmap.Context) (jmap.Result[map[jmap.AccountId]SRES], error),
	sorter func(a, b T) int,
	searchResultCtor func(canCalculateChanges jmap.ChangeCalculation, position *uint, limit *uint, total *uint, results []T) SRES,
) func(req Request, accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter FILTER, sortBy []COMP, ctx jmap.Context) (jmap.Result[SRES], NextToken, error) {
	return func(_ Request, accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter FILTER, sortBy []COMP, ctx jmap.Context) (jmap.Result[SRES], NextToken, error) { //NOSONAR
		if m, err := mapQueryParams("", accountIds, qps); err != nil {
			return jmap.ZeroResultV[SRES](), NoNextToken, err
		} else {
			result, err := f(m, limit, filter, sortBy, true, ctx)
			if err != nil {
				return jmap.ZeroResult[SRES](result.Durations), NoNextToken, err
			} else {
				var singleAccountId jmap.AccountId = ""
				// TODO what about requests with zero accountIds, can these even happen at all?
				if len(accountIds) == 1 {
					// optimization: no need to combine the results of several accounts if the query was
					// performed with a single accountId
					singleAccountId = accountIds[0]
				} else {
					// the request includes multiple accounts, but that doesn't mean that we have results from
					// all accounts: let's first calculate the number of results we have for each account
					totalByAccount := structs.MapValues(result.Payload, func(a SRES) int { return len(a.GetResults()) })
					// and let's now pick out the accounts that do have results
					accountsWithResults := structs.FilterKeys(totalByAccount, func(_ jmap.AccountId, total int) bool { return total > 0 })
					if len(accountsWithResults) == 1 {
						singleAccountId = accountsWithResults[0]
					}
					// TODO what if we don't have any results at all? (accountsWithResults == 0)
				}
				if singleAccountId != "" {
					r, err := jmap.RefineResultPayload(result, func(a map[jmap.AccountId]SRES) (SRES, bool, error) {
						r, ok := a[accountIds[0]]
						return r, ok, nil
					})
					return r, NoNextToken, err
				} else {
					// more than one account with results
					return flattenMultipleAccounts(accountIds, qps, limit, result, sorter, searchResultCtor)
				}
			}
		}
	}
}

func flattenMultipleAccounts[SRES jmap.SearchResults[T], T jmap.Idable](
	accountIds []jmap.AccountId,
	qps QueryParamsSupplier,
	limit *uint,
	result jmap.Result[map[jmap.AccountId]SRES],
	sorter func(a, b T) int,
	searchResultCtor func(canCalculateChanges jmap.ChangeCalculation, position *uint, limit *uint, total *uint, results []T) SRES,
) (jmap.Result[SRES], NextToken, error) {
	// 1. we need to flatten all the results, and then sort them in memory
	all := []T{}
	// 2. we need to sort them in memory in the same way as the filter
	slices.SortFunc(all, sorter)

	if limit != nil && *limit > 0 && len(all) > int(*limit) {
		// 3. we need to cut if off to only keep *limit amount of elements
		shrunk := make([]T, *limit)
		copy(shrunk, all)
		all = shrunk

		cc := true
		total := uint(0)
		lastByAccountId := map[jmap.AccountId]string{}
		for accountId, searchResult := range result.Payload {
			if !searchResult.GetCanCalculateChanges() {
				cc = false
			}
			to := searchResult.GetTotal()
			if to != nil {
				total += *to
			}
			for _, item := range searchResult.GetResults() {
				lastByAccountId[accountId] = item.GetId()
			}
			all = append(all, searchResult.GetResults()...)
		}

		// 4. we need to build the NextToken by taking the ID of the last item
		// we kept after shrinking, but separately for each accountId
		n := map[jmap.AccountId]jmap.QueryParams{}
		for accountId, lastId := range lastByAccountId {
			n[accountId] = jmap.QueryParams{Anchor: lastId, AnchorOffset: ptr(1)}
		}
		if err := fillMissingAccounts(qps, "", accountIds, n); err != nil {
			return jmap.ZeroResult[SRES](result.Durations), NoNextToken, err
		}
		if t, err := nextSingle(n); err != nil {
			return jmap.ZeroResult[SRES](result.Durations), NoNextToken, err
		} else {
			if r, err := jmap.RefineResultPayload(result, func(a map[jmap.AccountId]SRES) (SRES, bool, error) {
				return searchResultCtor(jmap.ChangeCalculation(cc), nil, limit, &total, all), true, nil
			}); err != nil {
				return jmap.ZeroResult[SRES](result.Durations), NoNextToken, err
			} else {
				return r, t, nil
			}
		}
	} else {
		cc := true
		total := uint(0)
		for _, searchResult := range result.Payload {
			if !searchResult.GetCanCalculateChanges() {
				cc = false
			}
			to := searchResult.GetTotal()
			if to != nil {
				total += *to
			}
		}

		// no need to compute a NextToken since there was no limit,
		// which means that this Result contains all the elements,
		// and thus there is no "next" to query for
		if r, err := jmap.RefineResultPayload(result, func(a map[jmap.AccountId]SRES) (SRES, bool, error) {
			return searchResultCtor(jmap.ChangeCalculation(cc), nil, limit, &total, all), true, nil
		}); err != nil {
			return jmap.ZeroResult[SRES](result.Durations), NoNextToken, err
		} else {
			return r, NoNextToken, nil
		}
	}
}
