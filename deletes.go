package main

import (
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/dustin/go-humanize"
)

var (
	deletedLinks    int
	deletedBlobs    int
	deletedBlobSize int64
)

type deletesData []string

var deletesLock sync.Mutex

const linkFileSize = int64(len("sha256:") + 64)

func (d *deletesData) schedule(path string, size int64) {
	deletesLock.Lock()
	defer deletesLock.Unlock()

	logrus.Infoln("DELETE", path, size)
	name := filepath.Base(path)
	if name == "link" {
		deletedLinks++
	} else if name == "data" {
		deletedBlobs++
	}
	deletedBlobSize += size
	*d = append(*d, path)
}

func (d *deletesData) info() {
	logrus.Warningln("DELETEABLE INFO:", deletedLinks, "links,",
		deletedBlobs, "blobs,",
		humanize.Bytes(uint64(deletedBlobSize)),
	)
}

func (d *deletesData) run() {
	jg := jobsRunner.group()

	for _, path_ := range *d {
		path := path_
		jg.Dispatch(func() error {
			err := currentStorage.Delete(path)
			if err != nil {
				logrus.Fatalln(err)
			}
			return nil
		})
	}

	jg.Finish()
}
