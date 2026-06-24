package groupware

import (
	"slices"
	"strings"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/stretchr/testify/require"
)

type PetSupplier struct {
	id              SupplierId
	petsByAccountId map[jmap.AccountId][]Pet
}

// var _ ListSupplier[Pet, PetGetResponse] = &PetSupplier{}
var _ QuerySupplier[Pet, *PetSearchResults, PetFilterElement, PetComparator] = &PetSupplier{}

func (s *PetSupplier) GetId() SupplierId {
	return s.id
}
func (s *PetSupplier) IsMine(id string) bool {
	return strings.HasPrefix(id, string(s.id)+":")
}
func (s *PetSupplier) Query(accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter PetFilterElement, sortBy []PetComparator, calculateTotal bool, ctx jmap.Context) (jmap.Result[map[jmap.AccountId]*PetSearchResults], error) {
	return inmemquery(s.id, s.petsByAccountId, accountIds, qps, limit, calculateTotal,
		func(results []Pet, canCalculateChanges jmap.ChangeCalculation, position *uint, limit *uint, total *uint) *PetSearchResults {
			return &PetSearchResults{Results: results, CanCalculateChanges: canCalculateChanges, Position: position, Limit: limit, Total: total}
		},
	)
}

func inmemquery[T jmap.Idable, R jmap.SearchResults[T]](
	supplierId SupplierId,
	store map[jmap.AccountId][]T,
	accountIds []jmap.AccountId,
	qps QueryParamsSupplier, limit *uint,
	calculateTotal bool,
	searchResultCtor func(results []T, canCalculateChanges jmap.ChangeCalculation, position *uint, limit *uint, total *uint) R,
) (jmap.Result[map[jmap.AccountId]R], error) {
	payload := make(map[jmap.AccountId]R, len(accountIds))
	for _, accountId := range accountIds {
		qp := jmap.NullQueryParams
		if q, ok, err := qps.ForSupplier(supplierId, accountId); err != nil {
			return jmap.ZeroResultV[map[jmap.AccountId]R](), err
		} else if ok {
			qp = q
		}

		items := store[accountId]
		if items == nil {
			items = []T{}
		}
		total := len(items)
		all := []T{}
		p := uint(qp.Position)
		if qp.Position < total {
			all = items[qp.Position:]
		}
		if qp.Anchor != "" {
			a := slices.IndexFunc(all, func(e T) bool { return e.GetId() == qp.Anchor })
			if a >= 0 {
				if qp.AnchorOffset != nil {
					a += int(*qp.AnchorOffset)
				} else {
					a += 1
				}
				p += uint(a)
				if a < len(all) {
					all = all[a:]
				} else {
					all = []T{}
				}
			} else {
				all = []T{}
			}
		}
		if limit != nil {
			if len(all) > int(*limit) {
				all = all[:int(*limit)]
			}
		}

		var t *uint = nil
		if calculateTotal {
			ut := uint(total)
			t = &ut
		}
		res := searchResultCtor(all, false, &p, limit, t)

		payload[accountId] = res
	}
	return jmap.NewResult(payload, jmap.EmptySessionState, jmap.EmptyState, jmap.NoLanguage, nil), nil
}

func pets(
	suppliers []QuerySupplier[Pet, *PetSearchResults, PetFilterElement, PetComparator],
	accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint,
	filter PetFilterElement, sortBy []PetComparator,
	calculateTotal bool,
	ctx jmap.Context) (jmap.Result[*PetSearchResults], NextToken, error) {
	return squery(suppliers, accountIds, qps, limit, filter, sortBy, calculateTotal, ctx,
		func(supplier QuerySupplier[Pet, *PetSearchResults, PetFilterElement, PetComparator], filter PetFilterElement) bool {
			return true
		},
		func(a, b Pet) int { return strings.Compare(a.name, b.name) },
		func(canCalculateChanges jmap.ChangeCalculation, position, limit, total *uint, results []Pet) *PetSearchResults {
			return &PetSearchResults{
				Results:             results,
				CanCalculateChanges: canCalculateChanges,
				Position:            position,
				Limit:               limit,
				Total:               total,
			}
		},
	)
}

