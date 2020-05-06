# Build Environment
FROM golang:1.13-buster AS build
ARG GOBIN=/opt/k3c/bin
ARG GOPATH=/go
RUN apt-get update --assume-yes \
 && apt-get install --assume-yes \
    gcc \
    git \
    libapparmor-dev \
    libseccomp-dev \
    libselinux1-dev \
    make

# Debug Environment
FROM build AS shell
ARG GOBIN=/opt/k3c/bin
ARG GOPATH=/go
ENV GOBIN=${GOBIN} \
    GOPATH=${GOPATH} \
    EDITOR=vim \
    PAGER=less
RUN set -x \
 && export DEBIAN_FRONTEND=noninteractive \
 && apt-get update --assume-yes \
 && apt-get install --assume-yes \
    file \
    iptables \
    less \
    socat \
    vim
VOLUME /tmp
VOLUME /var/lib/cni
VOLUME /var/lib/rancher/k3c
VOLUME /var/log

# Dapper/Drone/CI environment
FROM build AS dapper
ARG GOBIN=/opt/k3c/bin
ARG GOFLAGS=" -mod=vendor"
ARG GOPATH=/go
ENV GOBIN=${GOBIN} \
    GOFLAGS=${GOFLAGS} \
    GOPATH=${GOPATH}

ENV DAPPER_ENV REPO TAG DRONE_TAG
ENV DAPPER_OUTPUT ./bin ./dist
ENV DAPPER_DOCKER_SOCKET true
ENV DAPPER_TARGET dapper
ENV DAPPER_RUN_ARGS "-v k3c-pkg:/go/pkg -v k3c-cache:/root/.cache/go-build"
WORKDIR /source

# Build: CNI plugins
FROM build AS cni
ARG CGO_ENABLED=0
ARG GOBIN=/opt/k3c/bin
ARG GOPATH=/go
ARG CNIPLUGINS_VERSION="v0.7.6-k3s1"
ARG CNIPLUGINS_PACKAGE=github.com/rancher/plugins
ARG CNIPLUGINS_LDFLAGS="-w -s"
ARG CNIPLUGINS_EXTRA_LDFLAGS="-static"
RUN git clone -b "${CNIPLUGINS_VERSION}" "https://${CNIPLUGINS_PACKAGE}.git" ${GOPATH}/src/github.com/containernetworking/plugins
RUN go build -o ${GOBIN}/cni  -ldflags "${CNIPLUGINS_LDFLAGS} -extldflags '${CNIPLUGINS_EXTRA_LDFLAGS}'" github.com/containernetworking/plugins
RUN set -x \
 && ${GOBIN}/cni --version
RUN ln -sv cni ${GOBIN}/bridge
RUN ln -sv cni ${GOBIN}/flannel
RUN ln -sv cni ${GOBIN}/host-local
RUN ln -sv cni ${GOBIN}/loopback
RUN ln -sv cni ${GOBIN}/portmap

# Build: runC
FROM build AS runc
ARG GOBIN=/opt/k3c/bin
ARG GOPATH=/go
ARG RUNC_VERSION="v1.0.0-rc10"
ARG RUNC_PACKAGE=github.com/opencontainers/runc
ARG RUNC_BUILDTAGS="apparmor seccomp selinux"
RUN git clone -b "${RUNC_VERSION}" "https://${RUNC_PACKAGE}.git" ${GOPATH}/src/github.com/opencontainers/runc
WORKDIR ${GOPATH}/src/github.com/opencontainers/runc
ENV GO111MODULE=off
RUN set -x \
 && make BUILDTAGS="${RUNC_BUILDTAGS}" static
RUN set -x \
 && make PREFIX="$(dirname ${GOBIN})" BINDIR="${GOBIN}" install

# Build: containerd + cri
FROM build AS containerd
ARG GOBIN=/opt/k3c/bin
ARG GOPATH=/go
ARG CONTAINERD_VERSION="v1.3.4+k3c.1"
ARG CONTAINERD_PACKAGE=github.com/dweomer/containerd
ARG CONTAINERD_BUILDTAGS="apparmor seccomp selinux netgo osusergo static_build no_btrfs"
ARG CONTAINERD_EXTRA_FLAGS="-buildmode pie"
ARG CONTAINERD_EXTRA_LDFLAGS='-w -s -extldflags "-fno-PIC -static"'
RUN git clone -b "${CONTAINERD_VERSION}" "https://${CONTAINERD_PACKAGE}.git" ${GOPATH}/src/github.com/containerd/containerd
WORKDIR ${GOPATH}/src/github.com/containerd/containerd
ENV GO111MODULE=off
RUN set -x \
 && make \
    BUILDTAGS="${CONTAINERD_BUILDTAGS}" \
    EXTRA_FLAGS="${CONTAINERD_EXTRA_FLAGS}" \
    EXTRA_LDFLAGS="${CONTAINERD_EXTRA_LDFLAGS}" \
    PACKAGE="${CONTAINERD_PACKAGE}" \
    VERSION="${CONTAINERD_VERSION}" \
    binaries
RUN set -x \
 && make DESTDIR="$(dirname ${GOBIN})" \
    install

# Unpack: k3s-root
FROM build AS k3s
ARG ARCH
ARG ROOT_VERSION=v0.4.1
RUN set -x \
 && mkdir -p /srv/etc \
 && echo 'hosts: files dns' > /srv/etc/nsswitch.conf
COPY --from=build   /etc/ssl/                       /srv/etc/ssl/
COPY --from=build   /usr/share/ca-certificates/     /srv/usr/share/ca-certificates/
RUN set -x \
 && ln -s certs/ca-certificates.crt /srv/etc/ssl/cert.pem
ADD https://github.com/rancher/k3s-root/releases/download/${ROOT_VERSION}/k3s-root-${ARCH}.tar /tmp/root.tar
RUN set -x \
 && tar -xvf /tmp/root.tar -C /srv

FROM build AS gather
ARG GOBIN=/opt/k3c/bin
COPY --from=k3s         /srv/                       /srv/
COPY --from=cni         ${GOBIN}/                   /srv/bin/
COPY --from=runc        ${GOBIN}/                   /srv/bin/
COPY --from=containerd  ${GOBIN}/containerd-shim*   /srv/bin/
COPY bin/k3c    /srv/usr/bin/
COPY Dockerfile /srv/usr/share/k3c/Dockerfile

FROM scratch as release
COPY --from=gather /srv/ /
VOLUME /tmp
VOLUME /var/lib/cni
VOLUME /var/lib/rancher/k3c
VOLUME /var/log
ENTRYPOINT ["k3c"]
CMD ["--help"]
