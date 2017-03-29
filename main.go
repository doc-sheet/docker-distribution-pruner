package main

import (
	"flag"
	"os"

	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
)

var (
	debug      = flag.Bool("debug", false, "Print debug messages")
	verbose    = flag.Bool("verbose", true, "Print verbose messages")
	dryRun     = flag.Bool("dry-run", true, "Dry run")
	rootDir    = flag.String("root-dir", "examples/registry/docker/registry", "root directory")
)

var (
	deletedLinks    int
	deletedBlobs    int
	deletedBlobSize int64
)

const linkFileSize = int64(len("sha256:") + 64)

type tag struct {
	repository *repository
	name       string
	current    string
	versions   []string
}

func (t *tag) currentLinkPath() string {
	return filepath.Join(repositoriesPath, t.repository.name, "_manifests", "tags", t.name, "current", "link")
}

func (t *tag) versionLinkPath(version string) string {
	return filepath.Join(repositoriesPath, t.repository.name, "_manifests", "tags", t.name, "index", "sha256", version, "link")
}

type manifestData struct {
	name   string
	layers []string
}

type repository struct {
	name      string
	layers    map[string]int
	manifests map[string]int
	tags      map[string]*tag
	uploads   map[string]int
}

func (r *repository) layerLinkPath(layer string) string {
	return filepath.Join(repositoriesPath, r.name, "_layers", "sha256", layer, "link")
}

func (r *repository) manifestRevisionPath(revision string) string {
	return filepath.Join(repositoriesPath, r.name, "_manifests", "revisions", "sha256", revision, "link")
}

func (r *repository) uploadPath(upload string) string {
	return filepath.Join(repositoriesPath, r.name, "_uploads", upload, "link")
}

type blobData struct {
	name       string
	size       int64
	references int64
}

func (b *blobData) path() string {
	return filepath.Join(blobsPath, "sha256", b.name[0:2], b.name, "data")
}

var blobs map[string]*blobData = make(map[string]*blobData)
var manifests map[string]*manifestData = make(map[string]*manifestData)
var repositories map[string]*repository = make(map[string]*repository)
var deletes []string

var repositoriesPath string
var blobsPath string

func scheduleDelete(path string, size int64) {
	logrus.Infoln("DELETE", path, size)
	name := filepath.Base(path)
	if name == "link" {
		deletedLinks++
	} else if name == "data" {
		deletedBlobs++
	}
	deletedBlobSize += size
	deletes = append(deletes, path)
}

func getRepository(path []string) *repository {
	repositoryName := strings.Join(path, "/")

	r := repositories[repositoryName]
	if r == nil {
		r = &repository{
			name:      repositoryName,
			layers:    make(map[string]int),
			manifests: make(map[string]int),
			tags:      make(map[string]*tag),
			uploads:   make(map[string]int),
		}
		repositories[repositoryName] = r
	}

	return r
}

func (r *repository) getTag(name string) *tag {
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

func getManifest(name string) (*manifestData, error) {
	m := manifests[name]
	if m == nil {
		blobPath := filepath.Join(blobsPath, "sha256", name[0:2], name, "data")
		data, err := ioutil.ReadFile(blobPath)
		if err != nil {
			return nil, err
		}

		manifest, err := deserializeManifest(data)
		if err != nil {
			return nil, err
		}

		m = &manifestData{
			name: name,
		}

		for _, reference := range manifest.References() {
			m.layers = append(m.layers, reference.Digest.Hex())
		}

		manifests[name] = m
	}
	return m, nil
}

func analyzeLink(args []string) (string, error) {
	if len(args) != 3 {
		return "", fmt.Errorf("invalid args for link: %v", args)
	}

	if args[0] != "sha256" {
		return "", fmt.Errorf("only sha256 is supported: %v", args[0])
	}

	if args[2] != "link" {
		return "", fmt.Errorf("expected link as path component: %v", args[2])
	}

	return args[1], nil
}

func analyzeLayer(repository *repository, args []string) error {
	// /test/_layers/sha256/579c7fc9b0d60a19706cd6c1573fec9a28fa758bfe1ece86a1e5c68ad6f4e9d1/link
	link, err := analyzeLink(args)
	if err != nil {
		return err
	}

	readLink, err := readLink(repository.layerLinkPath(link))
	if err != nil {
		return err
	}

	if readLink != link {
		return fmt.Errorf("read link for %s is not equal %s", link, readLink)
	}

	repository.layers[link] = 0
	return nil
}

func analyzeManifestRevision(repository *repository, args []string) error {
	// /test2/_manifests/revisions/sha256/708519982eae159899e908639f5fa22d23d247ad923f6e6ad6128894c5d497a0/link
	link, err := analyzeLink(args)
	if err != nil {
		return err
	}

	readLink, err := readLink(repository.manifestRevisionPath(link))
	if err != nil {
		return err
	}

	if readLink != link {
		return fmt.Errorf("read link for %s is not equal %s", link, readLink)
	}

	repository.manifests[link] = 0
	return nil
}

func readLink(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", nil
	}

	link := string(data)
	if !strings.HasPrefix(link, "sha256:") {
		return "", errors.New("Link has to start with sha256")
	}

	link = link[len("sha256:"):]
	if len(link) != 64 {
		return "", fmt.Errorf("Link has to be exactly 256 bit: %v", link)
	}

	return link, nil
}

