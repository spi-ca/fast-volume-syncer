# syntax=docker/dockerfile:1

##
## Build
##
FROM public.ecr.aws/docker/library/golang:1.19.5 AS build

WORKDIR /build

COPY internal internal/
COPY cmd cmd/
COPY go.mod go.sum ./

RUN set -xe && \
    go mod download && \
    go mod verify && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o fixturetool ./cmd/fixturetool && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o voltest ./cmd/voltest

##
## Deploy
##
FROM public.ecr.aws/docker/library/debian:sid-slim
LABEL maintainer="Sangbum Kim <sangbumkim@amuz.es>"
COPY --from=build /build/fixturetool /build/voltest ./
COPY deployment /deployment
RUN set -xe && \
    apt update && \
    DEBIAN_FRONTEND=noninteractive \
    apt install -y \
                iproute2 \
                nano \
                time \
                byobu \
                iperf3 \
                iptraf-ng \
                fio \
                htop && \
    apt install -y \
                ca-certificates curl \
                apt-transport-https && \
    sh -c 'curl -fsSLo /etc/apt/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg' && \
    sh -c 'echo "deb [signed-by=/etc/apt/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list' && \
    apt update && \
    DEBIAN_FRONTEND=noninteractive \
    apt install -y \
      kubectl

USER nobody:nogroup
ENTRYPOINT ["/bin/sleep", "infinity"]