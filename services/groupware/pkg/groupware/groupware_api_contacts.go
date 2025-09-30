package groupware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/jscontact"
)

// When the request succeeds.
// swagger:response GetAddressbooks200
type SwaggerGetAddressbooks200 struct {
	// in: body
	Body []jmap.AddressBook
}

// swagger:route GET /groupware/accounts/{account}/addressbooks addressbook addressbooks
// Get all addressbooks of an account.
//
// responses:
//
//	200: GetAddressbooks200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetAddressbooks(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		// TODO replace with proper implementation
		return response(AllAddressBooks, req.session.State)
	})
}

// When the request succeeds.
// swagger:response GetAddressbookById200
type SwaggerGetAddressbookById200 struct {
	// in: body
	Body struct {
		*jmap.AddressBook
	}
}

// swagger:route GET /groupware/accounts/{account}/addressbooks/{addressbookid} addressbook addressbook_by_id
// Get an addressbook of an account by its identifier.
//
// responses:
//
//	200: GetAddressbookById200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetAddressbook(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		addressBookId := chi.URLParam(r, UriParamAddressBookId)
		// TODO replace with proper implementation
		for _, ab := range AllAddressBooks {
			if ab.Id == addressBookId {
				return response(ab, req.session.State)
			}
		}
		return notFoundResponse(req.session.State)
	})
}

// When the request succeeds.
// swagger:response GetContactsInAddressbook200
type SwaggerGetContactsInAddressbook200 struct {
	// in: body
	Body []jscontact.ContactCard
}

// swagger:route GET /groupware/accounts/{account}/addressbooks/{addressbookid}/contacts contact contacts_in_addressbook
// Get all the contacts in an addressbook of an account by its identifier.
//
// responses:
//
//	200: GetContactsInAddressbook200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetContactsInAddressbook(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		addressBookId := chi.URLParam(r, UriParamAddressBookId)
		// TODO replace with proper implementation
		contactCards, ok := ContactsMapByAddressBookId[addressBookId]
		if !ok {
			return notFoundResponse(req.session.State)
		}
		return response(contactCards, req.session.State)
	})
}
