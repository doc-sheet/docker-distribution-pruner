package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/dustin/go-humanize"
)

var concurrentBlobAccess = flag.Bool("concurrent-blob-access", false, "Allow to use concurrent blob access")

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

	blob := &blobData{
		name: name,
		size: info.size,
		etag: info.etag,
	}
	b[name] = blob
	return nil
}

func (b blobsData) walk() error {
	logrus.Infoln("Walking BLOBS...")
	jg := jobsRunner.group()

	if *concurrentBlobAccess {
		listRootPath := filepath.Join("blobs", "sha256")
		err := currentStorage.List(listRootPath, func(listPath string, info fileInfo, err error) error {
			if !info.directory {
				return nil
			}

			jg.Dispatch(func() error {
				walkPath := filepath.Join(listRootPath, listPath)
				logrus.Infoln("BLOB DIR:", walkPath)
				return currentStorage.Walk(walkPath, "blobs", func(path string, info fileInfo, err error) error {
					err = b.addBlob(strings.Split(path, "/"), info)
					logrus.Infoln("BLOB:", path, ":", err)
					return err
				})
			})
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		err := currentStorage.Walk("blobs", "blobs", func(path string, info fileInfo, err error) error {
			if path == "" || info.directory {
				return nil
			}
			err = b.addBlob(strings.Split(path, "/"), info)
			logrus.Infoln("BLOB:", path, ":", err)
			return err
		})
		if err != nil {
			return err
		}
	}
	return jg.Finish()
}

func (b blobsData) info() {
	var blobsUsed, blobsUnused int
	var blobsUsedSize, blobsUnusedSize int64

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
