#!/bin/bash

set -xeo pipefail

cd "$(dirname "$0")"

IP_ADDR="$(hostname -I | cut -d' ' -f1)"
REPO="${REPO:-$IP_ADDR:5000}"

build_and_push() {
    docker build -q --target "$2" -t "$REPO/$1" .
    docker push "$REPO/$1"
}

# rebuild `image-A` and `B` and `AB`
# latest has 3 revisions
build_and_push "image:latest" "A"
build_and_push "image:latest" "B"
build_and_push "image:latest" "AB"

# lets build specifically `A`
build_and_push "image:A" "A"
build_and_push "image:latest" "A"

# lets replace latest with `A`
build_and_push "image:latest" "AB"

# redundant stays
# B
