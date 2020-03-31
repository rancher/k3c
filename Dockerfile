FROM golang:1.13-alpine as build
RUN apk -U --no-cache add bash git gcc musl-dev docker vim less file curl wget ca-certificates jq linux-headers zlib-dev tar zip squashfs-tools npm coreutils \
    python3 py3-pip python3-dev openssl-dev libffi-dev libseccomp libseccomp-dev make libuv-static
RUN curl -sL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v1.15.0
COPY . /go/src/github.com/rancher/k3c
ARG MAKE=build
ENV GO111MODULE=off
RUN cd /go/src/github.com/rancher/k3c && \
    mkdir -p bin dist && \
    make $MAKE

FROM scratch as make
COPY --from=build /go/src/github.com/rancher/k3c/dist/ /dist
COPY --from=build /go/src/github.com/rancher/k3c/bin/ /bin

FROM rancher/k3s:v1.17.3-k3s1 as data-base
RUN rm -rf etc/strongswan var run lib \
    bin/swanctl \
    bin/charon \
    bin/containerd \
    bin/kubectl \
    bin/k3s-server \
    bin/k3s-agent \
    bin/k3s \
    bin/ctr \
    bin/crictl \
 || true

FROM scratch as data
COPY --from=data-base /bin/ /bin

FROM scratch as bin
COPY --from=build /go/src/github.com/rancher/k3c/bin/k3c /bin/k3c

FROM scratch
COPY --from=bin /bin/ /bin
COPY --from=data-base /bin/ /bin
COPY --from=data-base /etc/ /etc
COPY --from=data-base /tmp/ /tmp
VOLUME /var/lib/rancher/k3c
CMD ["/bin/k3c", "daemon", "--bridge-cidr=172.19.0.0/16"]
