package main

import (
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

var (
	deletedLinks    int
	deletedBlobs    int
	deletedBlobSize int64
)

type deletesData []string

const linkFileSize = int64(len("sha256:") + 64)

func (d *deletesData) schedule(path string, size int64) {
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
	logrus.Warningln("Deleted:", deletedLinks, "links,",
		deletedBlobs, "blobs,",
		deletedBlobSize/1024/1024, "in MB",
	)
}

func (d *deletesData) run() {
	for _, path := range *d {
		err := currentStorage.Delete(path)
		if err != nil {
			logrus.Fatalln(err)
		}
	}
}
