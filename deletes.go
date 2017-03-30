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
	deletedOther    int
	deletedBlobSize int64
)

type deletesData struct {
	files []string
}

var deletesLock sync.Mutex

func (d *deletesData) schedule(path string, size int64) {
	deletesLock.Lock()
	defer deletesLock.Unlock()

	logrus.Infoln("DELETE", path, size)
	name := filepath.Base(path)
	if name == "link" {
		deletedLinks++
	} else if name == "data" {
		deletedBlobs++
	} else {
		deletedOther++
	}
	deletedBlobSize += size
	d.files = append(d.files, path)
}

func (d *deletesData) info() {
	logrus.Warningln("DELETEABLE INFO:", deletedLinks, "links,",
		deletedBlobs, "blobs,",
		deletedOther, "other,",
		humanize.Bytes(uint64(deletedBlobSize)),
	)
}

func (d *deletesData) run(softDelete bool) {
	jg := jobsRunner.group()

	for _, path_ := range d.files {
		path := path_
		jg.Dispatch(func() error {
			if softDelete {
				err := currentStorage.Move(path, filepath.Join("backup", path))
				if err != nil {
					logrus.Fatalln(err)
				}
			} else {
				err := currentStorage.Delete(path)
				if err != nil {
					logrus.Fatalln(err)
				}
			}
			return nil
		})
	}

	jg.Finish()
}
