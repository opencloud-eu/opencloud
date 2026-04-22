package metadata

import (
	"context"
	"io/fs"
	"os"
	"time"

	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

type FileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
	mode    os.FileMode
}

func newFileInfo(name string, fs *Fs, fileMode os.FileMode) (*FileInfo, error) {
	info, err := fs.storage.Stat(context.Background(), name)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		name:    info.GetName(),
		size:    int64(info.GetSize()),
		modTime: utils.TSToTime(info.GetMtime()),
		isDir:   info.GetType() == providerv1beta1.ResourceType_RESOURCE_TYPE_CONTAINER,
		mode:    fileMode,
	}, nil
}

func (f *FileInfo) Name() string {
	return f.name
}

func (f *FileInfo) Size() int64 {
	return f.size
}

func (f *FileInfo) ModTime() time.Time {
	return f.modTime
}

func (f *FileInfo) IsDir() bool {
	return f.isDir
}

func (f *FileInfo) Mode() fs.FileMode {
	return f.mode
}

func (f *FileInfo) Sys() any {
	return nil
}
