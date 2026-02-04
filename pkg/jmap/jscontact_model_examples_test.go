//go:build groupware_examples

package jmap

import c "github.com/opencloud-eu/opencloud/pkg/jscontact"

func JSContactExample() {
	SerializeExamples(JSContactExemplarInstance)
	//Output:
}

type JSContactExemplar struct {
}

var JSContactExemplarInstance = JSContactExemplar{}

func (e JSContactExemplar) Name() c.Name {
	return c.Name{
		Type: c.NameType,
		Components: []c.NameComponent{
			{Type: c.NameComponentType, Value: "Drummer", Kind: c.NameComponentKindGiven},
			{Type: c.NameComponentType, Value: "Camina", Kind: c.NameComponentKindGiven},
		},
		IsOrdered:        true,
		DefaultSeparator: ", ",
		Full:             "Camina Drummer",
	}
}
