package storage

import (
	"path/filepath"

	"gitlab.com/gitlab-org/docker-distribution-pruner/concurrency"
)

var ParallelWalkRunner = make(concurrency.JobsData)

func ParallelWalk(storage StorageObject, rootPath string, fn func(string) error) error {
	pwg := ParallelWalkRunner.Group()

	err := storage.List(rootPath, func(listPath string, info FileInfo, err error) error {
		if !info.Directory {
			return nil
		}

		pwg.Dispatch(func() error {
			walkPath := filepath.Join(rootPath, listPath)
			return fn(walkPath)
		})
		return nil
	})
	if err != nil {
		return err
	}

	return pwg.Finish()
}
