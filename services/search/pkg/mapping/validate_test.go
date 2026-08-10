package mapping

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type inner struct {
	Artist string `json:"artist"`
}

type sample struct {
	Name     string    `json:"Name"`
	Audio    *inner    `json:"audio,omitempty"`
	Location *struct { //nolint:unused
		Lon float64 `json:"longitude"`
		Lat float64 `json:"latitude"`
	} `json:"location,omitempty"`
}

var _ = Describe("Validate", func() {
	It("accepts known override keys", func() {
		err := Validate(reflect.TypeFor[sample](), map[string]FieldOpts{
			"Name":         {},
			"audio":        {Type: TypeObject},
			"audio.artist": {},
			"location":     {Type: TypeGeopoint},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("rejects unknown override keys", func() {
		err := Validate(reflect.TypeFor[sample](), map[string]FieldOpts{
			"nope":      {},
			"audio.zzz": {},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("nope"))
		Expect(err.Error()).To(ContainSubstring("audio.zzz"))
	})

	It("accepts empty overrides", func() {
		Expect(Validate(reflect.TypeFor[sample](), nil)).To(Succeed())
	})

	It("rejects CaseInsensitive on a non-keyword/path field", func() {
		True := true
		err := Validate(reflect.TypeFor[sample](), map[string]FieldOpts{
			"Name": {Type: TypeFulltext, CaseInsensitive: &True},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Name"))
		Expect(err.Error()).To(ContainSubstring("CaseInsensitive"))
	})
})
