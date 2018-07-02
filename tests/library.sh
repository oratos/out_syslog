#!/usr/bin/env bash
##-------------------------------------------------------------------
## File: library.sh
## Author : Denny <zdenny@vmware.com>
## Description :
## --
## Created : <2018-07-02>
## Updated: Time-stamp: <2018-07-02 15:44:33>
##-------------------------------------------------------------------
set -e

function ensure_variable_isset {
    var=${1?}
    message=${2:-"parameter name should be given"}
    # TODO support sudo, without source
    if [ -z "$var" ]; then
        echo "Error: Certain variable($message) is not set"
        exit 1
    fi
}

################################################################################
function install_golangci_lint () {
    if ! command -v golangci-lint 1>/dev/null 2>&1; then
        echo "Install golangci-lint"
        curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b "$GOPATH/bin" v1.8.1
    fi
}

install_golangci_lint
## File: library.sh ends
