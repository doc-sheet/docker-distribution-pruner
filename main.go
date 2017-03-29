package main

import (
	"flag"

	"github.com/Sirupsen/logrus"
)

var (
	debug   = flag.Bool("debug", false, "Print debug messages")
	verbose = flag.Bool("verbose", true, "Print verbose messages")
	dryRun  = flag.Bool("dry-run", true, "Dry run")
	storage = flag.String("storage", "filesystem", "Storage type to use: filesystem or s3")
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
		currentStorage = &fsStorage{}

	case "s3":
		currentStorage = newS3Storage()

	default:
		logrus.Fatalln("Unknown storage specified:", *storage)
	}

	blobs := make(blobsData)
	repositories := make(repositoriesData)
	deletes := make(deletesData, 0, 1000)

	logrus.Infoln("Walking REPOSITORIES...")
	err = currentStorage.Walk("repositories", repositories.walk)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Infoln("Walking BLOBS...")
	err = currentStorage.Walk("blobs", blobs.walk)
	if err != nil {
		logrus.Fatalln(err)
	}

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
}
