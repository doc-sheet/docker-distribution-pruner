package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const listMax = 1000

type s3Storage struct {
	S3 *s3.S3
}

var s3RootDir = flag.String("s3-root-dir", "", "s3 root directory")
var s3Bucket = flag.String("s3-bucket", "", "s3 bucket")
var s3Region = flag.String("s3-region", "us-east-1", "s3 region")

func newS3Storage() *s3Storage {
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

func (f *s3Storage) Walk(path string, fn walkFunc) error {
	path = f.fullPath(path)
	if path != "/" && path[len(path)-1] != '/' {
		path = path + "/"
	}

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
			if strings.HasPrefix(keyPath, path) {
				keyPath = keyPath[len(path):]
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

func (f *s3Storage) Read(path string) ([]byte, error) {
	resp, err := f.S3.GetObject(&s3.GetObjectInput{
		Bucket: s3Bucket,
		Key:    aws.String(f.fullPath(path)),
	})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (f *s3Storage) Delete(path string) error {
	return errors.New("not supported")
}
