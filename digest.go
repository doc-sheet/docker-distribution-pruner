package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
)

const digestAlgorithm = "sha256"
const digestReferenceAlgorithm = "sha256:"

var digestEmpty [sha256.Size]byte

const digestReferenceSize = int64(len(digestReferenceAlgorithm) + sha256.Size*2)

type digest struct {
	hash [sha256.Size]byte
}

func newDigestFromPath(components []string) (d digest, err error) {
	if len(components) != 2 {
		return digest{}, fmt.Errorf("digest components should contain exactly two items: %v", components)
	}

	if components[0] != digestAlgorithm {
		return digest{}, fmt.Errorf("only %v is supported: %v", digestAlgorithm, components[0])
	}

	err = d.decode([]byte(components[1]))
	return
}

func newDigestFromScopedPath(components []string) (d digest, err error) {
	if len(components) != 3 {
		return digest{}, fmt.Errorf("digest components should contain exactly three items: %v", components)
	}

	if components[0] != digestAlgorithm {
		return digest{}, fmt.Errorf("only %v is supported: %v", digestAlgorithm, components[0])
	}

	if components[1] != components[2][0:2] {
		return digest{}, fmt.Errorf("digest needs to be prefixed with %v: %v", components[2][0:2], components)
	}

	err = d.decode([]byte(components[2]))
	if err != nil {
		return
	}
	return
}

func newDigestFromReference(data []byte) (d digest, err error) {
	if !bytes.HasPrefix(data, []byte(digestReferenceAlgorithm)) {
		return digest{}, fmt.Errorf("digest reference should start with: %v, but was: %v", digestReferenceAlgorithm, data)
	}

	err = d.decode(data[len(digestReferenceAlgorithm):])
	return
}

func (d *digest) decode(data []byte) error {
	n, err := hex.Decode(d.hash[:], data)
	if err != nil {
		return err
	}

	if n != sha256.Size {
		return fmt.Errorf("component should be valid %v, but was: %v", digestAlgorithm, data)
	}

	return nil
}

func (d *digest) hexHash() string {
	return hex.EncodeToString(d.hash[:])
}

func (d *digest) path() string {
	return filepath.Join(digestAlgorithm, d.hexHash())
}

func (d *digest) scopedPath() string {
	hex := d.hexHash()
	return filepath.Join(digestAlgorithm, hex[0:2], hex)
}

func (d *digest) reference() []byte {
	return []byte(digestReferenceAlgorithm + d.hexHash())
}

func (d *digest) etag() string {
	md5sum := md5.Sum(d.reference())
	hex := hex.EncodeToString(md5sum[:])
	return "\"" + hex + "\""
}

func (d digest) String() string {
	return d.hexHash()
}

func (d *digest) valid() bool {
	return !bytes.Equal(d.hash[:], digestEmpty[:])
}
