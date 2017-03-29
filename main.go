package main

import (
	"flag"

	"github.com/Sirupsen/logrus"
	"sync"
)

var (
	debug   = flag.Bool("debug", false, "Print debug messages")
	verbose = flag.Bool("verbose", true, "Print verbose messages")
	dryRun  = flag.Bool("dry-run", true, "Dry run")
	storage = flag.String("storage", "filesystem", "Storage type to use: filesystem or s3")
	jobs    = flag.Int("jobs", 100, "Number of concurrent jobs to execute")
)

func main() {
	flag.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if *verbose {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	var err error

	switch *storage {
	case "filesystem":
		currentStorage = newFsStorage()

	case "s3":
		currentStorage = newS3Storage()

	default:
		logrus.Fatalln("Unknown storage specified:", *storage)
	}

	blobs := make(blobsData)
	repositories := make(repositoriesData)
	deletes := make(deletesData, 0, 1000)

	jobsRunner.run(*jobs)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err = repositories.walk()
		if err != nil {
			logrus.Fatalln(err)
		}
	}()

	go func() {
		defer wg.Done()

		err = blobs.walk()
		if err != nil {
			logrus.Fatalln(err)
		}
	}()

	wg.Wait()

	logrus.Infoln("Marking REPOSITORIES...")
	err = repositories.mark(blobs, deletes)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Infoln("Sweeping BLOBS...")
	blobs.sweep(deletes)

	deletes.info()

	if !*dryRun {
		logrus.Infoln("Sweeping...")
		deletes.run()
	}

	currentStorage.Info()
}
