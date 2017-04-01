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
	storage          = flag.String("storage", "", "Storage type to use: filesystem or s3")
	jobs             = flag.Int("jobs", 10, "Number of concurrent jobs to execute")
	parallelWalkJobs = flag.Int("parallel-walk-jobs", 10, "Number of concurrent parallel walk jobs to execute")
	ignoreBlobs      = flag.Bool("ignore-blobs", false, "Ignore blobs processing and recycling")
	softErrors       = flag.Bool("soft-errors", false, "Print errors, but do not fail")
)

func logErrorln(args ...interface{}) {
	if *softErrors {
		logrus.Errorln(args...)
	} else {
		logrus.Fatalln(args...)
	}
}

func main() {
	flag.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if *verbose {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true})

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

	jobsRunner.run(*jobs)
	parallelWalkRunner.run(*parallelWalkJobs)

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, os.Kill)

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
			logErrorln(err)
		}
	}()

	go func() {
		defer wg.Done()

		if *ignoreBlobs {
			return
		}

		err = blobs.walk()
		if err != nil {
			logErrorln(err)
		}
	}()

	wg.Wait()

	logrus.Infoln("Marking REPOSITORIES...")
	err = repositories.mark(blobs)
	if err != nil {
		logErrorln(err)
	}

	logrus.Infoln("Sweeping REPOSITORIES...")
	err = repositories.sweep()
	if err != nil {
		logErrorln(err)
	}

	logrus.Infoln("Sweeping BLOBS...")
	err = blobs.sweep()
	if err != nil {
		logErrorln(err)
	}

	logrus.Infoln("Summary...")
	repositories.info(blobs)
	blobs.info()
	deletesInfo()
	currentStorage.Info()
}
