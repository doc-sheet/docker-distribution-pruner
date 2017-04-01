package main

import (
	"errors"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type distributionStorageFilesystem struct {
	RootDirectory string `yaml:"rootdirectory"`
}

type distributionStorageS3 struct {
	AccessKey      string  `yaml:"accesskey"`
	SecretKey      string  `yaml:"secretkey"`
	Bucket         string  `yaml:"bucket"`
	Region         *string `yaml:"region"`
	RegionEndpoint *string `yaml:"regionendpoint"`
	RootDirectory  string  `yaml:"rootdirectory"`
}

type distributionStorage struct {
	Filesystem *distributionStorageFilesystem `yaml:"filesystem"`
	S3         *distributionStorageS3         `yaml:"s3"`
}

type distributionConfig struct {
	Version string              `yaml:"version"`
	Storage distributionStorage `yaml:"storage"`
}

func storageFromConfig(configFile string) (storageObject, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	config := &distributionConfig{}
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
