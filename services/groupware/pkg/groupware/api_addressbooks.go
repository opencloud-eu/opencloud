package groupware

import (
	"net/http"
)

// Get all addressbooks of an account.
func (g *Groupware) GetAddressbooks(w http.ResponseWriter, r *http.Request) {
	getall(AddressBook, w, r, g, g.jmap.GetAddressbooks)
}

// Get an addressbook of an account by its identifier.
func (g *Groupware) GetAddressbookById(w http.ResponseWriter, r *http.Request) {
	get(AddressBook, w, r, g, g.jmap.GetAddressbooks)
}

// Get the changes to Address Books since a certain State.
// @api:tags addressbook,changes
func (g *Groupware) GetAddressBookChanges(w http.ResponseWriter, r *http.Request) {
	changes(AddressBook, w, r, g, g.jmap.GetAddressbookChanges)
}

func (g *Groupware) CreateAddressBook(w http.ResponseWriter, r *http.Request) {
	create(AddressBook, w, r, g, nil, g.jmap.CreateAddressBook)
}

func (g *Groupware) DeleteAddressBook(w http.ResponseWriter, r *http.Request) {
	delete(AddressBook, w, r, g, g.jmap.DeleteAddressBook)
}

func (g *Groupware) ModifyAddressBook(w http.ResponseWriter, r *http.Request) {
	modify(AddressBook, w, r, g, g.jmap.UpdateAddressBook)
}
