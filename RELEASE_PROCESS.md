# Docker Distribution Pruner release process

Docker Distribution Pruner is released when it is ready.
There is no strict timeline of next releases.
We tend to release this project when the major features or bugs
are fixed.

## Workflow, merging and tagging strategy

### Stable release

To pin a new version of the application follow these steps:

1. Create a new `X-Y-stable` branch from `master`,
1. Update a `CHANGELOG.md` with a list of changes,
1. Create a new Merge Request pointing to `master`,
1. Merge a merge request that updates `VERSION` and `CHANGELOG.md`,
1. Push a branch `X-Y-stable` to GitLab (where `X` is `major`, `Y` is minor),
1. Push a new annotated tag `vX.Y.0` (where `Z` is patch release),
1. Merge `X-Y-stable` branch into `master`,
1. Update `VERSION` to point to next version and push to `master`,

If put in terminal steps:

```bash
# Create stable branch
git checkout -b 0-1-stable master
editor CHANGELOG.md
git add CHANGELOG.md VERSION
git commit -m "Update CHANGELOG for 0.1.0"
git tag -a v0.1.0 -m "Version 0.1.0"
git push origin 0-1-stable v0.1.0

# Update development
git checkout master
git merge 0-1-stable
editor VERSION # point to next version, like `0.2.0`
git commit -m "Update VERSION to 0.2.0"
git push origin master
```

### Patch release

To create a new patch release of the application follow these steps:

1. Start from `X-Y-stable` branch,
1. Pick all changes that goes into patch release, by picking a `merge commit` of the change,
1. Update `VERSION` and `CHANGELOG.md` to point to next patch release and list all changes,
1. Push a branch `X-Y-stable` to GitLab (where `X` is `major`, `Y` is minor),
1. Push a new annotated tag `vX.Y.0` (where `Z` is patch release),
1. DO NOT UPDATE `master` with the patch release.

If put in terminal steps:

```bash
# Create stable branch
git checkout 0-1-stable
editor CHANGELOG.md VERSION
git add CHANGELOG.md VERSION
git commit -m "Update CHANGELOG for 0.1.1"
git tag -a v0.1.1 -m "Version 0.1.1"
git push origin 0-1-stable v0.1.1
```
