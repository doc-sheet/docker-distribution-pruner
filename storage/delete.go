package storage

import (
	"path/filepath"
	"sync/atomic"

	"github.com/Sirupsen/logrus"
	humanize "github.com/dustin/go-humanize"
	"gitlab.com/gitlab-org/docker-distribution-pruner/flags"
)

var (
	deletedLinks    int32
	deletedBlobs    int32
	deletedOther    int32
	deletedBlobSize int64
)

func DeleteFile(storage StorageObject, path string, size int64) error {
	logrus.Infoln("DELETE", path, size)
	name := filepath.Base(path)
	if name == "link" {
		atomic.AddInt32(&deletedLinks, 1)
	} else if name == "data" {
		atomic.AddInt32(&deletedBlobs, 1)
	} else {
		atomic.AddInt32(&deletedOther, 1)
	}

	atomic.AddInt64(&deletedBlobSize, size)

	if !*flags.Delete {
		// Do not delete, only write
		return nil
	}

	if *flags.SoftDelete {
		return storage.Move(path, filepath.Join("backup", path))
	}

	return storage.Delete(path)
}

func DeletesInfo() {
	logrus.Warningln("DELETEABLE INFO:", deletedLinks, "links,",
		deletedBlobs, "blobs,",
		deletedOther, "other,",
		humanize.Bytes(uint64(deletedBlobSize)),
	)
}
