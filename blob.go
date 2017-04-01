package main

import "path/filepath"

type blobData struct {
	name       digest
	size       int64
	references int64
	etag       string
}

func (b *blobData) path() string {
	return filepath.Join("blobs", b.name.scopedPath(), "data")
}
