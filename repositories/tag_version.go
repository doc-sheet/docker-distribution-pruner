package repositories

import (
	"path/filepath"

	"gitlab.com/gitlab-org/docker-distribution-pruner/blobs"
	"gitlab.com/gitlab-org/docker-distribution-pruner/digest"
	"gitlab.com/gitlab-org/docker-distribution-pruner/flags"
	"gitlab.com/gitlab-org/docker-distribution-pruner/storage"
)

type TagVersion struct {
	tag     *Tag
	Version digest.Digest
}

func (t *TagVersion) IsCurrent() bool {
	return t.tag.Current == Version
}

func (t *TagVersion) Path() string {
	return filepath.Join(t.tag.Path(), "index", t.Version, "link")
}

func (t *TagVersion) Mark(blobs *blobs.BlobList) error {
	if !t.IsCurrent() && *flags.DeleteOldTagVersions {
		return nil
	}

	return t.Repository.MarkManifest(t.Version)
}

func (t *TagVersion) Sweep(storageObject storage.StorageObject) error {
	if !t.IsCurrent() && *flags.DeleteOldTagVersions {
		return nil
	}

	return storage.DeleteFile(storageObject, t.Path(), digest.DigestReferenceSize)
}
