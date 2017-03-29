package main

import (
	"flag"
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
)

var deleteOldTagVersions = flag.Bool("delete-old-tag-versions", true, "Delete old tag versions")

type tag struct {
	repository *repositoryData
	name       string
	current    string
	versions   []string
	lock       sync.Mutex
}

func (t *tag) currentLinkPath() string {
	return filepath.Join("repositories", t.repository.name, "_manifests", "tags", t.name, "current", "link")
}

func (t *tag) versionLinkPath(version string) string {
	return filepath.Join("repositories", t.repository.name, "_manifests", "tags", t.name, "index", "sha256", version, "link")
}

func (t *tag) mark(blobs blobsData, deletes deletesData) error {
	if t.current != "" {
		t.repository.markManifest(t.current)
	}

	for _, version := range t.versions {
		if version != t.current {
			if *deleteOldTagVersions {
				deletes.schedule(t.versionLinkPath(version), linkFileSize)
			} else {
				t.repository.markManifest(version)
			}
		}
	}

	return nil
}

func (t *tag) setCurrent(info fileInfo) error {
	//INFO[0000] /test2/_manifests/tags/latest/current/link

	readLink, err := readLink(t.currentLinkPath(), info.etag)
	if err != nil {
		return err
	}

	t.current = readLink
	logrus.Infoln("TAG:", t.repository.name, ":", t.name, ": is using:", t.current)
	return nil
}

func (t *tag) addVersion(args []string, info fileInfo) error {
	//INFO[0000] /test2/_manifests/tags/latest/index/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link

	link, err := analyzeLink(args)
	if err != nil {
		return err
	}

	err = verifyLink(link, info)
	if err != nil {
		return err
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	t.versions = append(t.versions, link)
	return nil
}
