#!/bin/bash

exec registry garbage-collect ./examples/registry/config.yml "$@"
