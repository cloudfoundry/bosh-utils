#!/usr/bin/env bash

set -ex

export PATH=/usr/local/go/bin:$GOPATH/bin:$PATH

cd gopath/src/github.com/cloudfoundry/bosh-utils
bin/test-unit
