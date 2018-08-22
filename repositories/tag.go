package repositories

import (
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/docker-distribution-pruner/blobs"
	"gitlab.com/gitlab-org/docker-distribution-pruner/digest"
	"gitlab.com/gitlab-org/docker-distribution-pruner/storage"
)

type Tag struct {
	repository *Repository
	Name       string
	Current    digest.Digest
	Versions   []TagVersion
	lock       sync.Mutex
}

func (t *Tag) Path() string {
	return filepath.Join(t.repository.Path(), "_manifests", "tags", t.Name)
}

func (t *Tag) CurrentPath() string {
	return filepath.Join(t.Path(), "current", "link")
}

func (t *Tag) Mark(blobs *blobs.BlobList) error {
	for _, tagVersion := range t.versions {
		tagVersion.Mark(blobs)
	}
	return nil
}

func (t *Tag) Sweep(storageObject storage.StorageObject) error {
	if !t.Current.Valid() {
		err := storage.DeleteFile(storageObject, t.CurrentPath(), digest.DigestReferenceSize)
		if err != nil {
			return err
		}
	}

	for _, tagVersion := range t.versions {
		err := tagVersion.Sweep(storageObject)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Tag) setCurrent(info FileInfo) error {
	//INFO[0000] /test2/_manifests/tags/latest/current/link

	link, err := readLink(t.CurrentPath(), info.etag)
	if err != nil {
		return err
	}

	t.Current = link
	logrus.Infoln("TAG:", t.Repository.Name, ":", t.Name, ": is using:", t.Current)
	return nil
}

func (t *Tag) addVersion(args []string, info FileInfo) error {
	//INFO[0000] /test2/_manifests/tags/latest/index/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link

	link, err := analyzeLink(args)
	if err != nil {
		return err
	}

	tagVersion := TagVersion{Tag: t, Version: link}

	err = verifyLink(link, tagVersion.Path(), info.etag)
	if err != nil {
		return err
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	t.Versions = append(t.Versions, tagVersion)
	return nil
}
