#!/bin/bash

set -eo pipefail

script_full_path=$(dirname "$0")

# test simple
bash "$script_full_path/lock.sh" run simple

# test all
repo_root_dir="$(git rev-parse --show-toplevel)"
fixturesdir="$repo_root_dir/test-fixtures/lock/"

fixtures=$(find "$fixturesdir" -type d -mindepth 1 -maxdepth 1 -exec basename {} \; | sort)

for fixture in ${fixtures}
do
  echo "$fixture"
  bash "$script_full_path/lock.sh" run "$fixture"
done
