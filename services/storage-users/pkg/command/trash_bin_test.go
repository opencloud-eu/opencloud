package command

import (
	"path/filepath"
	"testing"
)

func Test_modifyFilename(t *testing.T) {
	type args struct {
		filename string
		mod      int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "file",
			args: args{filename: "file.txt", mod: 1},
			want: "file (1).txt",
		},
		{
			name: "file with path",
			args: args{filename: "./file.txt", mod: 1},
			want: "./file (1).txt",
		},
		{
			name: "file with path 2",
			args: args{filename: "./subdir/file.tar.gz", mod: 99},
			want: "./subdir/file (99).tar.gz",
		},
		{
			name: "file with path 3",
			args: args{filename: "./sub dir/new file.tar.gz", mod: 99},
			want: "./sub dir/new file (99).tar.gz",
		},
		{
			name: "file without ext",
			args: args{filename: "./subdir/file", mod: 2},
			want: "./subdir/file (2)",
		},
		{
			name: "file without ext 2",
			args: args{filename: "./subdir/file 1", mod: 2},
			want: "./subdir/file 1 (2)",
		},
		{
			name: "file with emoji",
			args: args{filename: "./subdir/file 🙂.tar.gz", mod: 3},
			want: "./subdir/file 🙂 (3).tar.gz",
		},
		{
			name: "file with emoji 2",
			args: args{filename: "./subdir/file 🙂", mod: 2},
			want: "./subdir/file 🙂 (2)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modifyFilename(tt.args.filename, tt.args.mod)
			want := filepath.FromSlash(tt.want)
			if got != want {
				t.Errorf("modifyFilename() = %v, want %v", got, want)
			}
		})
	}
}
