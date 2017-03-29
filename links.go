package main

import (
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
