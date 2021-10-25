#!/bin/bash

set -e

git clone bosh-utils bumped-bosh-utils

cd bumped-bosh-utils

#intentionally cause an explicit commit if the underlying go version in our compiled Dockerfile changes. 
#assume the go-dep-bumper and bosh-utils are bumping at the same cadence.
export GO_MINOR=$(go version | sed 's/go version go1\.\([0-9]\+\)\..*$/\1/g')
sed -i "s/\(go 1.\)\([0-9]\+\)/\1$GO_MINOR/" go.mod

go get -u ./...
go mod tidy
go mod vendor

if [ "$(git status --porcelain)" != "" ]; then
  git status
  git add vendor go.sum go.mod
  git config user.name "CI Bot"
  git config user.email "cf-bosh-eng@pivotal.io"
  git commit -m "Update vendored dependencies"
fi