func TestSquery(t *testing.T) {
	require := require.New(t)

	suppliers := []QuerySupplier[Pet, *PetSearchResults, PetFilterElement, PetComparator]{
		&PetSupplier{
			id: "X",
			petsByAccountId: map[jmap.AccountId][]Pet{
				"a": {
					{id: "X:1", name: "ace"},
					{id: "X:2", name: "bella"},
				},
				"b": {
					{id: "X:3", name: "mitzi"},
				},
			},
		},
		&PetSupplier{
			id: "Y",
			petsByAccountId: map[jmap.AccountId][]Pet{
				"a": {
					{id: "Y:1", name: "cupcake"},
					{id: "Y:2", name: "elvis"},
				},
				"c": {
					{id: "Y:3", name: "fluffy"},
					{id: "Y:4", name: "rambo"},
				},
			},
		},
	}
	f := func(accountIds []jmap.AccountId, position int, anchor string, anchorOffset *int, limit *uint) (jmap.Result[*PetSearchResults], NextToken, error) {
		return pets(suppliers, accountIds,
			StaticQueryParamsSupplier{qp: jmap.QueryParams{Position: position, Anchor: anchor, AnchorOffset: anchorOffset}}, limit,
			PetFilterCondition{}, []PetComparator{{Property: "id", IsAscending: true}},
			true, jmap.Context{},
		)
	}
	n := func(accountIds []jmap.AccountId, nextToken NextToken, limit *uint) (jmap.Result[*PetSearchResults], NextToken, error) {
		if qps, err := unnext(nextToken); err != nil {
			return jmap.ZeroResultV[*PetSearchResults](), NoNextToken, err
		} else {
			return pets(suppliers, accountIds, qps, limit, PetFilterCondition{}, []PetComparator{{Property: "id", IsAscending: true}}, true, jmap.Context{})
		}
	}

	{
		res, n, err := f([]jmap.AccountId{"a", "b", "c"}, 0, "", nil, nil)
		require.NoError(err)
		require.Len(res.Payload.Results, 7)
		require.Equal(uint(len(res.Payload.Results)), *res.Payload.Total)
		{
			require.Equal("ace", res.Payload.Results[0].name)
			require.Equal("bella", res.Payload.Results[1].name)
			require.Equal("cupcake", res.Payload.Results[2].name)
			require.Equal("elvis", res.Payload.Results[3].name)
			require.Equal("fluffy", res.Payload.Results[4].name)
			require.Equal("mitzi", res.Payload.Results[5].name)
			require.Equal("rambo", res.Payload.Results[6].name)
		}
		require.Equal(jmap.IncapableOfChangeCalculation, res.Payload.CanCalculateChanges)
		require.Nil(res.Payload.Limit)
		require.Nil(res.Payload.Position)
		{
			m, err := unnext(n)
			require.NoError(err)
			require.NotEmpty(m)
		}
	}
	var nextToken NextToken
	{
		res, n, err := f([]jmap.AccountId{"a", "b", "c"}, 0, "", nil, uintPtr(4))
		nextToken = n
		require.NoError(err)
		require.Len(res.Payload.Results, 4)
		require.Equal(uint(7), *res.Payload.Total)
		{
			require.Equal("ace", res.Payload.Results[0].name)
			require.Equal("bella", res.Payload.Results[1].name)
			require.Equal("cupcake", res.Payload.Results[2].name)
			require.Equal("elvis", res.Payload.Results[3].name)
		}
		require.Equal(jmap.IncapableOfChangeCalculation, res.Payload.CanCalculateChanges)
		require.Equal(uint(4), *res.Payload.Limit)
		require.Nil(res.Payload.Position)
		{
			m, err := unnext(n)
			require.NoError(err)
			require.NotEmpty(m)
		}
	}
	{
		res, _, err := n([]jmap.AccountId{"a", "b", "c"}, nextToken, uintPtr(4))
		require.NoError(err)
		require.Equal(uint(7), *res.Payload.Total)
		require.Len(res.Payload.Results, 3)
		{
			require.Equal("fluffy", res.Payload.Results[0].name)
			require.Equal("mitzi", res.Payload.Results[1].name)
			require.Equal("rambo", res.Payload.Results[2].name)
		}
	}
}
