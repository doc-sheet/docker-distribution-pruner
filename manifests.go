package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
)

type manifestData struct {
	digest  digest
	layers  []digest
	loaded  bool
	loadErr error

	loadLock sync.Mutex
}

type manifestsData map[digest]*manifestData

var manifests manifestsData = make(map[digest]*manifestData)
var manifestsLock sync.Mutex

func deserializeManifest(data []byte) (distribution.Manifest, error) {
	var versioned manifest.Versioned
	if err := json.Unmarshal(data, &versioned); err != nil {
		return nil, err
	}

	switch versioned.SchemaVersion {
	case 1:
		var sm schema1.SignedManifest
		err := json.Unmarshal(data, &sm)
		return sm, err
	case 2:
		// This can be an image manifest or a manifest list
		switch versioned.MediaType {
		case schema2.MediaTypeManifest:
			var m schema2.DeserializedManifest
			err := json.Unmarshal(data, &m)
			return m, err
		case manifestlist.MediaTypeManifestList:
			var m manifestlist.DeserializedManifestList
			err := json.Unmarshal(data, &m)
			return m, err
		default:
			return nil, distribution.ErrManifestVerification{fmt.Errorf("unrecognized manifest content type %s", versioned.MediaType)}
		}
	}

	return nil, fmt.Errorf("unrecognized manifest schema version %d", versioned.SchemaVersion)
}

func (m *manifestData) path() string {
	return filepath.Join("blobs", m.digest.scopedPath(), "data")
}

func (m *manifestData) load(blobs blobsData) error {
	logrus.Println("MANIFEST:", m.path(), ": loading...")

	data, err := currentStorage.Read(m.path(), blobs.etag(m.digest))
	if err != nil {
		return err
	}

	manifest, err := deserializeManifest(data)
	if err != nil {
		return err
	}

	for _, reference := range manifest.References() {
		digest, err := newDigestFromReference([]byte(reference.Digest))
		if err != nil {
			return err
		}
		m.layers = append(m.layers, digest)
	}
	return nil
}

func (m manifestsData) get(digest digest, blobs blobsData) (*manifestData, error) {
	manifestsLock.Lock()
	manifest := m[digest]
	if manifest == nil {
		manifest = &manifestData{
			digest: digest,
		}
		m[digest] = manifest
	}
	manifestsLock.Unlock()

	if !manifest.loaded {
		manifest.loadLock.Lock()
		defer manifest.loadLock.Unlock()

		if !manifest.loaded {
			manifest.loadErr = manifest.load(blobs)
			manifest.loaded = true
		}
	}
	return manifest, manifest.loadErr
}
