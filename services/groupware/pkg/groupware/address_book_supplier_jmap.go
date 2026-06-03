package groupware

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type JmapContactCardSupplier struct {
	client *jmap.Client
}

var _ ListSupplier[jmap.AddressBook, jmap.AddressBookGetResponse] = &JmapContactCardSupplier{}
var _ QuerySupplier[jmap.ContactCard, *jmap.ContactCardSearchResults, jmap.ContactCardFilterElement, jmap.ContactCardComparator] = &JmapContactCardSupplier{}

func newJmapContactCardSupplier(client *jmap.Client) *JmapContactCardSupplier {
	return &JmapContactCardSupplier{client: client}
}

func (c *JmapContactCardSupplier) GetId() string {
	return "jmap"
}
func (c *JmapContactCardSupplier) IsMine(id string) bool {
	return id != "" && !strings.Contains(id, ":")
}
func (c *JmapContactCardSupplier) GetAll(accountId string, ids []string, ctx jmap.Context) (jmap.Result[jmap.AddressBookGetResponse], error) {
	return c.client.GetAddressbooks(accountId, ids, ctx)
}
func (c *JmapContactCardSupplier) Query(accountIds []string, qps QueryParamsSupplier, limit *uint, filter jmap.ContactCardFilterElement, sortBy []jmap.ContactCardComparator, calculateTotal bool, ctx jmap.Context) (jmap.Result[map[string]*jmap.ContactCardSearchResults], error) { //NOSONAR
	if m, err := mapQueryParams(c.GetId(), accountIds, qps); err != nil {
		return jmap.ZeroResult[map[string]*jmap.ContactCardSearchResults](), err
	} else {
		return c.client.QueryContactCards(m, limit, filter, sortBy, calculateTotal, ctx)
	}
}
