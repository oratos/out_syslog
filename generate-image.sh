#!/bin/bash

docker build \
		--build-arg BASE_IMAGE=cnabu-docker-local.artifactory.eng.vmware.com/base/ubuntu:bionic-vmware-2019-11-01-21-34-33 \
		--build-arg GOLANG_SOURCE=https://artifactory.eng.vmware.com:443/golang-dist/go1.14.10.linux-amd64.tar.gz \
		-f Dockerfile \
		. \
		-t gcr.io/cf-pks-releng-environments/oratos/fluent-bit:dev \
		> /tmp/fluent-bit.log 2>&1; 
