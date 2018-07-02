#!/usr/bin/env bash
## File: build_code.sh
## Author : Denny <zdenny@vmware.com>
## Description :
## --
## Created : <2018-07-02>
## Updated: Time-stamp: <2018-07-02 15:37:25>
##-------------------------------------------------------------------
function build_code {
    cd ..
    echo "go get out_syslog"
    go get -d github.com/oratos/out_syslog/...
    echo "go build tests/out_syslog.so"
    go build -buildmode c-shared -o tests/out_syslog.so github.com/oratos/out_syslog/cmd    
}

function run_container_build() {
    cd ..
    # Base Image: https://hub.docker.com/r/library/golang/
    # TODO: improve this
    docker stop go-build; docker rm go-build

    echo "Run container(go-build) to build the code"
    docker run -rm -t -d -h go-build --name go-build \
           -v ${PWD}/cmd:/go/cmd -v ${PWD}/pkg:/go/pkg \
           -v ${PWD}/tests:/go/tests \
           --entrypoint=/go/tests/build_code.sh golang:1.10.3
}
