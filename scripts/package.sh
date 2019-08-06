#!/usr/bin/env bash
set -eo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
./scripts/install_tools.sh

PACKAGE_DIR=${PACKAGE_DIR:-"${PWD##*/}_$(openssl rand -hex 4)"}

full_path=$(realpath "$PACKAGE_DIR")
args=".bin/cnb2cf package"

while getopts "csv:" arg
do
    case $arg in
    c) cached=true;;
    s) stack="${OPTARG}";;
    v) version="${OPTARG}";;
    *) echo "usage: $0 [-c] [-s <STACK>] [-v <VERSION>]" >&2;
      exit 1;;
    esac
done

if [[ -n "$cached" ]]; then # package as cached
    full_path="$full_path-cached"
    args="${args} -cached"
fi

if [[ -n "$stack" ]]; then # package for stack
    args="${args} -stack ${stack}"
fi

if [[ -z "$version" ]]; then # version not provided, use latest git tag
    git_tag=$(git describe --abbrev=0 --tags)
    version=${git_tag:1}
fi

args="${args} -version ${version}"

eval "${args}" "${full_path}"

if [[ -n "$BP_REWRITE_HOST" ]]; then
    sed -i '' -e "s|^uri = \"https:\/\/buildpacks\.cloudfoundry\.org\(.*\)\"$|uri = \"http://$BP_REWRITE_HOST\1\"|g" "$full_path/buildpack.toml"
fi
