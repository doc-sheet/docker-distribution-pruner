package main

import (
	"flag"
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
)

var deleteOldTagVersions = flag.Bool("delete-old-tag-versions", true, "Delete old tag versions")

type tagData struct {
	repository *repositoryData
	name       string
	current    digest
	versions   []digest
	lock       sync.Mutex
}

func (t *tagData) currentLinkPath() string {
	return filepath.Join("repositories", t.repository.name, "_manifests", "tags", t.name, "current", "link")
}

func (t *tagData) versionLinkPath(version digest) string {
	return filepath.Join("repositories", t.repository.name, "_manifests", "tags", t.name, "index", version.path(), "link")
}

func (t *tagData) mark(blobs blobsData, deletes deletesData) error {
	if t.current.valid() {
		t.repository.markManifest(t.current)
	} else {
		deletes.schedule(t.currentLinkPath(), digestReferenceSize)
	}

	for _, version := range t.versions {
		if version != t.current {
			if *deleteOldTagVersions {
				deletes.schedule(t.versionLinkPath(version), digestReferenceSize)
			} else {
				t.repository.markManifest(version)
			}
		}
	}

	return nil
}

func (t *tagData) setCurrent(info fileInfo) error {
	//INFO[0000] /test2/_manifests/tags/latest/current/link

	link, err := readLink(t.currentLinkPath(), info.etag)
	if err != nil {
		return err
	}

	t.current = link
	logrus.Infoln("TAG:", t.repository.name, ":", t.name, ": is using:", t.current)
	return nil
}

func (t *tagData) addVersion(args []string, info fileInfo) error {
	//INFO[0000] /test2/_manifests/tags/latest/index/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link

	link, err := analyzeLink(args)
	if err != nil {
		return err
	}

	err = verifyLink(link, t.versionLinkPath(link), info.etag)
	if err != nil {
		return err
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	t.versions = append(t.versions, link)
	return nil
}
