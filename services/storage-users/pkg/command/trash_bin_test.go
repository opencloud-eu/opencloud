package command

import (
	"testing"

	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"

	"github.com/spf13/cobra"
)

func Test_trashBinCommandUse(t *testing.T) {
	cfg := &config.Config{}
	tests := []struct {
		name string
		cmd  *cobra.Command
		want string
	}{
		{"list", listTrashBinItems(cfg), "list space"},
		{"restore-all", restoreAllTrashBinItems(cfg), "restore-all space"},
		{"restore", restoreTrashBinItem(cfg), "restore space item"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.Use; got != tt.want {
				t.Errorf("Use = %q, want %q", got, tt.want)
			}
		})
	}
}

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
			if got := modifyFilename(tt.args.filename, tt.args.mod); got != tt.want {
				t.Errorf("modifyFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}
