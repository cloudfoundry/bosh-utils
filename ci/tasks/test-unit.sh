#!/usr/bin/env bash

set -ex

export GOPATH=$(pwd)/gopath
export PATH=/usr/local/go/bin:$GOPATH/bin:$PATH
export GO15VENDOREXPERIMENT=1

go env
go version

cd gopath/src/github.com/cloudfoundry/bosh-utils
go install ./vendor/github.com/onsi/ginkgo/ginkgo
bin/test-unit
