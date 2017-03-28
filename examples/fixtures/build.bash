#!/bin/bash

set -xeo pipefail

cd "$(dirname "$0")"

REPO=192.168.65.1:5000

build_and_push() {
    date > file2
    docker build -t "$REPO/$1" .
    docker push "$REPO/$1"
}

tag_and_push() {
    docker tag "$REPO/$1" "$REPO/$2"
    docker push "$REPO/$2"
}

build_and_push "test:latest"
build_and_push "test:latest"
tag_and_push "test:latest" "test2:latest"
build_and_push "test:latest"
tag_and_push "test:latest" "test2:latest2"
