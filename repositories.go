package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/dustin/go-humanize"
)

type repositoryData struct {
	name               string
	layers             map[string]int
	manifests          map[string]int
	manifestSignatures map[string][]string
	tags               map[string]*tag
	uploads            map[string]int
	lock               sync.Mutex
}

type repositoriesData map[string]*repositoryData

var repositoriesLock sync.Mutex

var parallelRepositoryWalk = flag.Bool("parallel-repository-walk", true, "Allow to use parallel repository walker")

func newRepositoryData(name string) *repositoryData {
	return &repositoryData{
		name:               name,
		layers:             make(map[string]int),
		manifests:          make(map[string]int),
		manifestSignatures: make(map[string][]string),
		tags:               make(map[string]*tag),
		uploads:            make(map[string]int),
	}
}

func (r *repositoryData) layerLinkPath(layer string) string {
	return filepath.Join("repositories", r.name, "_layers", "sha256", layer, "link")
}

func (r *repositoryData) manifestRevisionPath(revision string) string {
	return filepath.Join("repositories", r.name, "_manifests", "revisions", "sha256", revision, "link")
}

func (r *repositoryData) manifestRevisionSignaturePath(revision, signature string) string {
	return filepath.Join("repositories", r.name, "_manifests", "revisions", "sha256", revision, "signatures", "sha256", signature, "link")
}

func (r *repositoryData) uploadPath(upload string) string {
	return filepath.Join("repositories", r.name, "_uploads", upload, "link")
}

func (r *repositoryData) tag(name string) *tag {
	r.lock.Lock()
	defer r.lock.Unlock()

	t := r.tags[name]
	if t == nil {
		t = &tag{
			repository: r,
			name:       name,
		}
		r.tags[name] = t
	}

	return t
}

func (r *repositoryData) markManifest(name string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.manifests[name]++
	return nil
}

