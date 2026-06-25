package groupware

import (
	"slices"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type MockContactCardSupplier struct {
	contacts []jmap.ContactCard
	state    jmap.State
}

var MockContactCardSupplierInstance *MockContactCardSupplier = &MockContactCardSupplier{
	contacts: []jmap.ContactCard{
		{
			Id:             "mock:alan",
			AddressBookIds: map[string]bool{"mock:1": true},
			Type:           jscontact.ContactCardType,
			Version:        jmap.DEFAULT_CONTACT_CARD_VERSION,
			Created:        mustParseTime("2026-05-26T10:21:00.000Z"),
			Kind:           jscontact.ContactCardKindIndividual,
			ProdId:         "OC:mock",
			Uid:            "dc2858d2-4826-412d-afc9-c4492f8f84bc",
			Updated:        mustParseTime("2026-05-26T10:21:00.000Z"),
			Name: &jscontact.Name{
				Type: jscontact.NameType,
				Components: []jscontact.NameComponent{
					{Kind: jscontact.NameComponentKindGiven, Value: "Alan"},
					{Kind: jscontact.NameComponentKindSurname, Value: "Turing"},
				},
				DefaultSeparator: " ",
				IsOrdered:        true,
			},
			Emails: map[string]jscontact.EmailAddress{
				"eedujae1": {
					Address: "alan@example.com",
				},
			},
		},
	},
	state: jmap.State("1"),
}

var _ QuerySupplier[jmap.ContactCard, *jmap.ContactCardSearchResults, jmap.ContactCardFilterElement, jmap.ContactCardComparator] = &MockContactCardSupplier{}
var _ ListSupplier[jmap.ContactCard, jmap.ContactCardGetResponse] = &MockContactCardSupplier{}
var _ ChangesSupplier[jmap.ContactCard, jmap.ContactCardChanges] = &MockContactCardSupplier{}

func newMockContactCardSupplier() *MockContactCardSupplier {
	return MockContactCardSupplierInstance
}

func (c *MockContactCardSupplier) GetId() SupplierId {
	return SupplierId("mock")
}
func (c *MockContactCardSupplier) IsMine(id string) bool {
	return strings.HasPrefix(id, "mock:")
}
func (c *MockContactCardSupplier) GetAll(accountId jmap.AccountId, ids []string, ctx jmap.Context) (jmap.Result[jmap.ContactCardGetResponse], error) {
	contacts := c.contacts
	if len(ids) > 0 {
		contacts = structs.Filter(contacts, func(a jmap.ContactCard) bool { return slices.Contains(ids, a.Id) })
	}
	return jmap.NewResult(
		jmap.ContactCardGetResponse{
			AccountId: accountId,
			State:     "mock",
			List:      contacts,
		},
		jmap.EmptySessionState,
		jmap.State("mock"),
		jmap.NoLanguage,
		nil,
	), nil
}
func (c *MockContactCardSupplier) Query(accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter jmap.ContactCardFilterElement, sortBy []jmap.ContactCardComparator, calculateTotal bool, ctx jmap.Context) (jmap.Result[map[jmap.AccountId]*jmap.ContactCardSearchResults], error) { //NOSONAR
	payload := make(map[jmap.AccountId]*jmap.ContactCardSearchResults, len(accountIds))
	total := len(c.contacts)
	for _, accountId := range accountIds {
		all := []jmap.ContactCard{}
		var qp jmap.QueryParams
		if q, ok, err := qps.ForSupplier(c.GetId(), accountId); err != nil {
			return jmap.ZeroResultV[map[jmap.AccountId]*jmap.ContactCardSearchResults](), err
		} else if ok {
			qp = q
		} else {
			qp = jmap.NullQueryParams
		}

		p := uint(qp.Position)
		if qp.Position < total {
			all = c.contacts[qp.Position:]
		}
		if qp.Anchor != "" {
			a := slices.IndexFunc(all, func(e jmap.ContactCard) bool { return e.Id == qp.Anchor })
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
					all = []jmap.ContactCard{}
				}
			} else {
				all = []jmap.ContactCard{}
			}
		}
		if limit != nil {
			if len(all) > int(*limit) {
				all = all[:int(*limit)]
			}
		}

		res := &jmap.ContactCardSearchResults{
			Results:             all,
			CanCalculateChanges: false,
			Position:            &p,
			Limit:               limit,
		}
		if calculateTotal {
			t := uint(total)
			res.Total = &t
		}
		payload[accountId] = res
	}

	return jmap.NewResult(payload, jmap.EmptySessionState, c.state, jmap.NoLanguage, nil), nil
}

func (c *MockContactCardSupplier) GetChanges(accountId jmap.AccountId, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (jmap.Result[jmap.ContactCardChanges], error) {
	if sinceState == c.state {
		ch := jmap.ContactCardChanges{
			HasMoreChanges: false,
			OldState:       c.state,
			NewState:       c.state,
		}
		return jmap.NewResult(ch, jmap.EmptySessionState, c.state, jmap.NoLanguage, nil), nil
	} else {
		// what happens when the state is unknown?
		ch := jmap.ContactCardChanges{
			HasMoreChanges: false,
			NewState:       c.state,
			Created:        c.contacts,
		}
		return jmap.NewResult(ch, jmap.EmptySessionState, c.state, jmap.NoLanguage, nil), nil
	}
}
