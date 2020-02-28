ARG BASE_IMAGE=ubuntu:bionic
FROM $BASE_IMAGE as builder

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y \
    bison \
    ca-certificates \
    build-essential \
    cmake \
    flex \
    git \
    libsasl2-dev \
    libssl-dev \
    libsystemd-dev \
    unzip \
    wget \
    && apt-get clean

COPY vm-inventory.sh /vm-inventory.sh
RUN /vm-inventory.sh -s deb -m /builder-dpkg-list ubuntu

# Install Go
ARG GOLANG_SOURCE=https://dl.google.com/go/go1.13.6.linux-amd64.tar.gz
RUN wget $GOLANG_SOURCE -O go.tar.gz && \
    tar -xf go.tar.gz && \
    mv go /usr/local && \
    rm go.tar.gz
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

ENV GOOS=linux \
    GOARCH=amd64

COPY / /syslog-plugin/

RUN cd /syslog-plugin && go build \
    -a \
    -installsuffix fluent \
    -buildmode c-shared \
    -o /syslog-plugin/out_syslog.so \
    -mod=readonly \
    -mod=vendor \
    cmd/main.go

# https://github.com/pivotal/fluent-bit/releases/tag/v1.3.4
ENV FLB_SHA 81e4d54
ENV FLB_VERSION 81e4d54526b70250764063d24d47f0e952fed808
ENV FLB_TARBALL https://github.com/pivotal/fluent-bit/archive/$FLB_SHA.zip

RUN mkdir -p /fluent-bit/bin /fluent-bit/etc /fluent-bit/log /tmp/src/ \
    && wget -O "/tmp/fluent-bit-$FLB_SHA.zip" ${FLB_TARBALL} \
    && cd /tmp && unzip "fluent-bit-$FLB_SHA.zip" \
    && cd "fluent-bit-$FLB_VERSION"/build/ \
    && cmake -DFLB_DEBUG=On \
    -DFLB_TRACE=Off \
    -DFLB_JEMALLOC=On \
    -DFLB_BUFFERING=On \
    -DFLB_TLS=On \
    -DFLB_SHARED_LIB=Off \
    -DFLB_EXAMPLES=Off \
    -DFLB_HTTP_SERVER=On \
    -DFLB_OUT_KAFKA=On .. \
    && make \
    && install bin/fluent-bit /fluent-bit/bin/ \
    && rm -rf /tmp/fluent-bit-*

RUN echo "other:fluent-bit:${FLB_SHA}:" >> /builder-dpkg-list
RUN echo "    version: ${FLB_SHA}" >> /builder-dpkg-list
RUN echo "    name: fluent-bit" >> /builder-dpkg-list
RUN echo "    url: https://codeload.github.com/fluent/fluent-bit/tar.gz/${FLB_SHA}" >> /builder-dpkg-list
RUN echo "    other-url: https://github.com/fluent/fluent-bit" >> /builder-dpkg-list
RUN echo "    repository: Other" >> /builder-dpkg-list

# Configuration files
COPY /config/fluent-bit.conf \
    /config/parsers.conf \
    /config/parsers_java.conf \
    /config/parsers_mult.conf \
    /config/parsers_openstack.conf \
    /config/parsers_cinder.conf \
    /fluent-bit/etc/

FROM $BASE_IMAGE

RUN apt update && apt install -y --no-install-recommends ca-certificates && apt-get autoclean

# These COPY commands have been interlaced with RUN true due to the following
# issues:
# https://github.com/moby/moby/issues/37965#issuecomment-448926448
# https://github.com/moby/moby/issues/38866
COPY --from=builder /fluent-bit /fluent-bit
COPY --from=builder /syslog-plugin /syslog-plugin
COPY --from=builder /builder-dpkg-list /builder-dpkg-list
RUN true
COPY --from=builder /usr/lib/x86_64-linux-gnu/*sasl* /usr/lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /usr/lib/x86_64-linux-gnu/libz* /usr/lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /lib/x86_64-linux-gnu/libz* /lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /usr/lib/x86_64-linux-gnu/libssl.so* /usr/lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /usr/lib/x86_64-linux-gnu/libcrypto.so* /usr/lib/x86_64-linux-gnu/
# These below are all needed for systemd
COPY --from=builder /lib/x86_64-linux-gnu/libsystemd* /lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /lib/x86_64-linux-gnu/libselinux.so* /lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /lib/x86_64-linux-gnu/liblzma.so* /lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /usr/lib/x86_64-linux-gnu/liblz4.so* /usr/lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /lib/x86_64-linux-gnu/libgcrypt.so* /lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /lib/x86_64-linux-gnu/libpcre.so* /lib/x86_64-linux-gnu/
RUN true
COPY --from=builder /lib/x86_64-linux-gnu/libgpg-error.so* /lib/x86_64-linux-gnu/

EXPOSE 2020

CMD ["/fluent-bit/bin/fluent-bit", "--plugin", "/syslog-plugin/out_syslog.so", "--config", "/fluent-bit/etc/fluent-bit.conf"]
