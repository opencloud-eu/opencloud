package groupware

import (
	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

var petObjectType = jmap.ObjectType{Name: jmap.ObjectTypeName("Pet"), Namespaces: []jmap.JmapNamespace{}}

type Pet struct {
	id   string
	name string
}

func (m Pet) GetId() string                  { return m.id }
func (m Pet) GetObjectType() jmap.ObjectType { return petObjectType }

var _ jmap.Idable = &Pet{}

type PetGetResponse struct {
	AccountId string
	State     jmap.State
	List      []Pet
}

var _ jmap.GetResponse[Pet] = &PetGetResponse{}

func (r PetGetResponse) GetMarker() Pet        { return Pet{} }
func (r PetGetResponse) GetState() jmap.State  { return r.State }
func (r PetGetResponse) GetNotFound() []string { return []string{} }
func (r PetGetResponse) GetList() []Pet        { return r.List }

type PetSearchResults jmap.SearchResultsTemplate[Pet]

var _ jmap.SearchResults[Pet] = &PetSearchResults{}

func (r *PetSearchResults) GetResults() []Pet { return r.Results }
func (r *PetSearchResults) GetCanCalculateChanges() jmap.ChangeCalculation {
	return r.CanCalculateChanges
}
func (r *PetSearchResults) GetPosition() *uint         { return r.Position }
func (r *PetSearchResults) GetLimit() *uint            { return r.Limit }
func (r *PetSearchResults) GetTotal() *uint            { return r.Total }
func (r *PetSearchResults) RemoveResults()             { r.Results = nil }
func (r *PetSearchResults) SetLimit(limit *uint)       { r.Limit = limit }
func (r *PetSearchResults) SetPosition(position *uint) { r.Position = position }

type PetComparator struct {
	Property    string
	IsAscending bool
}

var _ jmap.Comparator[Pet] = &PetComparator{}

func (c PetComparator) GetMarker() Pet { return Pet{} }

type PetFilterElement interface {
	_isAPetFilterElement() // marker method
	IsNotEmpty() bool
	jmap.FilterElement[Pet]
}

type PetFilterCondition struct {
	id string
}

var _ PetFilterElement = &PetFilterCondition{}

var _ jmap.FilterElement[Pet] = &PetFilterCondition{}

func (f PetFilterCondition) GetMarker() Pet { return Pet{} }

func (f PetFilterCondition) _isAPetFilterElement() { //NOSONAR
	// marker interface method, does not need to do anything
}

func (f PetFilterCondition) IsNotEmpty() bool { //NOSONAR
	if f.id != "" {
		return true
	}
	return false
}

type PetFilterOperator struct {
	Operator   jmap.FilterOperatorTerm
	Conditions []PetFilterElement
}

var _ PetFilterElement = &PetFilterOperator{}

var _ jmap.FilterElement[Pet] = &PetFilterOperator{}

func (f PetFilterOperator) GetMarker() Pet { return Pet{} }

func (o PetFilterOperator) _isAPetFilterElement() { //NOSONAR
	// marker interface method, does not need to do anything
}

func (o PetFilterOperator) IsNotEmpty() bool {
	return len(o.Conditions) > 0
}
