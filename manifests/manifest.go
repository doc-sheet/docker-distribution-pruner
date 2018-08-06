package manifests

import (
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/docker-distribution-pruner/blobs"
	"gitlab.com/gitlab-org/docker-distribution-pruner/digest"
)

type Manifest struct {
	Digest     digest.Digest
	Layers     []digest.Digest
	LoadLock   *sync.Once
	LoadStatus error
}

func (m *Manifest) Path() string {
	return filepath.Join("blobs", m.Digest.ScopedPath(), "data")
}

func (m *Manifest) Load(blobs *blobs.BlobList) {
	logrus.Println("MANIFEST:", m.Path(), ": loading...")

	data, err := blobs.ReadBlob(m.Digest)
	if err != nil {
		m.LoadStatus = err
		return
	}

	manifest, err := deserializeManifest(data)
	if err != nil {
		m.LoadStatus = err
		return
	}

	for _, reference := range manifest.References() {
		referenceDigest, err := digest.NewDigestFromReference([]byte(reference.Digest))
		if err != nil {
			m.LoadStatus = err
			return
		}
		m.Layers = append(m.Layers, referenceDigest)
	}

	m.LoadStatus = nil
	return
}

func (m *Manifest) LoadOnce(blobs *blobs.BlobList) error {
	// load manifest only once, recycle afterwards
	loadLock := m.LoadLock
	if loadLock != nil {
		loadLock.Do(func() {
			m.Load(blobs)
		})
		m.LoadLock = nil
	}

	return m.LoadStatus
}

func newManifest(digest digest.Digest) *Manifest {
	return &Manifest{
		Digest:   digest,
		LoadLock: &sync.Once{},
	}
}
