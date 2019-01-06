#!/bin/bash

# to use this tool, your `Docker Engine` needs to run with
# `--insecure-registry IP:5000`
#
# Take a look at: https://docs.docker.com/registry/insecure/

exec registry serve ./examples/registry/config.yml
