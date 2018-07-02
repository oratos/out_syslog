#!/usr/bin/env bash
## File: test.sh
## Author : Denny <zdenny@vmware.com>
## Description :
## --
## Created : <2018-07-02>
## Updated: Time-stamp: <2018-07-02 15:42:49>
##-------------------------------------------------------------------
# shellcheck disable=SC1091
source library.sh
set -e

function test_golangci_lint() {
    go get "github.com/onsi/ginkgo"
    go get "github.com/onsi/gomega"
    go get "github.com/oratos/out_syslog/pkg/fluentbin"
    go get "code.cloudfoundry.org/rfc5424"
    log "Run golangci-lint run"
    # run a bunch of code checkers/linters in parallel
    golangci-lint run
}

################################################################################
install_golangci_lint
cd ..
test_golangci_lint

# TODO: Add more tests like below
# go test -v -race ./...
