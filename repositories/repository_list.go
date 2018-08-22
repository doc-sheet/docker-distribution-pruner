package repositories

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
)

type repositoriesData map[string]*Repository

var repositoriesLock sync.Mutex

func (r repositoriesData) get(path []string) *Repository {
	repositoryName := strings.Join(path, "/")

	repositoriesLock.Lock()
	defer repositoriesLock.Unlock()

	repository := r[repositoryName]
	if repository == nil {
		repository = newRepositoryData(repositoryName)
		r[repositoryName] = repository
	}

	return repository
}

func (r repositoriesData) process(segments []string, info FileInfo) error {
	for idx := 0; idx < len(segments)-1; idx++ {
		repository := segments[0:idx]
		args := segments[idx+1:]

		switch segments[idx] {
		case "_layers":
			return r.get(repository).addLayer(args, info)

		case "_manifests":
			return r.get(repository).addManifest(args, info)

		case "_uploads":
			return r.get(repository).addUpload(args, info)
		}
	}

	return fmt.Errorf("unparseable path: %v", segments)
}

func (r repositoriesData) walkFile(path string, info FileInfo, err error) error {
	err = r.process(strings.Split(path, "/"), info)

	if err != nil {
		logrus.Errorln("REPOSITORY:", path, ":", err)
		if *softErrors {
			return nil
		}
		return err
	}

	return nil
}

func (r repositoriesData) walkPath(walkPath string, jg *jobGroup) error {
	logrus.Infoln("REPOSITORIES DIR:", walkPath)
	return currentStorage.Walk(walkPath, "repositories", func(path string, info FileInfo, err error) error {
		jg.dispatch(func() error {
			return r.walkFile(path, info, err)
		})
		return nil
	})
}

func (r repositoriesData) walk(parallel bool) error {
	logrus.Infoln("Walking REPOSITORIES...")

	jg := jobsRunner.group()

	if parallel {
		err := parallelWalk("repositories", func(listPath string) error {
			return r.walkPath(listPath, jg)
		})
		if err != nil {
			return err
		}
	} else {
		err := r.walkPath("repositories", jg)
		if err != nil {
			return err
		}
	}

	return jg.finish()
}

func (r repositoriesData) markRepository(jg *jobGroup, blobs blobsData, repository *Repository) {
	jg.dispatch(func() error {
		return repository.mark(blobs)
	})
}

func (r repositoriesData) mark(blobs blobsData) error {
	jg := jobsRunner.group()

	for _, repository := range r {
		r.markRepository(jg, blobs, repository)
	}

	err := jg.finish()
	if err != nil {
		return err
	}
	return nil
}

func (r repositoriesData) sweepRepository(jg *jobGroup, repository *Repository) {
	jg.dispatch(func() error {
		return repository.sweep()
	})
}

func (r repositoriesData) sweep() error {
	jg := jobsRunner.group()

	for _, repository := range r {
		r.sweepRepository(jg, repository)
	}

	err := jg.finish()
	if err != nil {
		return err
	}
	return nil
}

func (r repositoriesData) info(blobs blobsData, csvOutput string) {
	var stream io.WriteCloser

	if csvOutput != "" {
		var err error
		stream, err = os.Create(csvOutput)
		if err == nil {
			defer stream.Close()

			labels := []string{
				"Repository",
				"Tags",
				"TagVersions",
				"Manifests",
				"ManifestsUnused",
				"Layers",
				"LayersUnused",
				"Data",
				"DataUnused",
				"Data-MB",
				"DataUnused-MB",
			}

			fmt.Fprintln(stream, strings.Join(labels, ","))
		} else {
			logrus.Warningln(err)
		}
	}

	for _, repository := range r {
		repository.info(blobs, stream)
	}
}
