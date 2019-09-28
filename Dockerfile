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

# Install Go
ARG GOLANG_SOURCE=dl.google.com/go
RUN wget https://$GOLANG_SOURCE/go1.12.9.linux-amd64.tar.gz -O go.tar.gz && \
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
ENV FLB_SHA b3adad27582ed7db0338b699391ecc6bd3779c1f
ENV FLB_TARBALL https://github.com/pivotal/fluent-bit/archive/$FLB_SHA.zip

RUN mkdir -p /fluent-bit/bin /fluent-bit/etc /fluent-bit/log /tmp/src/ \
    && wget -O "/tmp/fluent-bit-$FLB_SHA.zip" ${FLB_TARBALL} \
    && cd /tmp && unzip "fluent-bit-$FLB_SHA.zip" \
    && cd "fluent-bit-$FLB_SHA"/build/ \
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


# Configuration files
COPY /config/fluent-bit.conf \
     /config/parsers.conf \
     /config/parsers_java.conf \
     /config/parsers_mult.conf \
     /config/parsers_openstack.conf \
     /config/parsers_cinder.conf \
     /fluent-bit/etc/

RUN dpkg -l > /builder-dpkg-list

RUN apt-get update \
    && apt-get dist-upgrade -y \
    && apt-get install --no-install-recommends ca-certificates libssl1.0.2 -y libsasl2-2 curl \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get autoclean

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

#CMD ["/fluent-bit/bin/fluent-bit", "--plugin", "/fluent-bit/bin/out_syslog.so", "--config", "/fluent-bit/etc/fluent-bit.conf"]
