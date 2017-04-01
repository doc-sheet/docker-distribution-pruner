## Docker Distribution Pruner (highly experimental)

Go to Docker Distribution: https://github.com/docker/distribution/.

Highly efficient Garbage Collector to clean all old revisions from Docker Distribution based registry (also GitLab Container Registry).

It uses optimised file accesses and API calls to create walk DAG.

**It is only for testing purposes now. Do not yet use that for production data.**

### Installation

```bash
$ go get -u gitlab.com/gitlab-org/docker-distribution-pruner
```

### Preface

This tool watches Docker Distribution storage and tries to recycle unreferenced version of tags, unreferenced manifests,
unreferenced layers and thus blobs, effectively recycling used container registry storage.

By default it runs in dry run mode (no changes). When run with `-delete` it will soft delete all data by moving them to
`docker-backup` folder. In case of any problems you can move data back and restore previous state of registry.

If you run `-delete -soft-delete=false` you will remove data forever.

### Run

Dry run:

```bash
$ docker-distribution-pruner -config=/path/to/registry/configuration
```

Reclaim disk space:

```bash
$ docker-distribution-pruner -config=/path/to/registry/configuration -delete
```

### GitLab Omnibus

Run:

```bash
$ docker-distribution-pruner -config=/var/opt/gitlab/registry/config.yml
```

### S3 effectiveness

We do not download individual objects, instead do global wide list API call, returning 1000 objects at single time.
We use ETag and name of files to ensure consistency of repository instead of reading files where it is possible to save 
API calls, network bandwidth and improve speed.

Sometimes we have to download objects (links, manifests), and usually it is wasteful to do it every time.
Instead, when S3 is used the downloaded data are stored by default in `tmp-cache/`.
To ensure the data consistency we verify ETag (md5 of the object content).
For large repositories it allows to save hundreds of thousands requests and also with fast SSD drive it makes it crazy fast.

### Large registries

This tool can effectively run on registries that consists of million objects and terrabytes of data in reasonable time.
To ensure smooth run ensure to have at least 4GB for 5 million objects stored in registry.

To speed-up processing of large repositories enable parallel blobs and repository processing:

```bash
$ docker-distribution-pruner -config=/path/to/registry/configuration -parallel-repository-walk -parallel-blob-walk
```

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
  -config string
    	Path to registry config file
  -debug
    	Print debug messages
  -delete
    	Delete data, instead of dry run
  -delete-old-tag-versions
    	Delete old tag versions (default true)
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
  -s3-storage-cache string
    	s3 cache (default "tmp-cache")
  -soft-delete
    	When deleting, do not remove, but move to backup/ folder (default true)
  -soft-errors
    	Print errors, but do not fail
  -verbose
    	Print verbose messages (default true)
```

### License

GitLab, MIT, 2017

### Author

Kamil Trzciński <kamil@gitlab.com>
