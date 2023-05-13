package experimental

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const listMax = 1000

type s3Storage struct {
	*distributionStorageS3
	S3                *s3.S3
	apiCalls          int64
	expensiveApiCalls int64
	freeApiCalls      int64
	cacheHits         int64
	cacheError        int64
	cacheMiss         int64
}

var s3CacheStorage = flag.String("s3-storage-cache", "tmp-cache", "s3 cache")

func (f *s3Storage) fullPath(path string) string {
	return filepath.Join(f.RootDirectory, "docker", "registry", "v2", path)
}

func (f *s3Storage) backupPath(path string) string {
	return filepath.Join(f.RootDirectory, "docker-backup", "registry", "v2", path)
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
		Bucket:  aws.String(f.Bucket),
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

			if keyPath == "" {
				continue
			}

			if strings.HasSuffix(keyPath, "/") {
				logrus.Debugln("S3 Walk:", keyPath, "for", baseDir)
				continue
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
				Bucket:  aws.String(f.Bucket),
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
		Bucket:    aws.String(f.Bucket),
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

			if keyPath == "" {
				continue
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

		for _, commonPrefix := range resp.CommonPrefixes {
			prefixPath := *commonPrefix.Prefix
			if strings.HasPrefix(prefixPath, path) {
				prefixPath = prefixPath[len(path):]
			}

			if prefixPath == "" {
				continue
			}

			fi := fileInfo{
				fullPath:  *commonPrefix.Prefix,
				directory: true,
			}

			err = fn(prefixPath, fi, err)
			if err != nil {
				return err
			}
		}

		if *resp.IsTruncated {
			atomic.AddInt64(&f.apiCalls, 1)
			resp, err = f.S3.ListObjects(&s3.ListObjectsInput{
				Bucket:    aws.String(f.Bucket),
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
			logrus.Infoln("CACHE MISS:", path)
		}
	}

	atomic.AddInt64(&f.apiCalls, 1)
	resp, err := f.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(f.Bucket),
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
	atomic.AddInt64(&f.freeApiCalls, 1)
	_, err := f.S3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(f.Bucket),
		Key:    aws.String(f.fullPath(path)),
	})
	return err
}

func (f *s3Storage) Move(path, newPath string) error {
	atomic.AddInt64(&f.expensiveApiCalls, 1)
	_, err := f.S3.CopyObject(&s3.CopyObjectInput{
		CopySource: aws.String("/" + f.Bucket + "/" + f.fullPath(path)),
		Bucket:     aws.String(f.Bucket),
		Key:        aws.String(f.backupPath(newPath)),
	})
	if err != nil {
		return err
	}
	return f.Delete(path)
}

func (f *s3Storage) Info() {
	logrus.Infoln("S3 INFO: API calls/expensive/free:", f.apiCalls, f.expensiveApiCalls, f.freeApiCalls,
		"Cache (hit/miss/error):", f.cacheHits, f.cacheMiss, f.cacheError)
}

func newS3Storage(config *distributionStorageS3) (storageObject, error) {
	awsConfig := aws.NewConfig()
	awsConfig.Endpoint = config.RegionEndpoint
	awsConfig.Region = config.Region

	if config.AccessKey != "" && config.SecretKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, "")
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	storage := &s3Storage{
		distributionStorageS3: config,
		S3: s3.New(sess, awsConfig),
	}
	return storage, err
}
