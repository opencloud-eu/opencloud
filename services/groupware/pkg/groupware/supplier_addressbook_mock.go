package groupware

import (
	"slices"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type MockAddressBookSupplier struct {
	addressBook jmap.AddressBook
}

var MockAddressBookSupplierInstance *MockAddressBookSupplier = &MockAddressBookSupplier{
	addressBook: jmap.AddressBook{
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
	},
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
	abooks := []jmap.AddressBook{c.addressBook}
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