func analyzeManifestTagCurrent(tag *tag) error {
	//INFO[0000] /test2/_manifests/tags/latest/current/link

	readLink, err := readLink(tag.currentLinkPath())
	if err != nil {
		return err
	}

	tag.current = readLink
	return nil
}

func analyzeManifestTagVersion(tag *tag, args []string) error {
	//INFO[0000] /test2/_manifests/tags/latest/index/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link

	link, err := analyzeLink(args)
	if err != nil {
		return err
	}

	readLink, err := readLink(tag.versionLinkPath(link))
	if err != nil {
		return err
	}

	if readLink != link {
		return fmt.Errorf("read link for %s is not equal %s", link, readLink)
	}

	tag.versions = append(tag.versions, link)
	return nil
}

func analyzeManifestTag(repository *repository, args []string) error {
	//INFO[0000] /test2/_manifests/tags/latest/current/link
	//INFO[0000] /test2/_manifests/tags/latest/index/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link

	tag := repository.getTag(args[0])

	if args[1] == "current" {
		return analyzeManifestTagCurrent(tag)
	} else if args[1] == "index" {
		return analyzeManifestTagVersion(tag, args[2:])
	} else {
		return fmt.Errorf("undefined manifest tag type: %v", args[1])
	}
}

func analyzeManifest(repository *repository, args []string) error {
	//INFO[0000] /test2/_manifests/revisions/sha256/708519982eae159899e908639f5fa22d23d247ad923f6e6ad6128894c5d497a0/link
	//INFO[0000] /test2/_manifests/revisions/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link
	//INFO[0000] /test2/_manifests/tags/latest/current/link
	//INFO[0000] /test2/_manifests/tags/latest/index/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link
	//INFO[0000] /test2/_manifests/tags/latest2/current/link
	//INFO[0000] /test2/_manifests/tags/latest2/index/sha256/708519982eae159899e908639f5fa22d23d247ad923f6e6ad6128894c5d497a0/link

	if args[0] == "revisions" {
		return analyzeManifestRevision(repository, args[1:])
	} else if args[0] == "tags" {
		return analyzeManifestTag(repository, args[1:])
	} else {
		return fmt.Errorf("undefined manifest type: %v", args[0])
	}
}

func analyzeUploads(repository *repository, args []string) error {
	// /test/_uploads/579c7fc9b0d60a19706cd6c1573fec9a28fa758bfe1ece86a1e5c68ad6f4e9d1
	if len(args) != 1 {
		return fmt.Errorf("invalid args for uploads: %v", args)
	}

	repository.uploads[args[0]] = 1
	return nil
}

func analyzeRepositoryPath(segments []string) error {
	for idx := 0; idx < len(segments)-1; idx++ {
		repository := segments[0:idx]
		args := segments[idx+1:]

		switch segments[idx] {
		case "_layers":
			return analyzeLayer(getRepository(repository), args)

		case "_manifests":
			return analyzeManifest(getRepository(repository), args)

		case "_uploads":
			return analyzeUploads(getRepository(repository), args)
		}
	}

	return fmt.Errorf("unparseable path: %v", segments)
}

