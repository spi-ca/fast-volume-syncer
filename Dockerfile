# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:alpine AS build
LABEL org.opencontainers.image.authors="Sangbum Kim <sangbumkim@amuz.es>"

WORKDIR /build

COPY internal internal/
COPY main.go go.mod go.sum ./

RUN set -xeu && \
    apk --no-cache add \
    git \
    curl \
    binutils

RUN set -xe && \
    go mod download && \
    go mod verify && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go run mvdan.cc/garble@latest -seed=OutbmGbyoPRHgSVanyVcRg build . && \
    strip -S -x fast-volume-syncer && \
    strip -sxX \
        --remove-section=.bss\
        --remove-section=.comment\
        --remove-section=.eh_frame\
        --remove-section=.eh_frame_hdr\
        --remove-section=.fini\
        --remove-section=.fini_array\
        --remove-section=.gnu.build.attributes\
        --remove-section=.gnu.hash\
        --remove-section=.gnu.version\
        --remove-section=.gosymtab\
        --remove-section=.got\
        --remove-section=.note.ABI-tag\
        --remove-section=.note.gnu.build-id\
        --remove-section=.note.go.buildid\
        --remove-section=.shstrtab\
        --remove-section=.typelink \
        fast-volume-syncer && \
    mkdir -p v && \
    curl -L "https://packages.timber.io/vector/0.32.1/vector-0.32.1-x86_64-unknown-linux-gnu.tar.gz"  | \
    tar -C v --strip-components 3 -zxv ./vector-x86_64-unknown-linux-gnu/bin/vector

##
## Deploy
##
FROM alpine:3.18
LABEL maintainer="Sangbum Kim<sangbumkim@amuz.es>"
COPY --from=build /build/fast-volume-syncer /usr/local/bin/fast-volume-syncer
COPY --from=build /build/v/vector /usr/local/bin/

ARG UID=1000
ARG GID=1000

RUN set -xeu && \
    apk --no-cache add \
    strace \
    lsof \
    tini \
    htop \
    strace \
    gcompat \
    sudo \
    && \
    apk --purge del \
    apk-tools \
    && \
    rm -rvf \
    /etc/apk        \
    /sbin/apk       \
    /var/cache/apk  \
    /lib/apk/db \
    /lib/libapk.so* \
    /usr/share/apk

RUN set -xeu && \
    mkdir -p "/home/bc-user" && \
    adduser \
    -h "/home/spi-ca" \
    -g "spi-ca" \
    -s /bin/bash \
    -D \
    -u $UID \
    "spi-ca" && \
    echo 'spi-ca ALL=(root) NOPASSWD:ALL' > /etc/sudoers.d/spi-ca && \
    chmod 0440 "/etc/sudoers.d/spi-ca" && \
    chown -R spi-ca:spi-ca "/home/spi-ca"

USER spi-ca:spi-ca
WORKDIR /home/spi-ca
ENV PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
ENTRYPOINT [ "/sbin/tini", "-s", "--", "fast-volume-syncer" ]