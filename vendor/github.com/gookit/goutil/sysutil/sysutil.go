// Package sysutil provide some system util functions. eg: sysenv, exec, user, process
package sysutil

import (
	"os"
	"path/filepath"
)

// Workdir get
func Workdir() string {
	dir, _ := os.Getwd()
	return dir
}

// BinDir get
func BinDir() string {
	return filepath.Dir(os.Args[0])
}

// BinName get
func BinName() string {
	return filepath.Base(os.Args[0])
}

// BinFile get
func BinFile() string {
	return os.Args[0]
}

// Open file or url address
func Open(fileOrURL string) error { return OpenURL(fileOrURL) }

// OpenBrowser file or url address
func OpenBrowser(fileOrURL string) error { return OpenURL(fileOrURL) }

// OpenFile open files browser window for the file path.
func OpenFile(path string) error {
	fpath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return OpenURL("file://" + fpath)
}
