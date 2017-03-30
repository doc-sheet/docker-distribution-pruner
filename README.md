## Docker Distribution Pruner (highly experimental)

Go to Docker Distribution: https://github.com/docker/distribution/.

Highly efficient Garbage Collector to clean all old revisions from Docker Distribution based registry (also GitLab Container Registry).

It uses optimised file accesses and API calls to create walk DAG.

**It is only for testing purposes now. Do not yet use that for production data.**

### Installation

```
go get -u gitlab.com/gitlab-org/docker-distribution-pruner
```

### Preface

This tool watches Docker Distribution storage and tries to recycle unreferenced version of tags, unreferenced manifests,
unreferenced layers and thus blobs, effectively recycling used container registry storage.

By default it runs in dry run mode (no changes). When run with `-delete` it will soft delete all data by moving them to
`docker-backup` folder. In case of any problems you can move data back and restore previous state of registry.

If you run `-delete -soft-delete=false` you will remove data forever.

### Run (filesystem)

Dry run:

```bash
docker-distribution-pruner -storage=filesystem -fs-root-dir=/path/to/registry/storage
```

Reclaim disk space:

```bash
docker-distribution-pruner -storage=filesystem -fs-root-dir=/path/to/registry/storage -delete
```

### Run (s3)

Configure credentials:
```
aws configure
```

Dry run:

```bash
docker-distribution-pruner -storage=s3 -s3-bucket=my-bucket
```

Reclaim disk space:

```bash
docker-distribution-pruner -storage=s3 -s3-bucket=my-bucket -delete
```

### Large registries

This tool can effectively run on registries that consists of million objects and terrabytes of data in reasonable time.
To ensure smooth run ensure to have at least 4GB for 5 million objects stored in registry.

You can also tune performance settings (less or more):
```
-jobs=100 -parallel-walk-jobs=100
```

### Report

After success run application generates number of data, lke a list of repositories with detailed usage.

### Warranty

Application was manually tested, also was run in dry run mode against large repositories to verify consistency.
**As of today we are still afraid of executing that on production data. There's no warranty that it will not break repository.**

### Options

It is highly not advised to change these options as it can leave left-overs in repository.

```
-delete-old-tag-versions=true - delete old versions for each found tag of the repository repository
```

### Command line

```
Usage of docker-distribution-pruner:
  -debug
    	Print debug messages
  -delete
    	Delete data, instead of dry run
  -delete-old-tag-versions
    	Delete old tag versions (default true)
  -fs-root-dir string
    	root directory (default "examples/registry")
  -ignore-blobs
    	Ignore blobs processing and recycling
  -jobs int
    	Number of concurrent jobs to execute (default 10)
  -parallel-blob-walk
    	Allow to use parallel blob walker (default true)
  -parallel-repository-walk
    	Allow to use parallel repository walker (default true)
  -parallel-walk-jobs int
    	Number of concurrent parallel walk jobs to execute (default 10)
  -repository-csv-output string
    	File to which CSV will be written with all metrics (default "repositories.csv")
  -s3-bucket string
    	s3 bucket
  -s3-region string
    	s3 region (default "us-east-1")
  -s3-root-dir string
    	s3 root directory
  -s3-storage-cache string
    	s3 cache (default "tmp-cache")
  -soft-delete
    	When deleting, do not remove, but move to backup/ folder (default true)
  -soft-errors
    	Print errors, but do not fail
  -storage string
    	Storage type to use: filesystem or s3
  -verbose
    	Print verbose messages (default true)
```

### License

GitLab, MIT, 2017

### Author

Kamil Trzciński <kamil@gitlab.com>
