package groupware

import (
	"net/http"
)

// Get the list of identities that are associated with an account.
func (g *Groupware) GetIdentities(w http.ResponseWriter, r *http.Request) {
	getall(Identity, w, r, g, g.jmap.GetIdentities)
}

func (g *Groupware) GetIdentityById(w http.ResponseWriter, r *http.Request) {
	get(Identity, w, r, g, g.jmap.GetIdentities)
}

func (g *Groupware) CreateIdentity(w http.ResponseWriter, r *http.Request) {
	create(Identity, w, r, g, nil, g.jmap.CreateIdentity)
}

func (g *Groupware) ModifyIdentity(w http.ResponseWriter, r *http.Request) {
	modify(Identity, w, r, g, g.jmap.UpdateIdentity)
}

// Delete an identity.
func (g *Groupware) DeleteIdentity(w http.ResponseWriter, r *http.Request) {
	delete(Identity, w, r, g, g.jmap.DeleteIdentity)
}

// Get changes to Identities since a given State
// @api:tags identity,changes
func (g *Groupware) GetIdentityChanges(w http.ResponseWriter, r *http.Request) {
	changes(Identity, w, r, g, g.jmap.GetIdentityChanges)
}
