package digest

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

type Digest struct {
	hash [sha256.Size]byte
}

func NewDigestFromPath(components []string) (d Digest, err error) {
	if len(components) != 2 {
		return Digest{}, fmt.Errorf("digest components should contain exactly two items: %v", components)
	}

	if components[0] != digestAlgorithm {
		return Digest{}, fmt.Errorf("only %v is supported: %v", digestAlgorithm, components[0])
	}

	err = d.decode([]byte(components[1]))
	return
}

func NewDigestFromScopedPath(components []string) (d Digest, err error) {
	if len(components) != 3 {
		return Digest{}, fmt.Errorf("digest components should contain exactly three items: %v", components)
	}

	if components[0] != digestAlgorithm {
		return Digest{}, fmt.Errorf("only %v is supported: %v", digestAlgorithm, components[0])
	}

	if components[1] != components[2][0:2] {
		return Digest{}, fmt.Errorf("digest needs to be prefixed with %v: %v", components[2][0:2], components)
	}

	err = d.decode([]byte(components[2]))
	if err != nil {
		return
	}
	return
}

func NewDigestFromReference(data []byte) (d Digest, err error) {
	if !bytes.HasPrefix(data, []byte(digestReferenceAlgorithm)) {
		return Digest{}, fmt.Errorf("digest reference should start with: %v, but was: %v", digestReferenceAlgorithm, data)
	}

	err = d.decode(data[len(digestReferenceAlgorithm):])
	return
}

func (d *Digest) decode(data []byte) error {
	n, err := hex.Decode(d.hash[:], data)
	if err != nil {
		return err
	}

	if n != sha256.Size {
		return fmt.Errorf("component should be valid %v, but was: %v", digestAlgorithm, data)
	}

	return nil
}

func (d *Digest) HexHash() string {
	return hex.EncodeToString(d.hash[:])
}

func (d *Digest) Path() string {
	return filepath.Join(digestAlgorithm, d.HexHash())
}

func (d *Digest) ScopedPath() string {
	hex := d.HexHash()
	return filepath.Join(digestAlgorithm, hex[0:2], hex)
}

func (d *Digest) Reference() []byte {
	return []byte(digestReferenceAlgorithm + d.HexHash())
}

func (d *Digest) Etag() string {
	md5sum := md5.Sum(d.Reference())
	hex := hex.EncodeToString(md5sum[:])
	return "\"" + hex + "\""
}

func (d *Digest) String() string {
	return d.HexHash()
}

func (d *Digest) Valid() bool {
	return !bytes.Equal(d.hash[:], digestEmpty[:])
}
