package conversions_test

import (
	"fmt"
	"testing"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
)

func checkIdentical[T any](t *testing.T, p T, want string) {
	t.Helper()
	got := fmt.Sprintf("%T", p)
	if got != want {
		t.Errorf("want:%q got:%q", want, got)
	}
}

func TestToPointer2(t *testing.T) {
	checkIdentical(t, conversions.ToPointer("a"), "*string")
	checkIdentical(t, conversions.ToPointer(1), "*int")
	checkIdentical(t, conversions.ToPointer(-1), "*int")
	checkIdentical(t, conversions.ToPointer(float64(1)), "*float64")
	checkIdentical(t, conversions.ToPointer(float64(-1)), "*float64")
	checkIdentical(t, conversions.ToPointer(libregraph.UnifiedRoleDefinition{}), "*libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToPointer([]string{"a"}), "*[]string")
	checkIdentical(t, conversions.ToPointer([]int{1}), "*[]int")
	checkIdentical(t, conversions.ToPointer([]float64{1}), "*[]float64")
	checkIdentical(t, conversions.ToPointer([]libregraph.UnifiedRoleDefinition{{}}), "*[]libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToPointer(conversions.ToPointer("a")), "**string")
	checkIdentical(t, conversions.ToPointer(conversions.ToPointer(1)), "**int")
	checkIdentical(t, conversions.ToPointer(conversions.ToPointer(-1)), "**int")
	checkIdentical(t, conversions.ToPointer(conversions.ToPointer(float64(1))), "**float64")
	checkIdentical(t, conversions.ToPointer(conversions.ToPointer(float64(-1))), "**float64")
	checkIdentical(t, conversions.ToPointer(conversions.ToPointer(libregraph.UnifiedRoleDefinition{})), "**libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToPointer(conversions.ToPointer([]string{"a"})), "**[]string")
	checkIdentical(t, conversions.ToPointer(conversions.ToPointer([]int{1})), "**[]int")
	checkIdentical(t, conversions.ToPointer(conversions.ToPointer([]float64{1})), "**[]float64")
	checkIdentical(t, conversions.ToPointer(conversions.ToPointer([]libregraph.UnifiedRoleDefinition{{}})), "**[]libregraph.UnifiedRoleDefinition")
}

func TestToValue(t *testing.T) {
	checkIdentical(t, conversions.ToValue((*int)(nil)), "int")
	checkIdentical(t, conversions.ToValue((*string)(nil)), "string")
	checkIdentical(t, conversions.ToValue((*float64)(nil)), "float64")
	checkIdentical(t, conversions.ToValue((*libregraph.UnifiedRoleDefinition)(nil)), "libregraph.UnifiedRoleDefinition")
	checkIdentical(t, conversions.ToValue((*[]string)(nil)), "[]string")
	checkIdentical(t, conversions.ToValue((*[]libregraph.UnifiedRoleDefinition)(nil)), "[]libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToValue(conversions.ToPointer("a")), "string")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(1)), "int")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(-1)), "int")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(float64(1))), "float64")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(float64(-1))), "float64")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(libregraph.UnifiedRoleDefinition{})), "libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToValue(conversions.ToPointer([]string{"a"})), "[]string")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer([]int{1})), "[]int")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer([]float64{1})), "[]float64")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer([]libregraph.UnifiedRoleDefinition{{}})), "[]libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer("a"))), "*string")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer(1))), "*int")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer(-1))), "*int")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer(float64(1)))), "*float64")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer(float64(-1)))), "*float64")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer(libregraph.UnifiedRoleDefinition{}))), "*libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer([]string{"a"}))), "*[]string")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer([]int{1}))), "*[]int")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer([]float64{1}))), "*[]float64")
	checkIdentical(t, conversions.ToValue(conversions.ToPointer(conversions.ToPointer([]libregraph.UnifiedRoleDefinition{{}}))), "*[]libregraph.UnifiedRoleDefinition")
}

func TestToPointerSlice(t *testing.T) {
	checkIdentical(t, conversions.ToPointerSlice([]string{"a"}), "[]*string")
	checkIdentical(t, conversions.ToPointerSlice([]int{1}), "[]*int")
	checkIdentical(t, conversions.ToPointerSlice([]libregraph.UnifiedRoleDefinition{{}}), "[]*libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToPointerSlice(([]string)(nil)), "[]*string")
	checkIdentical(t, conversions.ToPointerSlice(([]int)(nil)), "[]*int")
	checkIdentical(t, conversions.ToPointerSlice(([]libregraph.UnifiedRoleDefinition)(nil)), "[]*libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToPointerSlice([]*string{conversions.ToPointer("a")}), "[]**string")
	checkIdentical(t, conversions.ToPointerSlice([]*int{conversions.ToPointer(1)}), "[]**int")
	checkIdentical(t, conversions.ToPointerSlice(([]*libregraph.UnifiedRoleDefinition)(nil)), "[]**libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToPointerSlice(([]*string)(nil)), "[]**string")
	checkIdentical(t, conversions.ToPointerSlice(([]*int)(nil)), "[]**int")
	checkIdentical(t, conversions.ToPointerSlice(([]*libregraph.UnifiedRoleDefinition)(nil)), "[]**libregraph.UnifiedRoleDefinition")
}

func TestToValueSlice(t *testing.T) {
	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice([]string{"a"})), "[]string")
	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice([]int{1})), "[]int")
	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice([]libregraph.UnifiedRoleDefinition{{}})), "[]libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice(([]string)(nil))), "[]string")
	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice(([]int)(nil))), "[]int")
	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice(([]libregraph.UnifiedRoleDefinition)(nil))), "[]libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice([]*string{conversions.ToPointer("a")})), "[]*string")
	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice([]*int{conversions.ToPointer(1)})), "[]*int")
	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice([]*libregraph.UnifiedRoleDefinition{conversions.ToPointer(libregraph.UnifiedRoleDefinition{})})), "[]*libregraph.UnifiedRoleDefinition")

	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice(([]*string)(nil))), "[]*string")
	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice(([]*int)(nil))), "[]*int")
	checkIdentical(t, conversions.ToValueSlice(conversions.ToPointerSlice(([]*libregraph.UnifiedRoleDefinition)(nil))), "[]*libregraph.UnifiedRoleDefinition")
}
