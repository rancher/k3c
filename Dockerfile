ARG GOLANG=golang:1.15-alpine
FROM ${GOLANG} AS build
RUN apk --no-cache add \
    file \
    gcc \
    git \
    libseccomp-dev \
    libselinux-dev \
    make \
    musl-dev \
    protobuf-dev \
    protoc
RUN GO111MODULE=on go get github.com/gogo/protobuf/protoc-gen-gofast@v1.3.2
COPY . /go/src/k3c
WORKDIR /go/src/k3c
ENV CGO_ENABLED=1
ENV GO_BUILDTAGS="seccomp,selinux,static_build,netgo,osusergo"
RUN go mod vendor
RUN go generate -x
#RUN go build -ldflags '-w -linkmode=external -extldflags=-static' -tags="${GO_BUILDTAGS}" -o /usr/local/bin/k3c .
RUN go build -ldflags '-w -extldflags=-static' -tags="${GO_BUILDTAGS}" -o /usr/local/bin/k3c .
RUN file /usr/local/bin/k3c

FROM scratch AS release
COPY --from=build /usr/local/bin/k3c /bin/k3c
ENTRYPOINT ["k3c"]
CMD ["--help"]