func (r *repositoryData) markManifestLayers(blobs blobsData, name string) error {
	err := blobs.mark(name)
	if err != nil {
		return err
	}

	manifest, err := manifests.get(name, blobs)
	if err != nil {
		return err
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	for _, layer := range manifest.layers {
		_, ok := r.layers[layer]
		if !ok {
			return fmt.Errorf("layer %s not found reference from manifest %s", layer, name)
		}

		r.layers[layer]++
	}

	return nil
}

func (r *repositoryData) markManifestSignatures(deletes deletesData, blobs blobsData, name string, signatures []string) error {
	if r.manifests[name] > 0 {
		for _, signature := range signatures {
			blobs.mark(signature)
		}
	} else {
		for _, signature := range signatures {
			deletes.schedule(r.manifestRevisionSignaturePath(name, signature), linkFileSize)
		}
	}
	return nil
}

func (r *repositoryData) markLayer(blobs blobsData, name string) error {
	return blobs.mark(name)
}

func (r *repositoryData) mark(blobs blobsData, deletes deletesData) error {
	for _, t := range r.tags {
		err := t.mark(blobs, deletes)
		if err != nil {
			return err
		}
	}

	for name_, used := range r.manifests {
		name := name_
		if used > 0 {
			err := r.markManifestLayers(blobs, name)
			if err != nil {
				return err
			}
		} else {
			deletes.schedule(r.manifestRevisionPath(name), linkFileSize)
		}
	}

	for name, signatures := range r.manifestSignatures {
		err := r.markManifestSignatures(deletes, blobs, name, signatures)
		if err != nil {
			return err
		}
	}

	for name_, used := range r.layers {
		name := name_
		if used > 0 {
			err := r.markLayer(blobs, name)
			if err != nil {
				return err
			}
		} else {
			deletes.schedule(r.layerLinkPath(name), linkFileSize)
		}
	}

	return nil
}

func (r repositoriesData) get(path []string) *repositoryData {
	repositoryName := strings.Join(path, "/")

	repositoriesLock.Lock()
	defer repositoriesLock.Unlock()

	repository := r[repositoryName]
	if repository == nil {
		repository = newRepositoryData(repositoryName)
		r[repositoryName] = repository
	}

	return repository
}

func (r *repositoryData) addLayer(args []string, info fileInfo) error {
	// /test/_layers/sha256/579c7fc9b0d60a19706cd6c1573fec9a28fa758bfe1ece86a1e5c68ad6f4e9d1/link
	link, err := analyzeLink(args)
	if err != nil {
		return err
	}

	err = verifyLink(link, r.layerLinkPath(link), info.etag)
	if err != nil {
		return err
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	r.layers[link] = 0
	return nil
}

func (r *repositoryData) addManifestRevision(args []string, info fileInfo) error {
	// /test2/_manifests/revisions/sha256/708519982eae159899e908639f5fa22d23d247ad923f6e6ad6128894c5d497a0/link
	link, err := analyzeLink(args)
	if err == nil {
		err = verifyLink(link, r.manifestRevisionPath(link), info.etag)
		if err != nil {
			return err
		}

		r.lock.Lock()
		defer r.lock.Unlock()

		r.manifests[link] = 0
		return nil
	}

	link, signature, err := analyzeLinkSignature(args)
	if err == nil {
		err = verifyLink(link, r.manifestRevisionSignaturePath(link, signature), info.etag)
		if err != nil {
			return err
		}

		r.lock.Lock()
		defer r.lock.Unlock()

		r.manifestSignatures[link] = append(r.manifestSignatures[link], signature)
		return nil
	}
	return err
}

func (r *repositoryData) addTag(args []string, info fileInfo) error {
	//INFO[0000] /test2/_manifests/tags/latest/current/link
	//INFO[0000] /test2/_manifests/tags/latest/index/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link

	tag := r.tag(args[0])
	if args[1] == "current" {
		return tag.setCurrent(info)
	} else if args[1] == "index" {
		return tag.addVersion(args[2:], info)
	} else {
		return fmt.Errorf("undefined manifest tag type: %v", args[1])
	}
}

func (r *repositoryData) addManifest(args []string, info fileInfo) error {
	//INFO[0000] /test2/_manifests/revisions/sha256/708519982eae159899e908639f5fa22d23d247ad923f6e6ad6128894c5d497a0/link
	//INFO[0000] /test2/_manifests/revisions/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link
	//INFO[0000] /test2/_manifests/tags/latest/current/link
	//INFO[0000] /test2/_manifests/tags/latest/index/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link
	//INFO[0000] /test2/_manifests/tags/latest2/current/link
	//INFO[0000] /test2/_manifests/tags/latest2/index/sha256/708519982eae159899e908639f5fa22d23d247ad923f6e6ad6128894c5d497a0/link

	if args[0] == "revisions" {
		return r.addManifestRevision(args[1:], info)
	} else if args[0] == "tags" {
		return r.addTag(args[1:], info)
	} else {
		return fmt.Errorf("undefined manifest type: %v", args[0])
	}
}

func (r *repositoryData) addUpload(args []string, info fileInfo) error {
	// /test/_uploads/579c7fc9b0d60a19706cd6c1573fec9a28fa758bfe1ece86a1e5c68ad6f4e9d1
	if len(args) != 1 {
		// logrus.Warningln("invalid args for uploads: %v", args)
		return nil
	}

	r.uploads[args[0]] = 1
	return nil
}

func (r *repositoryData) info(blobs blobsData) {
	var layersUsed, layersUnused int
	var manifestsUsed, manifestsUnused int
	var tagsVersions int
	var layersUsedSize, layersUnusedSize int64

	for name, used := range r.layers {
		blob := blobs[name]
		if blob == nil {
			continue
		}
		if used > 0 {
			layersUsed++
			layersUsedSize += blob.size
		} else {
			layersUnused++
			layersUnusedSize += blob.size
		}
	}

	for _, used := range r.manifests {
		if used > 0 {
			manifestsUsed++
		} else {
			manifestsUnused++
		}
	}

	for _, tag := range r.tags {
		tagsVersions += len(tag.versions)
	}

	logrus.Println("REPOSITORY INFO:", r.name, ":",
		"Tags/Versions:", len(r.tags), "/", tagsVersions,
		"Manifests/Unused:", manifestsUsed, "/", manifestsUnused,
		"Layers/Unused:", layersUsed, "/", layersUnused,
		"Data/Unused:", humanize.Bytes(uint64(layersUsedSize)), "/", humanize.Bytes(uint64(layersUnusedSize)))
}

func (r repositoriesData) process(segments []string, info fileInfo) error {
	for idx := 0; idx < len(segments)-1; idx++ {
		repository := segments[0:idx]
		args := segments[idx+1:]

		switch segments[idx] {
		case "_layers":
			return r.get(repository).addLayer(args, info)

		case "_manifests":
			return r.get(repository).addManifest(args, info)

		case "_uploads":
			return r.get(repository).addUpload(args, info)
		}
	}

	return fmt.Errorf("unparseable path: %v", segments)
}

func (r repositoriesData) walkPath(walkPath string, jg *jobsGroup) error {
	logrus.Infoln("REPOSITORIES DIR:", walkPath)
	return currentStorage.Walk(walkPath, "repositories", func(path string, info fileInfo, err error) error {
		jg.Dispatch(func() error {
			err = r.process(strings.Split(path, "/"), info)
			if err != nil {
				logrus.Infoln("REPOSITORY:", path, ":", err)
				return err
			}
			return nil
		})
		return nil
	})
}

func (r repositoriesData) walk() error {
	logrus.Infoln("Walking REPOSITORIES...")

	jg := jobsRunner.group()

	if *parallelRepositoryWalk {
		err := parallelWalk("repositories", func(listPath string) error {
			return r.walkPath(listPath, jg)
		})
		if err != nil {
			return err
		}
	} else {
		err := r.walkPath("repositories", jg)
		if err != nil {
			return err
		}
	}

	return jg.Finish()
}

func (r repositoriesData) mark(blobs blobsData, deletes deletesData) error {
	jg := jobsRunner.group()

	for _, repository_ := range r {
		repository := repository_
		jg.Dispatch(func() error {
			return repository.mark(blobs, deletes)
		})
	}

	err := jg.Finish()
	if err != nil {
		return err
	}
	return nil
}

func (r repositoriesData) info(blobs blobsData) {
	for _, repository := range r {
		repository.info(blobs)
	}
}
