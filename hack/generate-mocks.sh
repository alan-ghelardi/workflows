#!/usr/bin/env bash

################################################################################
### This script generates mocks from interfaces using mockgen:
### https://github.com/golang/mock.
################################################################################

set -o errexit
set -o nounset
set -o pipefail

cur_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"

if ! command -v mockgen &> /dev/null
then
    echo "Installing mockgen tool..."
    go install github.com/golang/mock/mockgen@v1.5.0
    echo "Done"
fi

gen_mocks() {
    local source=$1
    echo "Generating mocks for interfaces declared at $source..."
    cd $cur_dir/..
    mockgen -source=$source \
            -destination="$(dirname $source)/mocks/$(basename $source)" \
            -package=mocks \
            -copyright_file="$cur_dir/boilerplate/boilerplate.go.txt"
    echo "Done"
}


gen_mocks pkg/github/interfaces.go
gen_mocks pkg/github/workflow_reader.go
