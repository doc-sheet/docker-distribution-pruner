package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const listMax = 1000

type s3Storage struct {
	S3         *s3.S3
	apiCalls   int64
	cacheHits  int64
	cacheError int64
	cacheMiss  int64
}

var s3RootDir = flag.String("s3-root-dir", "", "s3 root directory")
var s3Bucket = flag.String("s3-bucket", "", "s3 bucket")
var s3Region = flag.String("s3-region", "us-east-1", "s3 region")
var s3CacheStorage = flag.String("s3-storage-cache", "tmp-cache", "s3 cache")

func newS3Storage() storageObject {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return &s3Storage{
		S3: s3.New(sess, aws.NewConfig().WithRegion(*s3Region)),
	}
}

func (f *s3Storage) fullPath(path string) string {
	return filepath.Join(*s3RootDir, "docker", "registry", "v2", path)
}

func (f *s3Storage) Walk(path string, baseDir string, fn walkFunc) error {
	path = f.fullPath(path)
	if path != "/" && path[len(path)-1] != '/' {
		path = path + "/"
	}

	baseDir = f.fullPath(baseDir)
	if baseDir != "/" && baseDir[len(baseDir)-1] != '/' {
		baseDir = baseDir + "/"
	}

	atomic.AddInt64(&f.apiCalls, 1)
	resp, err := f.S3.ListObjects(&s3.ListObjectsInput{
		Bucket:  s3Bucket,
		Prefix:  aws.String(path),
		MaxKeys: aws.Int64(listMax),
	})
	if err != nil {
		return err
	}

	for {
		lastKey := ""

		for _, key := range resp.Contents {
			lastKey = *key.Key
			keyPath := *key.Key
			if strings.HasPrefix(keyPath, baseDir) {
				keyPath = keyPath[len(baseDir):]
			}

			fi := fileInfo{
				fullPath:     *key.Key,
				size:         *key.Size,
				etag:         *key.ETag,
				lastModified: *key.LastModified,
			}

			err = fn(keyPath, fi, err)
			if err != nil {
				return err
			}
		}

		if *resp.IsTruncated {
			atomic.AddInt64(&f.apiCalls, 1)
			resp, err = f.S3.ListObjects(&s3.ListObjectsInput{
				Bucket:  s3Bucket,
				Prefix:  aws.String(path),
				MaxKeys: aws.Int64(listMax),
				Marker:  aws.String(lastKey),
			})
			if err != nil {
				return err
			}
		} else {
			break
		}
	}

	return nil
}

func (f *s3Storage) List(path string, fn walkFunc) error {
	path = f.fullPath(path)
	if path != "/" && path[len(path)-1] != '/' {
		path = path + "/"
	}

	atomic.AddInt64(&f.apiCalls, 1)
	resp, err := f.S3.ListObjects(&s3.ListObjectsInput{
		Bucket:    s3Bucket,
		Prefix:    aws.String(path),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int64(listMax),
	})
	if err != nil {
		return err
	}

	for {
		for _, key := range resp.Contents {
			keyPath := *key.Key
			if strings.HasPrefix(keyPath, path) {
				keyPath = keyPath[len(path):]
			}

			fi := fileInfo{
				fullPath:     *key.Key,
				size:         *key.Size,
				etag:         *key.ETag,
				lastModified: *key.LastModified,
				directory:    strings.HasSuffix(*key.Key, "/"),
			}

			err = fn(keyPath, fi, err)
			if err != nil {
				return err
			}
		}

		if *resp.IsTruncated {
			atomic.AddInt64(&f.apiCalls, 1)
			resp, err = f.S3.ListObjects(&s3.ListObjectsInput{
				Bucket:    s3Bucket,
				Prefix:    aws.String(path),
				MaxKeys:   aws.Int64(listMax),
				Delimiter: aws.String("/"),
				Marker:    resp.NextMarker,
			})
			if err != nil {
				return err
			}
		} else {
			break
		}
	}

	return nil
}

func (f *s3Storage) Read(path string, etag string) ([]byte, error) {
	cachePath := filepath.Join(*s3CacheStorage, path)
	if etag != "" && *s3CacheStorage != "" {
		file, err := ioutil.ReadFile(cachePath)
		if err == nil {
			if compareEtag(file, etag) {
				atomic.AddInt64(&f.cacheHits, 1)
				return file, nil
			} else {
				atomic.AddInt64(&f.cacheError, 1)
			}
		} else if os.IsNotExist(err) {
			atomic.AddInt64(&f.cacheMiss, 1)
		}
	}

	atomic.AddInt64(&f.apiCalls, 1)
	resp, err := f.S3.GetObject(&s3.GetObjectInput{
		Bucket: s3Bucket,
		Key:    aws.String(f.fullPath(path)),
	})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if etag != "" && *s3CacheStorage != "" {
		os.MkdirAll(filepath.Dir(cachePath), 0700)
		ioutil.WriteFile(cachePath, data, 0600)
	}

	return data, nil
}

func (f *s3Storage) Delete(path string) error {
	return errors.New("not supported")
}

func (f *s3Storage) Info() {
	logrus.Infoln("S3 INFO: API calls:", f.apiCalls,
		"Cache (hit/miss/error):", f.cacheHits, f.cacheMiss, f.cacheError)
}
