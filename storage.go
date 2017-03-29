package main

import "time"

type fileInfo struct {
	fullPath     string
	size         int64
	etag         string
	lastModified time.Time
}

type walkFunc func(path string, info fileInfo, err error) error

type storageObject interface {
	Walk(path string, fn walkFunc) error
	Read(path string) ([]byte, error)
	Delete(path string) error
}

var currentStorage storageObject
