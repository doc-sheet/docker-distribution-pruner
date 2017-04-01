package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type fsStorage struct {
	*distributionStorageFilesystem
}

func (f *fsStorage) fullPath(path string) string {
	return filepath.Join(f.RootDirectory, "docker", "registry", "v2", path)
}

func (f *fsStorage) backupPath(path string) string {
	return filepath.Join(f.RootDirectory, "docker_backup", "registry", "v2", path)
}

func (f *fsStorage) Walk(rootDir string, baseDir string, fn walkFunc) error {
	rootDir, err := filepath.Abs(f.fullPath(rootDir))
	if err != nil {
		return nil
	}
	rootDir += "/"

	baseDir, err = filepath.Abs(f.fullPath(baseDir))
	if err != nil {
		return nil
	}
	baseDir += "/"

	return filepath.Walk(rootDir, func(fullPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		path := fullPath

		if strings.HasPrefix(path, baseDir) {
			path = path[len(baseDir):]
		}

		if path == "" {
			return nil
		}

		fi := fileInfo{fullPath: fullPath, size: info.Size()}
		return fn(path, fi, err)
	})
}

func (f *fsStorage) List(rootDir string, fn walkFunc) error {
	rootDir, err := filepath.Abs(f.fullPath(rootDir))
	if err != nil {
		return nil
	}
	rootDir += "/"

	return filepath.Walk(rootDir, func(fullPath string, info os.FileInfo, err error) error {
		path := fullPath

		if strings.HasPrefix(path, rootDir) {
			path = path[len(rootDir):]
		}

		if path == "" {
			return nil
		}

		fi := fileInfo{fullPath: fullPath, size: info.Size(), directory: info.IsDir()}

		if info.IsDir() {
			err = fn(path, fi, err)
			if err != nil {
				return err
			}

			return filepath.SkipDir
		} else {

			fi := fileInfo{fullPath: fullPath, size: info.Size()}
			return fn(path, fi, err)
		}
	})
}

func (f *fsStorage) Read(path string, etag string) ([]byte, error) {
	return ioutil.ReadFile(f.fullPath(path))
}

func (f *fsStorage) Delete(path string) error {
	return os.Remove(f.fullPath(path))
}

func (f *fsStorage) Move(path, newPath string) error {
	path = f.fullPath(path)
	newPath = f.backupPath(newPath)
	os.MkdirAll(filepath.Dir(newPath), 0700)
	return os.Rename(path, newPath)
}

func (f *fsStorage) Info() {
}

func newFilesystemStorage(config *distributionStorageFilesystem) (storageObject, error) {
	return &fsStorage{config}, nil
}
