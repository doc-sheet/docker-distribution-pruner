package main

type fileInfo struct {
	path string
	size int64
	etag string
}

type walkFunc func(path string, info fileInfo, err error) error

type storage interface {
	Walk(path string, fn walkFunc) error
	Read(path string) ([]byte, error)
	Delete(path string) error
}

var currentStorage storage
