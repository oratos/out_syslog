#!/usr/bin/env bash
set -e

function log {
    local msg=$*
    date_timestamp=$(date +['%Y-%m-%d %H:%M:%S'])
    echo -ne "$date_timestamp $msg\\n"
}

function wait_for() {
    local check_command=${1?}
    local timeout_seconds=${2:-10}
    local wait_interval=${3:-1}
    log "Wait for: $check_command"
    for((i=0; i<timeout_seconds; i++)); do
        if eval "$check_command"; then
            return
        fi
        sleep "$wait_interval"
    done

    log "Error: wait for more than $timeout_seconds seconds"
    exit 1
}

function shell_exit {
    errcode=$?
    if [ $errcode -eq 0 ]; then
        log "Status check is fine"
    else
        log "ERROR: some status check has failed"
    fi
    exit $errcode
}

trap shell_exit SIGHUP SIGINT SIGTERM 0
################################################################################
echo "Note: The whole test would take 40 seconds on average"

log "Build golang code"
./build-code.sh

log "Wait for code build, which might take tens of seconds"
wait_for "test -f out_syslog.so" 50

log "Run: docker-compose down"
docker-compose down

log "Run: docker-compose up -d"
docker-compose up -d

log "Sleep several seconds for fluent-bit delay"
# TODO: better logic
sleep 5

log "Verify syslog for output"
## We should see sample output like below in syslog-server container
## ,-----------
## | 50 <14>1 2018-07-03T01:29:49.002601+00:00 - - - - - 
## | 50 <14>1 2018-07-03T01:29:50.000191+00:00 - - - - - 
## | 49 <14>1 2018-07-03T01:29:51.00013+00:00 - - - - - 
## `-----------
docker logs syslog-server | grep "^[0-9][0-9] <[0-9][0-9]>[0-9] [0-9][0-9][0-9][0-9]-" | tail -n 10
