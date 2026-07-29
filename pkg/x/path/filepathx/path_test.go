package filepathx_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/x/path/filepathx"
	"github.com/stretchr/testify/require"
)

func TestJailJoin(t *testing.T) {
	type args struct {
		jail string
		elem []string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "regular use case",
			args: args{
				jail: "/",
				elem: []string{"a", "b", "c"},
			},
			want: "/a/b/c",
		},
		{
			name: "access parent directory",
			args: args{
				jail: "/",
				elem: []string{"a", "b", "c", ".."},
			},
			want: "/a/b",
		},
		{
			name: "restrict breaking out of jail",
			args: args{
				jail: "/",
				elem: []string{"a", "b", "c", "..", "..", "..", "..", "..", "..", ".."},
			},
			want: "/",
		},
		{
			name: "restrict to child of jail",
			args: args{
				jail: "/a/b",
				elem: []string{"a", "b", "c", "..", "..", "..", "..", "..", "..", ".."},
			},
			want: "/a/b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := filepathx.JailJoin(tt.args.jail, tt.args.elem...); got != tt.want {
				t.Errorf("JailJoin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSameOrContainedBy(t *testing.T) {
	for _, tt := range []struct {
		parent   string
		child    string
		expected bool
	}{
		{"foo", "foo", true},
		{"/foo", "/foo", true},
		{"foo", "foo/bar", true},
		{"foo", "bar", false},
	} {
		t.Run(fmt.Sprintf("%s: %s vs %s", t.Name(), strings.ReplaceAll(tt.parent, "/", "."), strings.ReplaceAll(tt.child, "/", ".")), func(t *testing.T) {
			require := require.New(t)
			b, err := filepathx.IsSameOrContainedBy(tt.parent, tt.child)
			require.NoError(err)
			require.Equal(tt.expected, b)
		})
	}
}
