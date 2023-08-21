# syntax=docker/dockerfile:1

##
## Build
##
#FROM docker.io/library/golang:1.21.0-alpine AS build
FROM public.ecr.aws/docker/library/golang:1.21.0-alpine AS build

WORKDIR /build

COPY internal internal/
COPY main.go go.mod go.sum ./

RUN set -xe && \
    go mod download && \
    go mod verify && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o fast-volume-syncer .

##
## Deploy
##
#FROM docker.io/library/alpine:3.14
FROM public.ecr.aws/docker/library/alpine:3.14
LABEL maintainer="Sangbum Kim <sangbumkim@amuz.es>"
COPY --from=build /build/fast-volume-syncer /usr/local/bin/fast-volume-syncer

ARG UID=1111
ARG GID=1111

COPY --from=build /opt/gitea /opt/gitea
WORKDIR /opt/gitea

RUN set -x && \
    apk --no-cache add \
    bash \
    nano \
    ca-certificates \
    tini \
    gettext \
    git \
    curl \
    gnupg && \
    addgroup -g $GID bc-user && \
    adduser -S -D -h /home/bc-user -s /bin/bash -G bc-user -u $GID bc-user && \
    apk --purge del apk-tools &&\
    rm -rvf \
        /etc/apk        \
        /sbin/apk       \
        /var/cache/apk  \
        /usr/share/apk \

USER bc-user:bc-user
ENV PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
ENTRYPOINT [ "/sbin/tini", "-s", "--", "fast-volume-syncer" ]
