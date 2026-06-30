package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

// Get all addressbooks of an account.
func (g *Groupware) GetAddressbooks(w http.ResponseWriter, r *http.Request) {
	getall(AddressBook, w, r, g, g.listAddressBooks)
}

// Get an addressbook of an account by its identifier.
func (g *Groupware) GetAddressbookById(w http.ResponseWriter, r *http.Request) {
	get(AddressBook, w, r, g, g.listAddressBooks)
}

// Get the changes to Address Books since a certain State.
// @api:tags addressbook,changes
func (g *Groupware) GetAddressBookChanges(w http.ResponseWriter, r *http.Request) {
	changes(AddressBook, w, r, g, g.addressBooksChanges)
}

func (g *Groupware) CreateAddressBook(w http.ResponseWriter, r *http.Request) {
	create(AddressBook, w, r, g, nil, g.createAddressBook)
}

func (g *Groupware) DeleteAddressBook(w http.ResponseWriter, r *http.Request) {
	deleteById(AddressBook, w, r, g, g.deleteAddressBooks)
}

func (g *Groupware) ModifyAddressBook(w http.ResponseWriter, r *http.Request) {
	modify(AddressBook, w, r, g, g.updateAddressBook)
}

func (g *Groupware) createAddressBook(accountId jmap.AccountId, addressbook jmap.AddressBookChange, ctx jmap.Context) (jmap.Result[*jmap.AddressBook], error) {
	return screate(g.suppliers.addressBooks.create, accountId, addressbook, ctx)
}

func (g *Groupware) updateAddressBook(accountId jmap.AccountId, id string, addressbook jmap.AddressBookChange, ctx jmap.Context) (jmap.Result[jmap.AddressBook], error) {
	return supdate(g.suppliers.addressBooks.update, accountId, id, addressbook, ctx)
}

func (g *Groupware) deleteAddressBooks(accountId jmap.AccountId, destroyIds []string, ctx jmap.Context) (jmap.Result[map[string]jmap.SetError], error) {
	return sdelete[jmap.AddressBook](g.suppliers.addressBooks.delete, accountId, destroyIds, ctx)
}

func (g *Groupware) listAddressBooks(accountId jmap.AccountId, ids []string, ctx jmap.Context) (jmap.Result[jmap.AddressBookGetResponse], error) {
	return slist(g.suppliers.addressBooks.list, accountId, ids, ctx, func(accountId jmap.AccountId, state jmap.State, notFound []string, list []jmap.AddressBook) jmap.AddressBookGetResponse {
		return jmap.AddressBookGetResponse{AccountId: accountId, State: state, NotFound: notFound, List: list}
	})
}

func (g *Groupware) addressBooksChanges(accountId jmap.AccountId, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (jmap.Result[jmap.AddressBookChanges], error) {
	return schanges(g.suppliers.addressBooks.changes, accountId, sinceState, maxChanges, ctx,
		func(accountId jmap.AccountId, oldState, newState jmap.State, created, updated []jmap.AddressBook, destroyed []string, hasMoreChanges bool) jmap.AddressBookChanges {
			return jmap.AddressBookChanges{HasMoreChanges: hasMoreChanges, OldState: oldState, NewState: newState, Created: created, Updated: updated, Destroyed: destroyed}
		},
	)
}
