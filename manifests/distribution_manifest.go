package manifests

import (
	"encoding/json"
	"fmt"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
)

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
