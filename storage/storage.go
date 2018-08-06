package storage

import (
	"time"
)

type FileInfo struct {
	FullPath     string
	Size         int64
	Etag         string
	LastModified time.Time
	Directory    bool
}

type WalkFunc func(path string, info FileInfo, err error) error

type StorageObject interface {
	Walk(path string, basePath string, fn WalkFunc) error
	List(path string, fn WalkFunc) error
	Read(path string, etag string) ([]byte, error)
	Delete(path string) error
	Move(path, newPath string) error
	Info()
}
