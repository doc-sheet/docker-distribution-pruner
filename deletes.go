package main

import (
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"syscall"
)

var (
	deletedLinks    int
	deletedBlobs    int
	deletedBlobSize int64
	deletes         []string
)

const linkFileSize = int64(len("sha256:") + 64)

func scheduleDelete(path string, size int64) {
	logrus.Infoln("DELETE", path, size)
	name := filepath.Base(path)
	if name == "link" {
		deletedLinks++
	} else if name == "data" {
		deletedBlobs++
	}
	deletedBlobSize += size
	deletes = append(deletes, path)
}

func runDeletes() {
	for _, path := range deletes {
		err := syscall.Unlink(path)
		if err != nil {
			logrus.Fatalln(err)
		}
	}
}
