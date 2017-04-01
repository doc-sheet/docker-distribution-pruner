package main

import (
	"sync"
)

type manifestsData map[digest]*manifestData

var manifests manifestsData = make(map[digest]*manifestData)
var manifestsLock sync.Mutex

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

	return manifest, manifest.ensureLoaded(blobs)
}
