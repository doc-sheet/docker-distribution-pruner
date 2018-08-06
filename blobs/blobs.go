package blobs

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/dustin/go-humanize"

	"gitlab.com/gitlab-org/docker-distribution-pruner/concurrency"
	"gitlab.com/gitlab-org/docker-distribution-pruner/digest"
	"gitlab.com/gitlab-org/docker-distribution-pruner/flags"
	"gitlab.com/gitlab-org/docker-distribution-pruner/storage"
)

type BlobsData struct {
	blobs   map[digest.Digest]*BlobData
	lock    sync.RWMutex
	storage storage.StorageObject
}

func (b *BlobsData) Mark(digest digest.Digest) error {
	if *flags.IgnoreBlobs {
		return nil
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	blob := b.blobs[digest]
	if blob == nil {
		return fmt.Errorf("blob not found: %v", digest)
	}
	blob.References++
	return nil
}

func (b *BlobsData) Etag(digest digest.Digest) string {
	b.lock.RLock()
	defer b.lock.RUnlock()

	blob := b.blobs[digest]
	if blob != nil {
		return blob.Etag
	}
	return ""
}

func (b *BlobsData) Size(digest digest.Digest) int64 {
	b.lock.RLock()
	defer b.lock.RUnlock()

	blob := b.blobs[digest]
	if blob != nil {
		return blob.Size
	}
	return 0
}

func (b *BlobsData) sweepBlob(jg *concurrency.JobGroup, blob *BlobData) {
	jg.Dispatch(func() error {
		if blob.References > 0 {
			return nil
		}

		err := storage.DeleteFile(b.storage, blob.Path(), blob.Size)
		if err != nil {
			return err
		}
		return nil
	})
}

func (b *BlobsData) Sweep(jobs concurrency.JobsData) error {
	jg := jobs.Group()

	for _, blob := range b.blobs {
		b.sweepBlob(jg, blob)
	}

	return jg.Finish()
}

func (b *BlobsData) AddBlob(segments []string, info storage.FileInfo) error {
	if len(segments) != 4 {
		return fmt.Errorf("unparseable path: %v", segments)
	}

	if segments[3] != "data" {
		return fmt.Errorf("file needs to be data: %v", segments)
	}

	blobDigest, err := digest.NewDigestFromScopedPath(segments[0:3])
	if err != nil {
		return err
	}

	if segments[0] != "sha256" {
		return fmt.Errorf("path needs to start with sha256: %v", segments)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	blob := &BlobData{
		Name: blobDigest,
		Size: info.Size,
		Etag: info.Etag,
	}
	if b.blobs == nil {
		b.blobs = make(map[digest.Digest]*BlobData)
	}
	b.blobs[blobDigest] = blob
	return nil
}

func (b *BlobsData) walkPath(walkPath string) error {
	logrus.Infoln("BLOBS DIR:", walkPath)
	return b.storage.Walk(walkPath, "blobs", func(path string, info storage.FileInfo, err error) error {
		err = b.AddBlob(strings.Split(path, "/"), info)

		if err != nil {
			logrus.Errorln("BLOB:", path, ":", err)
			if *flags.SoftErrors {
				return nil
			}
		}

		return err
	})
}

func (b *BlobsData) Walk(parallel bool) error {
	logrus.Infoln("Walking BLOBS...")

	if parallel {
		listRootPath := filepath.Join("blobs", "sha256")
		return storage.ParallelWalk(b.storage, listRootPath, b.walkPath)
	}

	return b.walkPath("blobs")
}

func (b *BlobsData) Info() {
	var blobsUsed, blobsUnused int
	var blobsUsedSize, blobsUnusedSize int64

	if *flags.IgnoreBlobs {
		return
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	for _, blob := range b.blobs {
		if blob.References > 0 {
			blobsUsed++
			blobsUsedSize += blob.Size
		} else {
			blobsUnused++
			blobsUnusedSize += blob.Size
		}
	}

	logrus.Infoln("BLOBS INFO:",
		"Objects/Unused:", blobsUsed, "/", blobsUnused,
		"Data/Unused:", humanize.Bytes(uint64(blobsUsedSize)), "/", humanize.Bytes(uint64(blobsUnusedSize)),
	)
}