func walkRepository(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	if strings.HasPrefix(path, repositoriesPath) {
		path = path[len(repositoriesPath):]
	}

	err = analyzeRepositoryPath(strings.Split(path, "/"))
	logrus.Infoln("REPOSITORY:", path, ":", err)
	return err
}

func markBlob(name string) error {
	b := blobs[name]
	if b == nil {
		return fmt.Errorf("blob not found: %v", name)
	}
	b.references++
	return nil
}

func (t *tag) mark() error {
	if t.current != "" {
		t.repository.manifests[t.current]++
	}

	for _, version := range t.versions {
		if version != t.current {
			scheduleDelete(t.versionLinkPath(version), linkFileSize)
		}
	}

	return nil
}

func (r *repository) markManifest(name string) error {
	err := markBlob(name)
	if err != nil {
		return err
	}

	manifest, err := getManifest(name)
	if err != nil {
		return err
	}

	for _, layer := range manifest.layers {
		_, ok := r.layers[layer]
		if !ok {
			return fmt.Errorf("layer %s not found reference from manifest %s", layer, name)
		}

		r.layers[layer]++
	}

	return nil
}

func (r *repository) markLayer(name string) error {
	return markBlob(name)
}

func (r *repository) mark() error {
	for _, t := range r.tags {
		err := t.mark()
		if err != nil {
			return err
		}
	}

	for name, used := range r.manifests {
		if used > 0 {
			err := r.markManifest(name)
			if err != nil {
				return err
			}
		} else {
			scheduleDelete(r.manifestRevisionPath(name), linkFileSize)
		}
	}

	for name, used := range r.layers {
		if used > 0 {
			err := r.markLayer(name)
			if err != nil {
				return err
			}
		} else {
			scheduleDelete(r.layerLinkPath(name), linkFileSize)
		}
	}

	return nil
}

func analyzeBlobPath(segments []string, size int64) error {
	if len(segments) != 4 {
		return fmt.Errorf("unparseable path: %v", segments)
	}

	if segments[0] != "sha256" {
		return fmt.Errorf("path needs to start with sha256: %v", segments)
	}

	if segments[3] != "data" {
		return fmt.Errorf("file needs to be data: %v", segments)
	}

	blob := segments[2]
	if len(blob) != 64 {
		return fmt.Errorf("blobs need to be sha256: %v", segments)
	}

	if segments[1] != blob[0:2] {
		return fmt.Errorf("path needs to be prefixed with %v: %v", blob[0:2], segments)
	}

	b := &blobData{
		name: blob,
		size: size,
	}
	blobs[blob] = b

	return nil
}

func walkBlob(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	if strings.HasPrefix(path, blobsPath) {
		path = path[len(blobsPath):]
	}

	err = analyzeBlobPath(strings.Split(path, "/"), info.Size())
	logrus.Infoln("BLOB:", path, ":", err)
	return err
}

func main() {
	flag.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if *verbose {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	var err error

	repositoriesPath = filepath.Join(*rootDir, "v2", "repositories")
	repositoriesPath, err = filepath.Abs(repositoriesPath)
	if err != nil {
		logrus.Fatalln(err)
	}
	repositoriesPath += "/"

	blobsPath = filepath.Join(*rootDir, "v2", "blobs")
	blobsPath, err = filepath.Abs(blobsPath)
	if err != nil {
		logrus.Fatalln(err)
	}
	blobsPath += "/"

	logrus.Infoln("Walking BLOBS...")
	err = filepath.Walk(blobsPath, walkBlob)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Infoln("Walking REPOSITORIES...")
	err = filepath.Walk(repositoriesPath, walkRepository)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Infoln("Marking REPOSITORIES...")
	for _, r := range repositories {
		err := r.mark()
		if err != nil {
			logrus.Fatalln(err)
		}
	}

	logrus.Infoln("Marking BLOBS...")
	for _, b := range blobs {
		if b.references == 0 {
			scheduleDelete(b.path(), b.size)
		}
	}

	if !*dryRun {
		logrus.Infoln("Sweeping...")
		for _, path := range deletes {
			err := syscall.Unlink(path)
			if err != nil {
				logrus.Fatalln(err)
			}
		}
	}

	logrus.Warningln("Deleted:", deletedLinks, "links,",
		deletedBlobs, "blobs,",
		deletedBlobSize/1024/1024, "in MB",
	)
}
