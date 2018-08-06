package storage

import (
	"errors"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type DistributionStorageFilesystem struct {
	RootDirectory string `yaml:"rootdirectory"`
}

type DistributionStorageS3 struct {
	AccessKey      string  `yaml:"accesskey"`
	SecretKey      string  `yaml:"secretkey"`
	Bucket         string  `yaml:"bucket"`
	Region         *string `yaml:"region"`
	RegionEndpoint *string `yaml:"regionendpoint"`
	RootDirectory  string  `yaml:"rootdirectory"`
}

type DistributionStorage struct {
	Filesystem *DistributionStorageFilesystem `yaml:"filesystem"`
	S3         *DistributionStorageS3         `yaml:"s3"`
}

type DistributionConfig struct {
	Version string              `yaml:"version"`
	Storage DistributionStorage `yaml:"storage"`
}

func StorageFromConfig(configFile string) (StorageObject, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	config := &DistributionConfig{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	if config.Version != "0.1" {
		return nil, errors.New("only 0.1 version is supported")
	}

	if config.Storage.Filesystem != nil && config.Storage.S3 != nil {
		return nil, errors.New("multiple storages defined")
	}

	if config.Storage.Filesystem != nil {
		return newFilesystemStorage(config.Storage.Filesystem)
	} else if config.Storage.S3 != nil {
		return newS3Storage(config.Storage.S3)
	} else {
		return nil, errors.New("unsupported storage")
	}
}
