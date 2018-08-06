package blobs

import (
	"path/filepath"

	"gitlab.com/gitlab-org/docker-distribution-pruner/digest"
)

type BlobData struct {
	Name       digest.Digest
	Size       int64
	References int64
	Etag       string
}

func (b *BlobData) Path() string {
	return filepath.Join("blobs", b.Name.ScopedPath(), "data")
}
