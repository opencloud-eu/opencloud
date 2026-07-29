package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("getLocation", func() {
	It("maps latitude and longitude to the location facet", func() {
		location := Tika{}.getLocation(map[string][]string{
			"geo:lat":  {"49.48675890884328"},
			"geo:long": {"11.103870357204285"},
		})
		Expect(location).ToNot(BeNil())
		Expect(location.Latitude).To(Equal(libregraph.PtrFloat64(49.48675890884328)))
		Expect(location.Longitude).To(Equal(libregraph.PtrFloat64(11.103870357204285)))
	})

	It("returns nil when no location metadata is present", func() {
		Expect(Tika{}.getLocation(map[string][]string{})).To(BeNil())
	})
})
