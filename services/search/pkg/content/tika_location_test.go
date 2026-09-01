package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("getLocation", func() {
	It("maps lat/long and converts altitude from metres to feet", func() {
		metres := 227.4
		location := Tika{}.getLocation(map[string][]string{
			"geo:lat":  {"49.48675890884328"},
			"geo:long": {"11.103870357204285"},
			"geo:alt":  {"227.4"},
		})
		Expect(location).ToNot(BeNil())
		Expect(location.Latitude).To(Equal(libregraph.PtrFloat64(49.48675890884328)))
		Expect(location.Longitude).To(Equal(libregraph.PtrFloat64(11.103870357204285)))
		Expect(location.Altitude).To(Equal(libregraph.PtrFloat64(metres * metresToFeet)))
	})

	It("keeps below-sea-level altitude negative", func() {
		metres := -227.4
		location := Tika{}.getLocation(map[string][]string{
			"geo:lat":  {"31.5"},
			"geo:long": {"35.47"},
			"geo:alt":  {"-227.4"},
		})
		Expect(location).ToNot(BeNil())
		Expect(location.Altitude).To(Equal(libregraph.PtrFloat64(metres * metresToFeet)))
	})

	It("returns nil for an incomplete coordinate pair", func() {
		Expect(Tika{}.getLocation(map[string][]string{"geo:lat": {"49.48"}})).To(BeNil())
		Expect(Tika{}.getLocation(map[string][]string{"geo:alt": {"227.4"}})).To(BeNil())
	})

	It("returns nil for out-of-range coordinates", func() {
		Expect(Tika{}.getLocation(map[string][]string{"geo:lat": {"91"}, "geo:long": {"11.1"}})).To(BeNil())
		Expect(Tika{}.getLocation(map[string][]string{"geo:lat": {"49.48"}, "geo:long": {"-180.5"}})).To(BeNil())
		Expect(Tika{}.getLocation(map[string][]string{"geo:lat": {"NaN"}, "geo:long": {"11.1"}})).To(BeNil())
	})

	It("returns nil when no location metadata is present", func() {
		Expect(Tika{}.getLocation(map[string][]string{})).To(BeNil())
	})
})
