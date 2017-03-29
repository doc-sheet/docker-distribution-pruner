package main

import (
	"io/ioutil"
	"path/filepath"
	"syscall"
	"flag"
	"os"
	"strings"
)

type fsStorage struct {
	rootDir string
}

var fsRootDir = flag.String("fs-root-dir", "examples/registry", "root directory")

func (f *fsStorage) fullPath(path string) string {
	return filepath.Join(*fsRootDir, "docker", "registry", "v2", path)
}

func (f *fsStorage) Walk(rootDir string, fn walkFunc) error {
	rootDir, err := filepath.Abs(f.fullPath(rootDir))
	if err != nil {
		return nil
	}
	rootDir += "/"

	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if strings.HasPrefix(path, rootDir) {
			path = path[len(rootDir):]
		}

		fi := fileInfo{path: path, size: info.Size()}
		return fn(path, fi, err)
	})
}

func (f *fsStorage) Read(path string) ([]byte, error) {
	return ioutil.ReadFile(f.fullPath(path))
}

func (f *fsStorage) Delete(path string) error {
	return syscall.Unlink(f.fullPath(path))
}
