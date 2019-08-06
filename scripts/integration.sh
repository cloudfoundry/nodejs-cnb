#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source .envrc
./scripts/install_tools.sh

GINKGO_NODES=${GINKGO_NODES:-3}
GINKGO_ATTEMPTS=${GINKGO_ATTEMPTS:-1}
export CF_STACK=${CF_STACK:-cflinuxfs3}

pushd integration
    echo "Run Uncached Shim Buildpack"
    ginkgo -mod vendor -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES --slowSpecThreshold=60 -- --cached=false

    echo "Run Cached Shim Buildpack"
    ginkgo -mod vendor -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES --slowSpecThreshold=160 -- --cached=true
popd
