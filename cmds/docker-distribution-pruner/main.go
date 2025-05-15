package main

import (
	"os"

	"github.com/doc-sheet/docker-distribution-pruner/experimental"
)

func main() {
	println("docker-distribution-pruner has been deprecated and will be removed soon.")
	println("Use https://docs.gitlab.com/ee/administration/packages/container_registry.html#container-registry-garbage-collection instead.")
	env := os.Getenv("EXPERIMENTAL")
	if env == "true" || env == "1" {
		experimental.Main()
		return
	}

	println("Use old experimental set of the features with `EXPERIMENTAL=true`")
	os.Exit(1)
}
