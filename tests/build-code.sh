#!/usr/bin/env bash
set -e
function log {
    local msg=$*
    date_timestamp=$(date +['%Y-%m-%d %H:%M:%S'])
    echo -ne "$date_timestamp $msg\\n"
}

function ensure_variable_isset {
    var=${1?}
    message=${2:-"parameter name should be given"}
    if [ -z "$var" ]; then
        echo "Error: Certain variable($message) is not set"
        exit 1
    fi
}

function build_code {
    if [ -f /go/tests/out_syslog.so ]; then
        log "Remove old out_syslog.so"
        rm -rf /go/tests/out_syslog.so
    fi

    cd "$GOPATH/src/github.com/pivotal-cf/fluent-bit-out-syslog/cmd"
    if [ ! -d "$GOPATH/src/github.com/fluent/fluent-bit-go" ]; then
        log "go get dependency"
        go get -d -t ./ ...
    else
        log "Avoid go get dependencies, since fluent-bit-go package is already detected"
    fi
    
    log "go build local code directory"
    go build -buildmode c-shared -o /go/tests/out_syslog.so .
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
    # Note: here we mount the whole $GOPATH folder, which might not be clean
    docker run -t -d -h "$container_name" --name "$container_name" \
           -v "${GOPATH}:/go" \
           -v "${PWD}/tests:/go/tests" \
           golang:1.10.3 bash -c "cd /go && tests/build-code.sh build_code"

    log "To check detail status, run: docker logs -f $container_name"
}

action=${1:-run_container}

if  [ "$action" = "build_code" ]; then
    ensure_variable_isset "$GOPATH" "GOPATH env should be set"
    build_code
    log "Keep container up and running, via \"tail -f /dev/null\""
    tail -f /dev/null
else
    run_container
fi
