package distribution

import (
	"github.com/docker/distribution/context"
	"github.com/opencontainers/go-digest"
)

// TagService provides access to information about tagged objects.
type TagService interface {
	// Get retrieves the descriptor identified by the tag. Some
	// implementations may differentiate between "trusted" tags and
	// "untrusted" tags. If a tag is "untrusted", the mapping will be returned
	// as an ErrTagUntrusted error, with the target descriptor.
	Get(ctx context.Context, tag string) (Descriptor, error)

	// Tag associates the tag with the provided descriptor, updating the
	// current association, if needed.
	Tag(ctx context.Context, tag string, desc Descriptor) error

	// Untag removes the given tag association
	Untag(ctx context.Context, tag string) error

	// All returns the set of tags managed by this tag service
	All(ctx context.Context) ([]string, error)

	// Lookup returns the set of tags referencing the given digest.
	Lookup(ctx context.Context, digest Descriptor) ([]string, error)

	Versions(ctx context.Context, tag string) (TagVersionsService, error)
}

type TagVersionsService interface {
	// Exists returns true if the manifest exists.
	Get(ctx context.Context, dgst digest.Digest) (Descriptor, error)

	// Delete removes the manifest specified by the given digest. Deleting
	// a manifest that doesn't exist will return ErrManifestNotFound
	Delete(ctx context.Context, dgst digest.Digest) error
}

// ManifestEnumerator enables iterating over manifests
type TagVersionsEnumerator interface {
	// Enumerate calls ingester for each manifest.
	Enumerate(ctx context.Context, ingester func(digest.Digest) error) error
}
