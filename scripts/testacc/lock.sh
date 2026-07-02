#!/bin/bash

set -eo pipefail

usage()
{
  cat << EOF
  Usage: $(basename "$0") <command> <fixture>

  Arguments:
    command: A name of step to run. Valid values are:
             run | setup | provider | lock | cleanup
    fixture: A name of fixture in test-fixtures/lock/
EOF
}

setup()
{
  cp -prT "$FIXTUREDIR" ./
  ALL_DIRS=$(find . -type f -print0 -name '*.tf' | xargs -0 -I {} dirname {} | sort | uniq | grep -v 'modules/')
  for dir in ${ALL_DIRS}
  do
    pushd "$dir"

    # always create a new lock
    rm -f .terraform.lock.hcl
    if [[ "$REGISTRY_H1_SUPPORT" -eq 1 ]]; then
      "$TFUPDATE_EXEC_PATH" providers lock
    else
      "$TFUPDATE_EXEC_PATH" providers lock -platform=linux_amd64 -platform=darwin_amd64 -platform=darwin_arm64
    fi
    cat .terraform.lock.hcl
    rm -rf .terraform

    popd
  done
}

provider()
{
  TFUPDATE_LOG=DEBUG tfupdate provider null -v 3.2.1 -r ./
}

lock()
{
    if [[ "$REGISTRY_H1_SUPPORT" -eq 1 ]]; then
      TFUPDATE_LOG=DEBUG tfupdate lock -r ./
    else
      TFUPDATE_LOG=DEBUG tfupdate lock --platform=linux_amd64 --platform=darwin_amd64 --platform=darwin_arm64 -r ./
    fi

  ALL_DIRS=$(find . -type f -print0 -name '*.tf' | xargs -0 -I {} dirname {} | sort | uniq | grep -v 'modules/')
  for dir in ${ALL_DIRS}
  do
    pushd "$dir"

    # got
    mv .terraform.lock.hcl .terraform.lock.hcl.got

    # want
    if [[ "$REGISTRY_H1_SUPPORT" -eq 1 ]]; then
      "$TFUPDATE_EXEC_PATH" providers lock
    else
      "$TFUPDATE_EXEC_PATH" providers lock -platform=linux_amd64 -platform=darwin_amd64 -platform=darwin_arm64
    fi

    # assert the result
    cat .terraform.lock.hcl
    cat .terraform.lock.hcl.got
    diff -u .terraform.lock.hcl .terraform.lock.hcl.got

    popd
  done
}

cleanup()
{
  find ./ -mindepth 1 -delete
}

run()
{
  setup
  provider
  lock
  cleanup
}

# main
if [[ $# -ne 2 ]]; then
  usage
  exit 1
fi

set -x

COMMAND=$1
FIXTURE=$2

REPO_ROOT_DIR="$(git rev-parse --show-toplevel)"
WORKDIR="$REPO_ROOT_DIR/tmp/testacc/lock/$FIXTURE"
FIXTUREDIR="$REPO_ROOT_DIR/test-fixtures/lock/$FIXTURE/"
mkdir -p "$WORKDIR"
pushd "$WORKDIR"

# If the registry supports the h1 hashes, omit the platform args.
REGISTRY_H1_SUPPORT=0
if [[ "$TFUPDATE_EXEC_PATH" == tofu ]]; then
  CURRENT_TOFU_VERSION=$( tofu -v | grep -oE 'OpenTofu v?[0-9]+\.[0-9]+\.[0-9]+' | sed 's/^OpenTofu v//')
  MIN_TOFU_REGISTRY_H1_SUPPORT=1.12.0
  if [ "$(printf '%s\n%s\n' "$MIN_TOFU_REGISTRY_H1_SUPPORT" "$CURRENT_TOFU_VERSION" | sort -V | head -n1)" == "$MIN_TOFU_REGISTRY_H1_SUPPORT" ]; then
    REGISTRY_H1_SUPPORT=1
  fi
fi
echo "REGISTRY_H1_SUPPORT=$REGISTRY_H1_SUPPORT"

case "$COMMAND" in
  run | setup | provider | lock | cleanup )
    "$COMMAND"
    RET=$?
    ;;
  *)
    usage
    RET=1
    ;;
esac

popd
exit $RET
