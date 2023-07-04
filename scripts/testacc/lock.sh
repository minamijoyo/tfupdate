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
    terraform providers lock -platform=linux_amd64 -platform=darwin_amd64 -platform=darwin_arm64
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
  TFUPDATE_LOG=DEBUG tfupdate lock --platform=linux_amd64 --platform=darwin_amd64 --platform=darwin_arm64 -r ./

  ALL_DIRS=$(find . -type f -print0 -name '*.tf' | xargs -0 -I {} dirname {} | sort | uniq | grep -v 'modules/')
  for dir in ${ALL_DIRS}
  do
    pushd "$dir"

    # got
    mv .terraform.lock.hcl .terraform.lock.hcl.got

    # want
    terraform providers lock -platform=linux_amd64 -platform=darwin_amd64 -platform=darwin_arm64

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
