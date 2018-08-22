package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/Sirupsen/logrus"
)

var (
	jobsRunner         = make(jobsData)
	parallelWalkRunner = make(jobsData)
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

	if *config == "" {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	currentStorage, err = storageFromConfig(*config)
	if err != nil {
		logrus.Fatalln(err)
	}

	blobs := make(blobsData)
	repositories := make(Repository)

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

		err = repositories.walk(*parallelRepositoryWalk)
		if err != nil {
			logErrorln(err)
		}
	}()

	go func() {
		defer wg.Done()

		if *ignoreBlobs {
			return
		}

		err = blobs.walk(*parallelBlobWalk)
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
	repositories.info(blobs, *repositoryCsvOutput)
	blobs.info()
	deletesInfo()
	currentStorage.Info()
}
