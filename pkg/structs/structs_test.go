package structs

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type example struct {
	Attribute1 string
	Attribute2 string
}

func TestCopyOrZeroValue(t *testing.T) {
	var e *example

	zv := CopyOrZeroValue(e)

	if zv == nil {
		t.Error("CopyOrZeroValue returned nil")
	}

	if zv.Attribute1 != "" || zv.Attribute2 != "" {
		t.Error("CopyOrZeroValue didn't return zero value")
	}

	e2 := &example{Attribute1: "One", Attribute2: "Two"}

	cp := CopyOrZeroValue(e2)

	if cp == nil {
		t.Error("CopyOrZeroValue returned nil")
	}

	if cp == e2 {
		t.Error("CopyOrZeroValue returned reference with same address")
	}

	if cp.Attribute1 != e2.Attribute1 || cp.Attribute2 != e2.Attribute2 {
		t.Error("CopyOrZeroValue didn't correctly copy attributes")
	}
}

func TestUniqWithInts(t *testing.T) {
	tests := []struct {
		input    []int
		expected []int
	}{
		{[]int{5, 1, 3, 1, 4}, []int{5, 1, 3, 4}},
		{[]int{1, 1, 1}, []int{1}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: testing %v", i+1, tt.input), func(t *testing.T) { //NOSONAR
			result := Uniq(tt.input)
			assert.EqualValues(t, tt.expected, result)
		})
	}
}

type u struct {
	x int
	y string
}

var (
	u1 = u{x: 1, y: "un"}
	u2 = u{x: 2, y: "deux"}
	u3 = u{x: 3, y: "trois"}
)

func TestUniqWithStructs(t *testing.T) {
	tests := []struct {
		input    []u
		expected []u
	}{
		{[]u{u3, u1, u2, u3, u2, u1}, []u{u3, u1, u2}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: testing %v", i+1, tt.input), func(t *testing.T) {
			result := Uniq(tt.input)
			assert.EqualValues(t, tt.expected, result)
		})
	}
}

func TestKeys(t *testing.T) {
	tests := []struct {
		input    map[int]string
		expected []int
	}{
		{map[int]string{5: "cinq", 1: "un", 3: "trois", 4: "vier"}, []int{5, 1, 3, 4}},
		{map[int]string{1: "un"}, []int{1}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: testing %v", i+1, tt.input), func(t *testing.T) {
			result := Keys(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestIndex(t *testing.T) {
	tests := []struct {
		input    []string
		indexer  func(string) string
		expected map[string]string
	}{
		{
			[]string{"un", "deux", "trois"},
			strings.ToUpper,
			map[string]string{"UN": "un", "DEUX": "deux", "TROIS": "trois"},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: testing %v", i+1, tt.input), func(t *testing.T) {
			result := Index(tt.input, tt.indexer)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMap(t *testing.T) {
	tests := []struct {
		input    []string
		mapper   func(string) int
		expected []int
	}{
		{
			nil,
			func(s string) int { return len(s) },
			nil,
		},
		{
			[]string{},
			func(s string) int { return len(s) },
			[]int{},
		},
		{
			[]string{"un", "deux", "trois"},
			func(s string) int { return len(s) },
			[]int{2, 4, 5},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: testing %v", i+1, tt.input), func(t *testing.T) {
			result := Map(tt.input, tt.mapper)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapSeq(t *testing.T) {
	tests := []struct {
		input    []string
		mapper   func(string) int
		expected []int
	}{
		{
			nil,
			func(s string) int { return len(s) },
			nil,
		},
		{
			[]string{},
			func(s string) int { return len(s) },
			nil,
		},
		{
			[]string{"un", "deux", "trois"},
			func(s string) int { return len(s) },
			[]int{2, 4, 5},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: testing %v", i+1, tt.input), func(t *testing.T) {
			result := slices.Collect(MapSeq(Seq(tt.input), tt.mapper))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConcat(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c", "d", "e", "f"}, Concat([]string{"a", "b"}, []string{"c"}, []string{"d", "e", "f"}))
	assert.Equal(t, []string{"a"}, Concat([]string{"a"}))
	assert.Equal(t, []string{"a"}, Concat([]string{}, nil, []string{"a"}))
	assert.Equal(t, []string{}, Concat[string]())
}

func TestFilter(t *testing.T) {
	tests := []struct {
		input     []int
		predicate func(int) bool
		expected  []int
	}{
		{
			nil,
			func(i int) bool { return i%2 == 0 },
			nil,
		},
		{
			[]int{},
			func(i int) bool { return i%2 == 0 },
			[]int{},
		},
		{
			[]int{1, 2, 3, 4, 5},
			func(i int) bool { return i%2 == 0 },
			[]int{2, 4},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: testing %v", i+1, tt.input), func(t *testing.T) {
			result := Filter(tt.input, tt.predicate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterSeq(t *testing.T) {
	tests := []struct {
		input     []int
		predicate func(int) bool
		expected  []int
	}{
		{
			nil,
			func(i int) bool { return i%2 == 0 },
			nil,
		},
		{
			[]int{},
			func(i int) bool { return i%2 == 0 },
			nil,
		},
		{
			[]int{1, 2, 3, 4, 5},
			func(i int) bool { return i%2 == 0 },
			[]int{2, 4},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: testing %v", i+1, tt.input), func(t *testing.T) {
			result := slices.Collect(FilterSeq(Seq(tt.input), tt.predicate))
			assert.Equal(t, tt.expected, result)
		})
	}
}
