##  Build docker image:
##     docker build --no-cache -f Dockerfile --rm=true .
##
## Base Image: https://hub.docker.com/r/library/golang/
FROM golang:1.10.3

# Mount the code
ADD cmd /go/cmd
ADD pkg /go/pkg

# TODO: speed up this, by adding another layer of image
RUN go get -d github.com/oratos/out_syslog/...

# Build the code
RUN go build -buildmode c-shared -o /go/out_syslog.so github.com/oratos/out_syslog/cmd

# Verify docker image
RUN go version | grep 1.10
