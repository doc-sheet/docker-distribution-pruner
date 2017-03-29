## Docker Distribution Pruner (highly experimental)

Go to Docker Distribution: https://github.com/docker/distribution/.

Highly efficient Garbage Collector to clean all old revisions from Docker Distribution based registry (also GitLab Container Registry).

It uses optimised file accesses and API calls to create walk DAG.

**It is only for testing purposes now. Do not yet use that for production data.**

### Installation

go get -u gitlab.com/gitlab-org/docker-distribution-pruner

### Run

Dry run:
```bash
docker-distribution-pruner -config /path/to/docker/distribution/config/file
```

Reclaim disk space:
```bash
docker-distribution-pruner -config /path/to/docker/distribution/config/file -dry-run=false
```

### Options

It is highly not advised to change these options as it can leave left-overs in repository.

```
-delete-old-tag-versions=true - delete old versions for each found tag of the repository repository
```

### License

GitLab, MIT, 2017

### Author

Kamil Trzciński <kamil@gitlab.com>
