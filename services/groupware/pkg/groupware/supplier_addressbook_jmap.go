package groupware

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type JmapAddressBookSupplier struct {
	client *jmap.Client
}

var _ ListSupplier[jmap.AddressBook, jmap.AddressBookGetResponse] = &JmapAddressBookSupplier{}
var _ ChangesSupplier[jmap.ContactCard, jmap.ContactCardChanges] = &JmapContactCardSupplier{}

func newJmapAddressBookSupplier(client *jmap.Client) *JmapAddressBookSupplier {
	return &JmapAddressBookSupplier{client: client}
}

const jmapAddressBookSupplierId = SupplierId("jmap")

func (c *JmapAddressBookSupplier) GetId() SupplierId {
	return jmapAddressBookSupplierId
}

func (c *JmapAddressBookSupplier) IsMine(id string) bool {
	return id != "" && !strings.Contains(id, ":")
}

func (c *JmapAddressBookSupplier) GetAll(accountId jmap.AccountId, ids []string, ctx jmap.Context) (jmap.Result[jmap.AddressBookGetResponse], error) {
	return c.client.GetAddressbooks(accountId, ids, ctx)
}

func (c *JmapAddressBookSupplier) GetChanges(accountId jmap.AccountId, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (jmap.Result[jmap.AddressBookChanges], error) {
	return c.client.GetAddressbookChanges(accountId, sinceState, maxChanges, ctx)
}
