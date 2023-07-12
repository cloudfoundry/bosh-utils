#!/bin/bash

set -e

fly -t "${CONCOURSE_TARGET:-bosh-ecosystem}" set-pipeline -p bosh-utils -c ci/pipeline.yml
