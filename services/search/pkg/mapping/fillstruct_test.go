package mapping

import (
	"errors"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The walker is shared by both deserializers, so its structural behavior
// (flattening embedded structs, recursing into nested pointers, joining the
// prefix, keeping a pointer only when a field was set, fail-soft skipping) is
// tested here once against a trivial string setter instead of twice through
// Deserialize and DeserializeStringsAt.

// FsEmbVal/FsEmbPtr are exported so their embedded field is exported; an
// unexported embedded type would be skipped by resolveField.
type FsEmbVal struct {
	EV string `json:"ev"`
}

type FsEmbPtr struct {
	EP string `json:"ep"`
}

type fsNested struct {
	N string `json:"n"`
}

type fsRoot struct {
	FsEmbVal            // embedded value struct: fields promoted
	*FsEmbPtr           // embedded pointer struct: allocated on demand
	Leaf      string    `json:"leaf"`
	Nested    *fsNested `json:"nested"` // nested pointer: recursed under "nested."
}

var errBadLeaf = errors.New("bad leaf")

// fsSet writes raw into a string field; the "BAD" sentinel simulates a parse
// failure so the fail-soft path can be exercised.
func fsSet(v reflect.Value, raw string) error {
	if raw == "BAD" {
		return errBadLeaf
	}
	v.SetString(raw)
	return nil
}

var _ = Describe("fillStruct", func() {
	fill := func(fields map[string]string, prefix string) (fsRoot, bool) {
		var root fsRoot
		touched := fillStruct(reflect.ValueOf(&root).Elem(), fields, prefix, fsSet)
		return root, touched
	}

	It("flattens embedded and recurses nested", func() {
		root, touched := fill(map[string]string{
			"leaf":     "L",
			"ev":       "EV",
			"ep":       "EP",
			"nested.n": "N",
		}, "")
		Expect(touched).To(BeTrue())
		Expect(root.Leaf).To(Equal("L"))
		Expect(root.EV).To(Equal("EV"), "embedded value not promoted")
		Expect(root.FsEmbPtr).ToNot(BeNil(), "embedded pointer not allocated")
		Expect(root.EP).To(Equal("EP"))
		Expect(root.Nested).ToNot(BeNil(), "nested pointer not populated")
		Expect(root.Nested.N).To(Equal("N"))
	})

	It("leaves touched false and pointers nil when nothing matches", func() {
		root, touched := fill(map[string]string{"other": "x"}, "")
		Expect(touched).To(BeFalse())
		Expect(root.Nested).To(BeNil(), "Nested should stay nil")
		Expect(root.FsEmbPtr).To(BeNil(), "embedded pointer should stay nil")
	})

	It("joins the prefix arg with the field name", func() {
		root, touched := fill(map[string]string{
			"pre.leaf":     "L",
			"pre.nested.n": "N",
		}, "pre")
		Expect(touched).To(BeTrue())
		Expect(root.Leaf).To(Equal("L"))
		Expect(root.Nested).ToNot(BeNil())
		Expect(root.Nested.N).To(Equal("N"))
	})

	It("is fail-soft: errored leaf stays zero, walk continues", func() {
		root, touched := fill(map[string]string{
			"leaf": "BAD",
			"ev":   "EV",
		}, "")
		Expect(touched).To(BeTrue(), "expected touched because ev was set")
		Expect(root.Leaf).To(BeEmpty(), "errored leaf should stay zero")
		Expect(root.EV).To(Equal("EV"), "walk should continue past the error")
	})

	It("drops the embedded pointer when its only field errors", func() {
		root, touched := fill(map[string]string{"ep": "BAD"}, "")
		Expect(touched).To(BeFalse())
		Expect(root.FsEmbPtr).To(BeNil(), "embedded pointer should stay nil on error")
	})
})
