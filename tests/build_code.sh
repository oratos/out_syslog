#!/usr/bin/env bash
## File: build_code.sh
## Author : Denny <zdenny@vmware.com>
## Description :
## --
## Created : <2018-07-02>
## Updated: Time-stamp: <2018-07-02 16:46:31>
##-------------------------------------------------------------------
function build_code {
    cd ..
    go get -d github.com/oratos/out_syslog/...
    go build -buildmode c-shared -o tests/out_syslog.so github.com/oratos/out_syslog/cmd
}

function run_container {
    local container_name="go-build"
    cd ..
    # Base Image: https://hub.docker.com/r/library/golang/
    if docker ps | grep "$container_name" >/dev/null 2>&1; then
        echo "Delete existing container: $container_name"
        docker stop "$container_name" || docker stop "$container_name" || true
        docker rm "$container_name"
    fi

    echo "Run container($container_name) to build the code"
    docker run --rm -t -d -h "$container_name" --name "$container_name" \
           -v "${PWD}/cmd:/go/cmd" -v "${PWD}/pkg:/go/pkg" \
           -v "${PWD}/tests:/go/tests" \
           golang:1.10.3 bash -c "/go/tests/build_code.sh build_code"

    echo "To check status, run: docker logs $container_name"
}

action=${1:-run_container}

set -xe
if  [ "$action" = "build_code" ]; then
    build_code
    # avoid quit the container, thus people can login and debug
    tail -f /dev/null
else
    run_container
fi
