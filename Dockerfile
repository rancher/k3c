FROM golang:1.13.4-alpine3.10 as build
RUN apk -U --no-cache add bash git gcc musl-dev docker vim less file curl wget ca-certificates jq linux-headers zlib-dev tar zip squashfs-tools npm coreutils \
    python3 py3-pip python3-dev openssl-dev libffi-dev libseccomp libseccomp-dev make libuv-static
COPY . /go/src/github.com/rancher/k3c
ENV GO111MODULE=off
RUN cd /go/src/github.com/rancher/k3c && \
    go build -ldflags "-extldflags -static -s" -o /bin/k3c main.go

FROM rancher/k3s:v1.0.0 as data
RUN rm -rf etc/strongswan var run lib \
    bin/swanctl \
    bin/charon \
    bin/containerd \
    kubectl \
    k3s-server \
    k3s-agent \
    k3s \
    ctr \
    crictl

FROM scratch as bin
COPY --from=build /bin/k3c /bin/k3c

FROM scratch
COPY --from=bin /bin/ /bin
COPY --from=data /bin/ /bin
COPY --from=data /etc/ /etc
COPY --from=data /tmp/ /tmp
VOLUME /var/lib/rancher/k3c
CMD ["/bin/k3c","daemon", "--bridge-cidr=172.19.0.0/16"]
