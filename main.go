package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/Sirupsen/logrus"
)

var (
	debug            = flag.Bool("debug", false, "Print debug messages")
	verbose          = flag.Bool("verbose", true, "Print verbose messages")
	dryRun           = flag.Bool("dry-run", true, "Dry run")
	storage          = flag.String("storage", "filesystem", "Storage type to use: filesystem or s3")
	jobs             = flag.Int("jobs", 10, "Number of concurrent jobs to execute")
	parallelWalkJobs = flag.Int("parallel-walk-jobs", 10, "Number of concurrent parallel walk jobs to execute")
	ignoreBlobs      = flag.Bool("ignore-blobs", true, "Ignore blobs processing and recycling")
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
	parallelWalkRunner.run(*parallelWalkJobs)

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt)

	go func() {
		for signal := range signals {
			currentStorage.Info()
			logrus.Fatalln("Signal received:", signal)
		}
	}()

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

		if *ignoreBlobs {
			return
		}

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

	repositories.info(blobs)
	blobs.info()
	currentStorage.Info()
}
