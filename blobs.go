package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/dustin/go-humanize"
)

var parallelBlobWalk = flag.Bool("parallel-blob-walk", true, "Allow to use parallel blob walker")

type blobData struct {
	name       string
	size       int64
	references int64
	etag       string
}

func (b *blobData) path() string {
	return filepath.Join("blobs", "sha256", b.name[0:2], b.name, "data")
}

type blobsData map[string]*blobData

var blobsLock sync.Mutex

func (b blobsData) mark(name string) error {
	blobsLock.Lock()
	defer blobsLock.Unlock()

	if *ignoreBlobs {
		return nil
	}

	blob := b[name]
	if blob == nil {
		return fmt.Errorf("blob not found: %v", name)
	}
	blob.references++
	return nil
}

func (b blobsData) etag(name string) string {
	blob := b[name]
	if blob != nil {
		return blob.etag
	}
	return ""
}

func (b blobsData) size(name string) int64 {
	blob := b[name]
	if blob != nil {
		return blob.size
	}
	return 0
}

func (b blobsData) sweep(deletes deletesData) {
	for _, blob := range b {
		if blob.references == 0 {
			deletes.schedule(blob.path(), blob.size)
		}
	}
}

func (b blobsData) addBlob(segments []string, info fileInfo) error {
	if len(segments) != 4 {
		return fmt.Errorf("unparseable path: %v", segments)
	}

	if segments[0] != "sha256" {
		return fmt.Errorf("path needs to start with sha256: %v", segments)
	}

	if segments[3] != "data" {
		return fmt.Errorf("file needs to be data: %v", segments)
	}

	name := segments[2]
	if len(name) != 64 {
		return fmt.Errorf("blobs need to be sha256: %v", segments)
	}

	if segments[1] != name[0:2] {
		return fmt.Errorf("path needs to be prefixed with %v: %v", name[0:2], segments)
	}

	blobsLock.Lock()
	defer blobsLock.Unlock()

	blob := &blobData{
		name: name,
		size: info.size,
		etag: info.etag,
	}
	b[name] = blob
	return nil
}

func (b blobsData) walkPath(walkPath string) error {
	logrus.Infoln("BLOBS DIR:", walkPath)
	return currentStorage.Walk(walkPath, "blobs", func(path string, info fileInfo, err error) error {
		err = b.addBlob(strings.Split(path, "/"), info)
		if err != nil {
			logrus.Errorln("BLOB:", path, ":", err)
			if *softErrors {
				return nil
			}
		} else {
			logrus.Infoln("BLOB:", path, ":", err)
		}
		return err
	})
}

func (b blobsData) walk() error {
	logrus.Infoln("Walking BLOBS...")

	if *parallelBlobWalk {
		listRootPath := filepath.Join("blobs", "sha256")
		return parallelWalk(listRootPath, b.walkPath)
	} else {
		return b.walkPath("blobs")
	}
}

func (b blobsData) info() {
	var blobsUsed, blobsUnused int
	var blobsUsedSize, blobsUnusedSize int64

	if *ignoreBlobs {
		return
	}

	for _, blob := range b {
		if blob.references > 0 {
			blobsUsed++
			blobsUsedSize += blob.size
		} else {
			blobsUnused++
			blobsUnusedSize += blob.size
		}
	}

	logrus.Infoln("BLOBS INFO:",
		"Objects/Unused:", blobsUsed, "/", blobsUnused,
		"Data/Unused:", humanize.Bytes(uint64(blobsUsedSize)), "/", humanize.Bytes(uint64(blobsUnusedSize)),
	)
}
