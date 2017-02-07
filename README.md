## Docker Distribution Pruner (experimental)

Go to Docker Distribution: https://github.com/docker/distribution/.

Simple Go-lang docker-distribution based application to clean all old revisions from Docker Distribution based registry (also GitLab Container Registry)

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
docker-distribution-pruner -config /path/to/docker/distribution/config/file -delete-manifests=true -delete-blobs=true -delete-global-blobs=true
```

### Options

All options are off by default:
```
-delete-manifests=true - delete unreferenced manifests for each found repository, this unlinks all previous revisions of tags
-delete-blobs=true - delete unreferenced blobs for each found repository, this unlinks all blobs referenced in context of this repository
-delete-global-blobs=true - physically delete manifests and blobs that are no longer used, physically removes data
```

### License

GitLab, MIT, 2017

### Author

Kamil Trzciński <kamil@gitlab.com>
