#!/usr/bin/env bash
set -exuo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source .envrc

GOOS=linux go build -ldflags="-s -w" -o bin/v3-detector github.com/buildpack/lifecycle/cmd/detector
GOOS=linux go build -ldflags="-s -w" -o bin/v3-builder github.com/buildpack/lifecycle/cmd/builder

GOOS=linux go build -ldflags="-s -w" -o bin/detect github.com/cloudfoundry/libbuildpack/shims/cmd/detect
GOOS=linux go build -ldflags="-s -w" -o bin/supply github.com/cloudfoundry/libbuildpack/shims/cmd/supply
GOOS=linux go build -ldflags="-s -w" -o bin/finalize github.com/cloudfoundry/libbuildpack/shims/cmd/finalize
GOOS=linux go build -ldflags="-s -w" -o bin/release github.com/cloudfoundry/libbuildpack/shims/cmd/release

# TODO : iterate over the buildpacks listed in the order.toml file instead of hand-coding these
mkdir -p cnbs/org.cloudfoundry.buildpacks.nodejs/0.0.1
pushd cnbs/org.cloudfoundry.buildpacks.nodejs/0.0.1
    wget https://github.com/cloudfoundry/nodejs-cnb/releases/download/v0.0.1-alpha/nodejs-cnb.tgz
    tar xzvf nodejs-cnb.tgz
    rm nodejs-cnb.tgz
popd

mkdir -p cnbs/org.cloudfoundry.buildpacks.npm/0.0.1
pushd cnbs/org.cloudfoundry.buildpacks.npm/0.0.1
    wget https://github.com/cloudfoundry/npm-cnb/releases/download/v0.0.1-alpha/npm-cnb.tgz
    tar xzvf npm-cnb.tgz
    rm npm-cnb.tgz
popd
