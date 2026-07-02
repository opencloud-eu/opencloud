package mapping

import (
	"reflect"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestInferTypeUnsupported(t *testing.T) {
	if got := inferType(reflect.TypeFor[map[string]int]()); got != "" {
		t.Errorf("map: got %q, want empty", got)
	}
	if got := inferType(reflect.TypeFor[chan int]()); got != "" {
		t.Errorf("chan: got %q, want empty", got)
	}
}

func TestInferType(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"string", "", TypeKeyword},
		{"*string", (*string)(nil), TypeKeyword},
		{"[]string", []string(nil), TypeKeyword},
		{"bool", false, TypeBool},
		{"int", int(0), TypeNumeric},
		{"int64", int64(0), TypeNumeric},
		{"uint64", uint64(0), TypeNumeric},
		{"float64", float64(0), TypeNumeric},
		{"time.Time", time.Time{}, TypeDatetime},
		{"*time.Time", (*time.Time)(nil), TypeDatetime},
		{"*timestamppb.Timestamp", (*timestamppb.Timestamp)(nil), TypeDatetime},
		{"struct", struct{ X int }{}, TypeObject},
		{"*struct", (*struct{ X int })(nil), TypeObject},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := inferType(reflect.TypeOf(c.in))
			if got != c.want {
				t.Fatalf("inferType(%s): got %q, want %q", c.name, got, c.want)
			}
		})
	}
}

func TestResolveField(t *testing.T) {
	type S struct {
		Exported   string `json:"exp"`
		Renamed    string `json:"renamed,omitempty"`
		NoTag      string
		OmitOnly   string `json:",omitempty"`
		Skipped    string `json:"-"`
		unexported string //nolint:unused
	}
	st := reflect.TypeFor[S]()
	cases := []struct {
		fieldIdx int
		wantName string
		wantSkip bool
	}{
		{0, "exp", false},
		{1, "renamed", false},
		{2, "NoTag", false},
		{3, "OmitOnly", false},
		{4, "", true},
		{5, "", true},
	}
	for _, c := range cases {
		fi := resolveField(st.Field(c.fieldIdx))
		if fi.Skip != c.wantSkip {
			t.Errorf("field %d: skip=%v, want %v", c.fieldIdx, fi.Skip, c.wantSkip)
		}
		if !c.wantSkip && fi.Name != c.wantName {
			t.Errorf("field %d: name=%q, want %q", c.fieldIdx, fi.Name, c.wantName)
		}
	}
}

func TestWalkFieldsFlattensEmbedded(t *testing.T) {
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
	if err != nil {
		t.Fatalf("walkFields: %v", err)
	}
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("got %v, want %v", names, want)
	}
}

func TestStructType(t *testing.T) {
	type S struct{ X int }
	cases := []struct {
		name    string
		in      reflect.Type
		wantNil bool
	}{
		{"struct", reflect.TypeFor[S](), false},
		{"*struct", reflect.TypeFor[*S](), false},
		{"[]struct", reflect.TypeFor[[]S](), false},
		{"time.Time", reflect.TypeFor[time.Time](), true},
		{"string", reflect.TypeFor[string](), true},
	}
	for _, c := range cases {
		got := structType(c.in)
		if (got == nil) != c.wantNil {
			t.Errorf("%s: got %v, wantNil %v", c.name, got, c.wantNil)
		}
	}
}
