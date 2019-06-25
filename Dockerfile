FROM oratos/golang-base:1.11 as gobuilder

WORKDIR /root

ENV GOOS=linux \
    GOARCH=amd64

COPY / /root/

RUN go build \
    -a \
    -installsuffix fluent \
    -buildmode c-shared \
    -o /out_syslog.so \
    -mod=readonly \
    -mod=vendor \
    cmd/main.go

FROM ubuntu:xenial as builder

# Fluent Bit version
ENV FLB_MAJOR 1
ENV FLB_MINOR 1
ENV FLB_PATCH 3
ENV FLB_VERSION 1.1.3

ENV DEBIAN_FRONTEND noninteractive

ENV FLB_TARBALL http://github.com/fluent/fluent-bit/archive/v$FLB_VERSION.zip

RUN mkdir -p /fluent-bit/bin /fluent-bit/etc /fluent-bit/log /tmp/src/

RUN apt-get update \
    && apt-get dist-upgrade -y \
    && apt-get install -y \
       build-essential \
       cmake \
       make \
       wget \
       unzip \
       libsystemd-dev \
       libssl-dev \
       libasl-dev \
       libsasl2-dev \
       flex \
       bison \
       git \
    && wget -O "/tmp/fluent-bit-${FLB_VERSION}.zip" ${FLB_TARBALL} \
    && cd /tmp && unzip "fluent-bit-$FLB_VERSION.zip" \
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
    && install bin/fluent-bit /fluent-bit/bin/

# Configuration files
COPY /config/fluent-bit.conf \
     /config/parsers.conf \
     /config/parsers_java.conf \
     /config/parsers_mult.conf \
     /config/parsers_openstack.conf \
     /config/parsers_cinder.conf \
     /fluent-bit/etc/

FROM ubuntu:xenial

RUN apt-get update \
    && apt-get dist-upgrade -y \
    && apt-get install --no-install-recommends ca-certificates libssl1.0.2 -y libsasl2-2 wget curl \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get autoclean

COPY --from=builder /fluent-bit /fluent-bit
COPY --from=gobuilder /out_syslog.so /fluent-bit/bin/

EXPOSE 2020

#CMD ["/fluent-bit/bin/fluent-bit", "--plugin", "/fluent-bit/bin/out_syslog.so", "--config", "/fluent-bit/etc/fluent-bit.conf"]
