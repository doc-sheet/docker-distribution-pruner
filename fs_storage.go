package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type fsStorage struct {
}

var fsRootDir = flag.String("fs-root-dir", "examples/registry", "root directory")

func newFsStorage() storageObject {
	return &fsStorage{}
}

func (f *fsStorage) fullPath(path string) string {
	return filepath.Join(*fsRootDir, "docker", "registry", "v2", path)
}

func (f *fsStorage) Walk(rootDir string, fn walkFunc) error {
	rootDir, err := filepath.Abs(f.fullPath(rootDir))
	if err != nil {
		return nil
	}
	rootDir += "/"

	return filepath.Walk(rootDir, func(fullPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		path := fullPath

		if strings.HasPrefix(path, rootDir) {
			path = path[len(rootDir):]
		}

		fi := fileInfo{fullPath: fullPath, size: info.Size()}
		return fn(path, fi, err)
	})
}

func (f *fsStorage) Read(path string, etag string) ([]byte, error) {
	return ioutil.ReadFile(f.fullPath(path))
}

func (f *fsStorage) Delete(path string) error {
	return syscall.Unlink(f.fullPath(path))
}

func (f *fsStorage) Info() {
}
