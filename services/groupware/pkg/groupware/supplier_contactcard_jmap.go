package groupware

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type JmapContactCardSupplier struct {
	client *jmap.Client
}

var _ QuerySupplier[jmap.ContactCard, *jmap.ContactCardSearchResults, jmap.ContactCardFilterElement, jmap.ContactCardComparator] = &JmapContactCardSupplier{}
var _ ListSupplier[jmap.ContactCard, jmap.ContactCardGetResponse] = &JmapContactCardSupplier{}
var _ ChangesSupplier[jmap.ContactCard, jmap.ContactCardChanges] = &JmapContactCardSupplier{}
var _ CreateSupplier[jmap.ContactCard, jmap.ContactCardChange] = &JmapContactCardSupplier{}

func newJmapContactCardSupplier(client *jmap.Client) *JmapContactCardSupplier {
	return &JmapContactCardSupplier{client: client}
}

const jmapContactCardSupplierId = SupplierId("jmap")

func (c *JmapContactCardSupplier) GetId() SupplierId {
	return jmapContactCardSupplierId
}

func (c *JmapContactCardSupplier) IsMine(id string) bool {
	return id != "" && !strings.Contains(id, ":")
}

func (c *JmapContactCardSupplier) Query(accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, filter jmap.ContactCardFilterElement, sortBy []jmap.ContactCardComparator, calculateTotal bool, ctx jmap.Context) (jmap.Result[map[jmap.AccountId]*jmap.ContactCardSearchResults], error) { //NOSONAR
	if m, err := mapQueryParams(c.GetId(), accountIds, qps); err != nil {
		return jmap.ZeroResultV[map[jmap.AccountId]*jmap.ContactCardSearchResults](), err
	} else {
		return c.client.QueryContactCards(m, limit, filter, sortBy, calculateTotal, ctx)
	}
}

func (c *JmapContactCardSupplier) GetAll(accountId jmap.AccountId, ids []string, ctx jmap.Context) (jmap.Result[jmap.ContactCardGetResponse], error) {
	return c.client.GetContactCards(accountId, ids, ctx)
}

func (c *JmapContactCardSupplier) GetChanges(accountId jmap.AccountId, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (jmap.Result[jmap.ContactCardChanges], error) {
	return c.client.GetContactCardChanges(accountId, sinceState, maxChanges, ctx)
}

func (c *JmapContactCardSupplier) CanCreate(accountId jmap.AccountId, create jmap.ContactCardChange, ctx jmap.Context) bool {
	return true
}

func (c *JmapContactCardSupplier) Create(accountId jmap.AccountId, create jmap.ContactCardChange, ctx jmap.Context) (jmap.Result[*jmap.ContactCard], error) {
	return c.client.CreateContactCard(accountId, create, ctx)
}
