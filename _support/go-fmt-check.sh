#!/bin/sh
FILES=`find cmds/ experimental/ -type f -name '\*.go' -exec gofmt -l -s '{}' +`
if [ -z $FILES ]
then
    exit 0
else
    echo "Run go fmt on the following files:"
    echo $FILES
    exit 1
fi
