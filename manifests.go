package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
)

type manifestData struct {
	name   string
	layers []string
}

type manifestsData map[string]*manifestData

var manifests manifestsData = make(map[string]*manifestData)

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
	return filepath.Join("blobs", "sha256", m.name[0:2], m.name, "data")
}

func (m *manifestData) load() error {
	data, err := currentStorage.Read(m.path())
	if err != nil {
		return err
	}

	manifest, err := deserializeManifest(data)
	if err != nil {
		return err
	}

	for _, reference := range manifest.References() {
		m.layers = append(m.layers, reference.Digest.Hex())
	}
	return nil
}

func (m manifestsData) get(name string) (*manifestData, error) {
	manifest := m[name]
	if manifest == nil {
		manifest = &manifestData{
			name: name,
		}

		err := manifest.load()
		if err != nil {
			return nil, err
		}
		m[name] = manifest
	}
	return manifest, nil
}
