package storage

import (
	"crypto/md5"
	"encoding/hex"
)

func compareEtag(data []byte, etag string) bool {
	hash := md5.Sum(data)
	hex := hex.EncodeToString(hash[:])
	hex = "\"" + hex + "\""
	return etag == hex
}
