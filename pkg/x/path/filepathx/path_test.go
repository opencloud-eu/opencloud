package filepathx_test

import (
	"path/filepath"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/x/path/filepathx"
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := filepathx.JailJoin(tt.args.jail, tt.args.elem...)
			// Make expectation OS-agnostic: convert unix-style expectation to OS-specific path
			want := filepath.FromSlash(tt.want)
			if got != want {
				t.Errorf("JailJoin() = %v, want %v", got, want)
			}
		})
	}
}
