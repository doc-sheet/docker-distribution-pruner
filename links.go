package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
)

func analyzeLink(args []string) (digest, error) {
	if len(args) != 3 {
		return digest{}, fmt.Errorf("invalid args for link: %v", args)
	}

	if args[2] != "link" {
		return digest{}, fmt.Errorf("expected link as path component: %v", args[2])
	}

	return newDigestFromPath(args[0:2])
}

func analyzeLinkSignature(args []string) (digest, digest, error) {
	// sha256/8d0c94a38dfa0db8827089a036d47482aa30550510d62f8fb2021548f49b1c84/signatures/sha256/6b659c9f4d1ff9c422f7bc517a0e896bc7fadb99a00e5db4c9921ddf8b5d402c/link

	if len(args) != 6 {
		return digest{}, digest{}, fmt.Errorf("invalid args for signature link: %v", args)
	}

	if args[5] != "link" {
		return digest{}, digest{}, fmt.Errorf("expected link as path component: %v", args[2])
	}

	if args[2] != "signatures" {
		return digest{}, digest{}, fmt.Errorf("expected signatures as path component: %v", args[2])
	}

	link, err := newDigestFromPath(args[0:2])
	if err != nil {
		return digest{}, digest{}, err
	}

	signature, err := newDigestFromPath(args[3:5])
	if err != nil {
		return digest{}, digest{}, err
	}

	return link, signature, err
}

func compareEtag(data []byte, etag string) bool {
	hash := md5.Sum(data)
	hex := hex.EncodeToString(hash[:])
	hex = "\"" + hex + "\""
	return etag == hex
}

func readLink(path string, etag string) (digest, error) {
	data, err := currentStorage.Read(path, etag)
	if err != nil {
		return digest{}, err
	}

	d, err := newDigestFromReference(data)
	if err != nil {
		return digest{}, err
	}

	return d, nil
}

func verifyLink(link digest, path string, etag string) error {
	// If we have e-tag, let's verify e-tag
	if etag != "" {
		if link.etag() == etag {
			return nil
		}
	}

	readed, err := readLink(path, etag)
	if err != nil {
		return err
	}

	if readed != link {
		return fmt.Errorf("%s: readed link for %s is not equal %s", path, link, readed)
	}

	return nil
}
