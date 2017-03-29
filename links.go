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

func readLink(path string) (string, error) {
	data, err := currentStorage.Read(path)
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

func verifyLink(link string, info fileInfo) error {
	// If we have e-tag, let's verify e-tag
	if info.etag != "" {
		content := "sha256:" + link
		hash := md5.Sum([]byte(content))
		hex := hex.EncodeToString(hash[:])
		hex = "\"" + hex + "\""
		if info.etag == hex {
			return nil
		}

		return fmt.Errorf("etag for %s is not equal %s to %s", link, info.etag, hex)
	} else {
		readed, err := readLink(info.fullPath)
		if err != nil {
			return err
		}

		if readed != link {
			return fmt.Errorf("readed link for %s is not equal %s", link, readed)
		}

		return nil
	}
}
