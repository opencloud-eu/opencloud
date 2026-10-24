package mapping

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("inferType", func() {
	It("returns empty for unsupported kinds", func() {
		Expect(inferType(reflect.TypeFor[map[string]int]())).To(BeEmpty(), "map")
		Expect(inferType(reflect.TypeFor[chan int]())).To(BeEmpty(), "chan")
	})

	DescribeTable("infers the mapping type",
		func(in any, want string) {
			Expect(inferType(reflect.TypeOf(in))).To(Equal(want))
		},
		Entry("string", "", TypeKeyword),
		Entry("*string", (*string)(nil), TypeKeyword),
		Entry("[]string", []string(nil), TypeKeyword),
		Entry("bool", false, TypeBool),
		Entry("int", int(0), TypeNumeric),
		Entry("int64", int64(0), TypeNumeric),
		Entry("uint64", uint64(0), TypeNumeric),
		Entry("float64", float64(0), TypeNumeric),
		Entry("time.Time", time.Time{}, TypeDatetime),
		Entry("*time.Time", (*time.Time)(nil), TypeDatetime),
		Entry("*timestamppb.Timestamp", (*timestamppb.Timestamp)(nil), TypeDatetime),
		Entry("struct", struct{ X int }{}, TypeObject),
		Entry("*struct", (*struct{ X int })(nil), TypeObject),
	)
})

var _ = Describe("resolveField", func() {
	type S struct {
		Exported   string `json:"exp"`
		Renamed    string `json:"renamed,omitempty"`
		NoTag      string
		OmitOnly   string `json:",omitempty"`
		Skipped    string `json:"-"`
		unexported string //nolint:unused
	}
	st := reflect.TypeFor[S]()

	DescribeTable("resolves the field name and skip flag",
		func(fieldIdx int, wantName string, wantSkip bool) {
			fi := resolveField(st.Field(fieldIdx))
			Expect(fi.Skip).To(Equal(wantSkip), "field %d skip", fieldIdx)
			if !wantSkip {
				Expect(fi.Name).To(Equal(wantName), "field %d name", fieldIdx)
			}
		},
		Entry("exported json tag", 0, "exp", false),
		Entry("renamed with omitempty", 1, "renamed", false),
		Entry("no tag", 2, "NoTag", false),
		Entry("omitempty only", 3, "OmitOnly", false),
		Entry("json:- skipped", 4, "", true),
		Entry("unexported skipped", 5, "", true),
	)
})

var _ = Describe("walkFields", func() {
	It("flattens embedded structs", func() {
		type Inner struct {
			A string `json:"a"`
			B int    `json:"b"`
		}
		type Outer struct {
			Inner
			C bool `json:"c"`
		}
		var names []string
		err := walkFields(reflect.TypeFor[Outer](), func(fi fieldInfo) error {
			names = append(names, fi.Name)
			return nil
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(names).To(Equal([]string{"a", "b", "c"}))
	})
})

var _ = Describe("structType", func() {
	type S struct{ X int }

	DescribeTable("resolves struct-ish types",
		func(in reflect.Type, wantNil bool) {
			got := structType(in)
			Expect(got == nil).To(Equal(wantNil))
		},
		Entry("struct", reflect.TypeFor[S](), false),
		Entry("*struct", reflect.TypeFor[*S](), false),
		Entry("[]struct", reflect.TypeFor[[]S](), false),
		Entry("time.Time", reflect.TypeFor[time.Time](), true),
		Entry("string", reflect.TypeFor[string](), true),
	)
})
