package main

import (
	"flag"
	"github.com/Sirupsen/logrus"
)

var (
	debug   = flag.Bool("debug", false, "Print debug messages")
	verbose = flag.Bool("verbose", true, "Print verbose messages")
	dryRun  = flag.Bool("dry-run", true, "Dry run")
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

	currentStorage = &fsStorage{}

	blobs := make(blobsData)
	repositories := make(repositoriesData)

	logrus.Infoln("Walking BLOBS...")
	err = currentStorage.Walk("blobs", blobs.walk)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Infoln("Walking REPOSITORIES...")
	err = currentStorage.Walk("repositories", repositories.walk)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Infoln("Marking REPOSITORIES...")
	err = repositories.mark(blobs)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Infoln("Sweeping BLOBS...")
	blobs.sweep()

	logrus.Warningln("Deleted:", deletedLinks, "links,",
		deletedBlobs, "blobs,",
		deletedBlobSize/1024/1024, "in MB",
	)

	if !*dryRun {
		logrus.Infoln("Sweeping...")
		runDeletes()
	}
}
