# syntax=docker/dockerfile:1

##
## Build
##
FROM public.ecr.aws/docker/library/golang:1.20.4 AS build

WORKDIR /build

COPY internal internal/
COPY main.go selector.go syncer.go cmd/
COPY go.mod go.sum ./

RUN set -xe && \
    go mod download && \
    go mod verify && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o fast-volume-syncer . &&

##
## Deploy
##
FROM public.ecr.aws/docker/library/debian:sid-slim
LABEL maintainer="Sangbum Kim <sangbumkim@amuz.es>"
COPY --from=build /build/fast-volume-syncer ./
ENTRYPOINT ["/fast-volume-syncer"]