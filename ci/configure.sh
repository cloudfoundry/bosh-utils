#!/bin/bash

set -e

fly -t "${CONCOURSE_TARGET:-bosh}" set-pipeline -p bosh-utils -c ci/pipeline.yml \
    --load-vars-from <(lpass show -G "bosh-utils concourse secrets" --notes) \
    --load-vars-from <(lpass show --note "bosh:docker-images concourse secrets")
