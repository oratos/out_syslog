#!/usr/bin/env bash
## File: test.sh
## Author : Denny <zdenny@vmware.com>
## Description :
## --
## Created : <2018-07-02>
## Updated: Time-stamp: <2018-07-02 23:17:31>
##-------------------------------------------------------------------
set -e
function log {
    local msg=$*
    date_timestamp=$(date +['%Y-%m-%d %H:%M:%S'])
    echo -ne "$date_timestamp $msg\\n"

    if [ -n "$LOG_FILE" ]; then
        echo -ne "$date_timestamp $msg\\n" >> "$LOG_FILE"
    fi
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
log "Build golang code"
./build_code.sh

# TODO: better logic for code slow build issue
sleep 5

log "Run: docker-compose up -d"
docker-compose up -d

# TODO: better logic for fluent-bit delay
sleep 5

log "Verify syslog for output"
## We should see sample output like below in syslog-server container
## ,-----------
## | 50 <14>1 2018-07-03T01:29:49.002601+00:00 - - - - - 
## | 50 <14>1 2018-07-03T01:29:50.000191+00:00 - - - - - 
## | 49 <14>1 2018-07-03T01:29:51.00013+00:00 - - - - - 
## `-----------
docker logs syslog-server | grep "^[0-9][0-9] <[0-9][0-9]>[0-9] [0-9][0-9][0-9][0-9]-" | tail -n 10
