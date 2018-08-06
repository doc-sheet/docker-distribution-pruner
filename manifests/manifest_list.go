package manifests

import (
	"sync"

	"gitlab.com/gitlab-org/docker-distribution-pruner/blobs"
	"gitlab.com/gitlab-org/docker-distribution-pruner/digest"
)

type ManifestList struct {
	manifests map[digest.Digest]*Manifest
	lock      sync.Mutex
}

func (m *ManifestList) Get(manifestDigest digest.Digest) *Manifest {
	m.lock.Lock()
	defer m.lock.Unlock()

	manifest := m.manifests[manifestDigest]
	if manifest != nil {
		return manifest
	}

	manifest = newManifest(manifestDigest)
	if m.manifests == nil {
		m.manifests = make(map[digest.Digest]*Manifest)
	}
	m.manifests[manifestDigest] = manifest
	return manifest
}

func (m *ManifestList) Load(manifestDigest digest.Digest, blobs *blobs.BlobList) (*Manifest, error) {
	manifest := m.Get(manifestDigest)
	return manifest, manifest.LoadOnce(blobs)
}
