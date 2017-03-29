package main

import (
	"fmt"
	"path/filepath"
)

type tag struct {
	repository *repositoryData
	name       string
	current    string
	versions   []string
}

func (t *tag) currentLinkPath() string {
	return filepath.Join("repositories", t.repository.name, "_manifests", "tags", t.name, "current", "link")
}

func (t *tag) versionLinkPath(version string) string {
	return filepath.Join("repositories", t.repository.name, "_manifests", "tags", t.name, "index", "sha256", version, "link")
}

func (t *tag) mark(blobs blobsData) error {
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

func (t *tag) setCurrent(info fileInfo) error {
	//INFO[0000] /test2/_manifests/tags/latest/current/link

	readLink, err := readLink(t.currentLinkPath())
	if err != nil {
		return err
	}

	t.current = readLink
	return nil
}

func (t *tag) addVersion(args []string, info fileInfo) error {
	//INFO[0000] /test2/_manifests/tags/latest/index/sha256/af8338145978acd626bfb9e863fa446bebfc9f2660bee1af99ed29efc48d73b4/link

	link, err := analyzeLink(args)
	if err != nil {
		return err
	}

	readLink, err := readLink(t.versionLinkPath(link))
	if err != nil {
		return err
	}

	if readLink != link {
		return fmt.Errorf("read link for %s is not equal %s", link, readLink)
	}

	t.versions = append(t.versions, link)
	return nil
}
