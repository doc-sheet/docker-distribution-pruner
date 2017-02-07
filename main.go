package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/factory"
	"github.com/docker/libtrust"

	// Include all storage drivers
	_ "github.com/docker/distribution/registry/storage/driver/azure"
	_ "github.com/docker/distribution/registry/storage/driver/filesystem"
	_ "github.com/docker/distribution/registry/storage/driver/gcs"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	_ "github.com/docker/distribution/registry/storage/driver/middleware/cloudfront"
	_ "github.com/docker/distribution/registry/storage/driver/middleware/redirect"
	_ "github.com/docker/distribution/registry/storage/driver/oss"
	_ "github.com/docker/distribution/registry/storage/driver/s3-aws"
	_ "github.com/docker/distribution/registry/storage/driver/s3-goamz"
	_ "github.com/docker/distribution/registry/storage/driver/swift"
)

var configFile = flag.String("config", "", "Configuration file to use a storage settings from")
var deleteManifests = flag.Bool("delete-manifests", false, "Delete manifests that are unreferenced (repository level)")
var deleteBlobs = flag.Bool("delete-blobs", false, "Delete blobs that are unreferenced (repository level)")
var deleteGlobalBlobs = flag.Bool("delete-global-blobs", false, "Delete blobs from global storage that are unreferenced")

func resolveConfiguration() (*configuration.Configuration, error) {
	fp, err := os.Open(*configFile)
	if err != nil {
		return nil, err
	}

	defer fp.Close()

	config, err := configuration.Parse(fp)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", *configFile, err)
	}

	return config, nil
}

func markTags(ctx context.Context, repository distribution.Repository) (manifestsSet map[digest.Digest]struct{}, err error) {
	manifestsSet = make(map[digest.Digest]struct{})
	tagsService := repository.Tags(ctx)
	tags, err := tagsService.All(ctx)
	if err != nil {
		// In certain situations such as unfinished uploads, deleting all
		// tags in S3 or removing the _manifests folder manually, this
		// error may be of type PathNotFound.
		//
		// In these cases we can continue marking other manifests safely.
		if _, ok := err.(driver.PathNotFoundError); !ok {
			return nil, err
		}
	}

	for _, tagName := range tags {
		descriptor, err := tagsService.Get(ctx, tagName)
		if err != nil {
			// In certain situations such as unfinished uploads, deleting all
			// tags in S3 or removing the _manifests folder manually, this
			// error may be of type PathNotFound.
			//
			// In these cases we can continue marking other manifests safely.
			if _, ok := err.(driver.PathNotFoundError); !ok {
				return nil, err
			}
		}

		manifestsSet[descriptor.Digest] = struct{}{}
	}

	return nil, err
}

func markManifests(ctx context.Context, repository distribution.Repository, manifestsSet, globalBlobSet map[digest.Digest]struct{}) (manifestsDeleteSet, blobSet map[digest.Digest]struct{}, err error) {
	repoName := repository.Named().Name()
	manifestsDeleteSet = make(map[digest.Digest]struct{})
	blobSet = make(map[digest.Digest]struct{})

	manifestService, err := repository.Manifests(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to construct manifest service: %v", err)
	}

	manifestEnumerator, ok := manifestService.(distribution.ManifestEnumerator)
	if !ok {
		return nil, nil, fmt.Errorf("unable to convert ManifestService into ManifestEnumerator")
	}

	// swap repo phase
	err = manifestEnumerator.Enumerate(ctx, func(dgst digest.Digest) error {
		_, ok := manifestsSet[dgst]
		if ok {
			// Mark the manifest's blob
			logrus.Debugln("%s: marking manifest %s ", repoName, dgst)
			blobSet[dgst] = struct{}{}
			globalBlobSet[dgst] = struct{}{}

			manifest, err := manifestService.Get(ctx, dgst)
			if err != nil {
				return fmt.Errorf("failed to retrieve manifest for digest %v: %v", dgst, err)
			}

			descriptors := manifest.References()
			for _, descriptor := range descriptors {
				blobSet[descriptor.Digest] = struct{}{}
				globalBlobSet[descriptor.Digest] = struct{}{}
				logrus.Infoln("%s: marking blob %s", repoName, descriptor.Digest)
			}
		} else {
			manifestsDeleteSet[dgst] = struct{}{}
		}

		return nil
	})

	return manifestsDeleteSet, blobSet, err
}

func markBlobs(ctx context.Context, repository distribution.Repository, blobSet map[digest.Digest]struct{}) (blobDeleteSet map[digest.Digest]struct{}, err error) {
	blobDeleteSet = make(map[digest.Digest]struct{})

	blobSerivce := repository.Blobs(ctx)
	blobEnumerator, ok := blobSerivce.(distribution.BlobEnumerator)
	if !ok {
		return nil, fmt.Errorf("unable to convert BlobService into BlobEnumerator")
	}

	err = blobEnumerator.Enumerate(ctx, func(dgst digest.Digest) error {
		// check if digest is in markSet. If not, delete it!
		if _, ok := blobSet[dgst]; !ok {
			blobDeleteSet[dgst] = struct{}{}
		}
		return nil
	})

	return blobDeleteSet, err
}

func sweepManifests(ctx context.Context, repository distribution.Repository, manifestsDeleteSet map[digest.Digest]struct{}) error {
	repoName := repository.Named().Name()

	manifestService, err := repository.Manifests(ctx)
	if err != nil {
		return fmt.Errorf("failed to construct manifest service: %v", err)
	}

	for dgst := range manifestsDeleteSet {
		logrus.Infoln("%s: manifest eligible for deletion: %s", repoName, dgst)
		if *deleteManifests {
			continue
		}
		err = manifestService.Delete(ctx, dgst)
		if err != nil {
			return fmt.Errorf("failed to delete blob %s: %v", dgst, err)
		}
	}

	return nil
}

