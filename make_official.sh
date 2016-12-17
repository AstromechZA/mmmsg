#!/usr/bin/env bash

set -e

# first build the version string
VERSION=1.0

# add the git commit id and date
VERSION="$VERSION (commit $(git rev-parse --short HEAD) @ $(git log -1 --date=short --pretty=format:%cd))"

function buildbinary {
    goos=$1
    goarch=$2

    echo "Building official $goos $goarch binary for version '$VERSION'"

    outputfolder="build/${goos}_${goarch}"
    echo "Output Folder $outputfolder"
    mkdir -pv $outputfolder

    export GOOS=$goos
    export GOARCH=$goarch

    go build -i -v -o "$outputfolder/mmmsg" -ldflags "-X \"main.MMMsgVersion=$VERSION\"" github.com/AstromechZA/mmmsg

    echo "Done"
    ls -l "$outputfolder/mmmsg"
    file "$outputfolder/mmmsg"
    echo
}

# build for mac
buildbinary darwin amd64

# build for linux
buildbinary linux amd64
