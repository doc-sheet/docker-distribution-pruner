package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

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

func analyzeLinkSignature(args []string) (string, string, error) {
	// sha256/8d0c94a38dfa0db8827089a036d47482aa30550510d62f8fb2021548f49b1c84/signatures/sha256/6b659c9f4d1ff9c422f7bc517a0e896bc7fadb99a00e5db4c9921ddf8b5d402c/link

	if len(args) != 6 {
		return "", "", fmt.Errorf("invalid args for signature link: %v", args)
	}

	if args[0] != "sha256" || args[3] != "sha256" {
		return "", "", fmt.Errorf("only sha256 is supported: %v", args[0])
	}

	if args[2] != "signatures" {
		return "", "", fmt.Errorf("expected signatures as path component: %v", args[2])
	}

	if args[5] != "link" {
		return "", "", fmt.Errorf("expected link as path component: %v", args[2])
	}

	return args[1], args[4], nil
}

func compareEtag(data []byte, etag string) bool {
	hash := md5.Sum(data)
	hex := hex.EncodeToString(hash[:])
	hex = "\"" + hex + "\""
	return etag == hex
}

func readLink(path string, etag string) (string, error) {
	data, err := currentStorage.Read(path, etag)
	if err != nil {
		return "", err
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

func verifyLink(link string, path string, etag string) error {
	// If we have e-tag, let's verify e-tag
	if etag != "" {
		content := "sha256:" + link
		if compareEtag([]byte(content), etag) {
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