func sweepBlobs(ctx context.Context, repository distribution.Repository, blobsDeleteSet map[digest.Digest]struct{}) error {
	repoName := repository.Named().Name()
	blobsServise := repository.Blobs(ctx)

	for dgst := range blobsDeleteSet {
		logrus.Infoln("%s: manifest eligible for deletion: %s", repoName, dgst)
		if *deleteBlobs {
			continue
		}
		err := blobsServise.Delete(ctx, dgst)
		if err != nil {
			return fmt.Errorf("failed to delete blob %s: %v", dgst, err)
		}
	}

	return nil
}

func processRepository(ctx context.Context, repository distribution.Repository, globalBlobSet map[digest.Digest]struct{}) error {
	repoName := repository.Named().Name()

	// mark
	manifestsSet, err := markTags(ctx, repository)
	if err != nil {
		return fmt.Errorf("%s: unable to count manifests used by tags: %v", repoName, err)
	}

	manifestsDeleteSet, blobSet, err := markManifests(ctx, repository, manifestsSet, globalBlobSet)
	if err != nil {
		return fmt.Errorf("%s: unable to count blobs used by manifests: %v", repoName, err)
	}

	blobDeleteSet, err := markBlobs(ctx, repository, blobSet)
	if err != nil {
		return fmt.Errorf("%s: unable to list blobs to be deleted: %v", repoName, err)
	}

	logrus.Infoln("%s: %d manifests marked, %d manifests eligible for deletion", repoName, len(manifestsSet), len(manifestsDeleteSet))
	logrus.Infoln("%s: %d blobs marked, %d blobs eligible for deletion", repoName, len(blobSet), len(blobDeleteSet))

	err = sweepManifests(ctx, repository, manifestsDeleteSet)
	if err != nil {
		return fmt.Errorf("%s: unable to delete manifests: %v", repoName, err)
	}

	err = sweepBlobs(ctx, repository, blobDeleteSet)
	if err != nil {
		return fmt.Errorf("%s: unable to delete blobs: %v", repoName, err)
	}
	return nil
}

func sweepGlobalBlobs(ctx context.Context, storageDriver driver.StorageDriver, registry distribution.Namespace, markSet map[digest.Digest]struct{}) error {
	// sweep
	blobService := registry.Blobs()
	deleteSet := make(map[digest.Digest]struct{})
	err := blobService.Enumerate(ctx, func(dgst digest.Digest) error {
		// check if digest is in markSet. If not, delete it!
		if _, ok := markSet[dgst]; !ok {
			deleteSet[dgst] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error enumerating blobs: %v", err)
	}

	logrus.Infoln("%d blobs marked, %d blobs eligible for deletion", len(markSet), len(deleteSet))

	// Construct vacuum
	vacuum := storage.NewVacuum(ctx, storageDriver)
	for dgst := range deleteSet {
		logrus.Debugln("blob eligible for deletion: %s", dgst)
		if *deleteGlobalBlobs {
			continue
		}
		err := vacuum.RemoveBlob(string(dgst))
		if err != nil {
			return fmt.Errorf("failed to delete blob %s: %v", dgst, err)
		}
	}

	return nil
}

func processRegistry(ctx context.Context, storageDriver driver.StorageDriver, registry distribution.Namespace) error {
	markSet := make(map[digest.Digest]struct{})

	repositoryEnumerator, ok := registry.(distribution.RepositoryEnumerator)
	if !ok {
		return fmt.Errorf("unable to convert Namespace to RepositoryEnumerator")
	}

	err := repositoryEnumerator.Enumerate(ctx, func(repoName string) error {
		logrus.Debugln(repoName)

		var err error
		named, err := reference.ParseNamed(repoName)
		if err != nil {
			return fmt.Errorf("failed to parse repo name %s: %v", repoName, err)
		}
		repository, err := registry.Repository(ctx, named)
		if err != nil {
			return fmt.Errorf("failed to construct repository: %v", err)
		}

		return processRepository(ctx, repository, markSet)
	})

	if err != nil {
		return fmt.Errorf("failed to mark: %v", err)
	}

	err = sweepGlobalBlobs(ctx, storageDriver, registry, markSet)

	if err != nil {
		return fmt.Errorf("failed to sweep: %v", err)
	}
	return err
}

func main() {
	flag.Parse()

	config, err := resolveConfiguration()
	if err != nil {
		logrus.Println("configuration error: %v\n", err)
		flag.Usage()
		logrus.Fatalln()
	}

	driver, err := factory.Create(config.Storage.Type(), config.Storage.Parameters())
	if err != nil {
		logrus.Fatalln("failed to construct %s driver: %v", config.Storage.Type(), err)
	}

	ctx := context.Background()
	if err != nil {
		logrus.Fatalln("unable to configure logrusging with config: %s", err)
	}

	k, err := libtrust.GenerateECP256PrivateKey()
	if err != nil {
		logrus.Fatalln(err)
	}

	registry, err := storage.NewRegistry(ctx, driver, storage.Schema1SigningKey(k))
	if err != nil {
		logrus.Fatalln("failed to construct registry: %v", err)
	}

	err = processRegistry(ctx, driver, registry)
	if err != nil {
		logrus.Fatalln("failed to process registry: %v", err)
	}
}
