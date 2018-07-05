#!/usr/bin/env bash
set -e
function log {
    local msg=$*
    date_timestamp=$(date +['%Y-%m-%d %H:%M:%S'])
    echo -ne "$date_timestamp $msg\\n"
}

function build_code {
    if [ -f tests/out_syslog.so ]; then
        log "Remove old out_syslog.so"
        rm -rf tests/out_syslog.so
    fi
    log "go get dependency"
    go get -d -t github.com/fluent/fluent-bit-go/output
    go get -d -t code.cloudfoundry.org/rfc5424
    
    log "go build local code directory"
    go build -buildmode c-shared -o tests/out_syslog.so ./cmd
}

function run_container {
    local container_name="go-build"
    cd ..
    if docker ps -a | grep "$container_name" >/dev/null 2>&1; then
        log "Delete existing container: $container_name"
        docker stop "$container_name" || docker stop "$container_name" || true
        docker rm "$container_name" >/dev/null
    fi

    log "Run container($container_name) to build the code"
    docker run -t -d -h "$container_name" --name "$container_name" \
           -v "${PWD}/cmd:/go/cmd" \
           -v "${PWD}/pkg/syslog:/go/src/github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog" \
           -v "${PWD}/tests:/go/tests" \
           golang:1.10.3 bash -c "cd /go && tests/build-code.sh build_code"

    log "To check detail status, run: docker logs -f $container_name"
}

action=${1:-run_container}

if  [ "$action" = "build_code" ]; then
    build_code
    log "Keep container up and running, via \"tail -f /dev/null\""
    tail -f /dev/null
else
    run_container
fi
