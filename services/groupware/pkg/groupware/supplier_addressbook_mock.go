package groupware

import (
	"slices"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type MockAddressBookSupplier struct {
	addressBooks []jmap.AddressBook
	state        jmap.State
}

var MockAddressBookSupplierInstance *MockAddressBookSupplier = &MockAddressBookSupplier{
	addressBooks: []jmap.AddressBook{{
		Id:           "mock:1",
		Name:         "Automatic Addressbook",
		Description:  "Users",
		IsDefault:    false,
		IsSubscribed: true,
		MyRights: jmap.AddressBookRights{
			MayRead:   true,
			MayWrite:  false,
			MayAdmin:  false,
			MayDelete: false,
		},
	}},
	state: jmap.State("1"),
}

var _ ListSupplier[jmap.AddressBook, jmap.AddressBookGetResponse] = &MockAddressBookSupplier{}

func newMockAddressBookSupplier() *MockAddressBookSupplier {
	return MockAddressBookSupplierInstance
}

func (c *MockAddressBookSupplier) GetId() SupplierId {
	return SupplierId("mock")
}
func (c *MockAddressBookSupplier) IsMine(id string) bool {
	return strings.HasPrefix(id, "mock:")
}
func (c *MockAddressBookSupplier) GetAll(accountId jmap.AccountId, ids []string, ctx jmap.Context) (jmap.Result[jmap.AddressBookGetResponse], error) {
	abooks := c.addressBooks
	if len(ids) > 0 {
		abooks = structs.Filter(abooks, func(a jmap.AddressBook) bool { return slices.Contains(ids, a.Id) })
	}
	return jmap.NewResult(
		jmap.AddressBookGetResponse{
			AccountId: accountId,
			State:     "mock",
			List:      abooks,
		},
		jmap.EmptySessionState,
		jmap.State("mock"),
		jmap.NoLanguage,
		nil,
	), nil
}

func (c *MockAddressBookSupplier) GetChanges(accountId jmap.AccountId, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (jmap.Result[jmap.AddressBookChanges], error) {
	if sinceState == c.state {
		ch := jmap.AddressBookChanges{
			HasMoreChanges: false,
			OldState:       c.state,
			NewState:       c.state,
		}
		return jmap.NewResult(ch, jmap.EmptySessionState, c.state, jmap.NoLanguage, nil), nil
	} else {
		// what happens when the state is unknown?
		ch := jmap.AddressBookChanges{
			HasMoreChanges: false,
			NewState:       c.state,
			Created:        c.addressBooks,
		}
		return jmap.NewResult(ch, jmap.EmptySessionState, c.state, jmap.NoLanguage, nil), nil
	}
}
