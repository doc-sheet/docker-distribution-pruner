package flags

import "flag"

var (
	Config = flag.String("config", "", "Path to registry config file")

	IgnoreBlobs    = flag.Bool("ignore-blobs", false, "Ignore blobs processing and recycling")
	S3CacheStorage = flag.String("s3-storage-cache", "tmp-cache", "s3 cache")

	Jobs             = flag.Int("jobs", 10, "Number of concurrent jobs to execute")
	ParallelWalkJobs = flag.Int("parallel-walk-jobs", 10, "Number of concurrent parallel walk jobs to execute")

	Debug      = flag.Bool("debug", false, "Print debug messages")
	Verbose    = flag.Bool("verbose", true, "Print verbose messages")
	SoftErrors = flag.Bool("soft-errors", false, "Print errors, but do not fail")

	ParallelRepositoryWalk = flag.Bool("parallel-repository-walk", false, "Allow to use parallel repository walker")
	ParallelBlobWalk       = flag.Bool("parallel-blob-walk", false, "Allow to use parallel blob walker")

	RepositoryCsvOutput = flag.String("repository-csv-output", "repositories.csv", "File to which CSV will be written with all metrics")

	Delete               = flag.Bool("delete", false, "Delete data, instead of dry run")
	DeleteOldTagVersions = flag.Bool("delete-old-tag-versions", true, "Delete old tag versions")
	SoftDelete           = flag.Bool("soft-delete", true, "When deleting, do not remove, but move to backup/ folder")
)
