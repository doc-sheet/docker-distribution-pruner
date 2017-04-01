package main

import (
	"path/filepath"
	"time"
)

type fileInfo struct {
	fullPath     string
	size         int64
	etag         string
	lastModified time.Time
	directory    bool
}

type walkFunc func(path string, info fileInfo, err error) error

type storageObject interface {
	Walk(path string, basePath string, fn walkFunc) error
	List(path string, fn walkFunc) error
	Read(path string, etag string) ([]byte, error)
	Delete(path string) error
	Move(path, newPath string) error
	Info()
}

var currentStorage storageObject

func parallelWalk(rootPath string, fn func(string) error) error {
	pwg := parallelWalkRunner.group()

	err := currentStorage.List(rootPath, func(listPath string, info fileInfo, err error) error {
		if !info.directory {
			return nil
		}

		pwg.dispatch(func() error {
			walkPath := filepath.Join(rootPath, listPath)
			return fn(walkPath)
		})
		return nil
	})
	if err != nil {
		return err
	}

	return pwg.finish()
}
