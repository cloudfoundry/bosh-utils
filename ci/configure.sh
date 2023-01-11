#!/bin/bash

set -e

lpass ls > /dev/null

fly -t "${CONCOURSE_TARGET:-bosh-ecosystem}" set-pipeline -p bosh-utils -c ci/pipeline.yml
