PROG  	?= k3c
REPO  	?= rancher
IMAGE 	?= $(REPO)/k3c
PKG		?= github.com/rancher/k3c

GOLANGCI_VERSION ?= v1.25.1

ifneq "$(strip $(shell command -v go 2>/dev/null))" ""
	GOOS ?= $(shell go env GOOS)
	GOARCH ?= $(shell go env GOARCH)
else
	ifeq ($(GOOS),)
		# approximate GOOS for the platform if we don't have Go and GOOS isn't
		# set. We leave GOARCH unset, so that may need to be fixed.
		ifeq ($(OS),Windows_NT)
			GOOS = windows
		else
			UNAME_S := $(shell uname -s)
			ifeq ($(UNAME_S),Linux)
				GOOS = linux
			endif
			ifeq ($(UNAME_S),Darwin)
				GOOS = darwin
			endif
			ifeq ($(UNAME_S),FreeBSD)
				GOOS = freebsd
			endif
		endif
	else
		GOOS ?= $$GOOS
		GOARCH ?= $$GOARCH
	endif
endif

_GO_ENV := GOOS=$(GOOS) GOARCH=$(GOARCH)
ifeq ($(GOARCH),arm)
	ifndef GOARM
		GOARM ?= 7
	endif
	_GO_ENV += GOARM=$(GOARM)
endif
GO := $(_GO_ENV) go

ifndef GODEBUG
	EXTRA_LDFLAGS += -s -w
	DEBUG_GO_GCFLAGS :=
	DEBUG_TAGS :=
else
	DEBUG_GO_GCFLAGS := -gcflags=all="-N -l"
endif

ifndef GOBIN
	GOBIN := bin
endif

ifdef DRONE_TAG
	VERSION = ${DRONE_TAG}
else
	VERSION ?= $(shell git describe --match 'v[0-9]*' --dirty='.dirty' --always --tags)
endif
REVISION = $(shell git rev-parse HEAD)$(shell if ! git diff --no-ext-diff --quiet --exit-code; then echo .dirty; fi)
RELEASE = ${PROG}-${GOOS}-${GOARCH}
ifndef TAG
	TAG := $(shell echo "$(VERSION)" | tr '+' '-')-${GOARCH}
endif


ifdef BUILDTAGS
    GO_BUILDTAGS = ${BUILDTAGS}
else
	GO_BUILDTAGS = apparmor seccomp selinux netgo osusergo static_build no_btrfs
endif
GO_BUILDTAGS += ${DEBUG_TAGS}
GO_TAGS=$(if $(GO_BUILDTAGS),-tags "$(GO_BUILDTAGS)",)

GO_EXTLDFLAGS ?= -fno-PIC -static
GO_LDFLAGS ?= $(EXTRA_LDFLAGS)
GO_LDFLAGS += -X ${PKG}/pkg/version.Version=$(VERSION)
GO_LDFLAGS += -X ${PKG}/pkg/version.GitCommit=$(REVISION)
GO_LDFLAGS += -X ${PKG}/pkg/daemon/config.DefaultBootstrapImage=docker.io/${IMAGE}:${TAG}
GO_LDFLAGS += -X github.com/containerd/containerd/version.Package=$(shell grep 'github.com/containerd/containerd' go.mod | head -n1 | awk '{print $$3}')
GO_LDFLAGS += -X github.com/containerd/containerd/version.Revision=$(shell grep 'github.com/containerd/containerd' go.mod | head -n1 | awk '{print $$4}')
GO_LDFLAGS += -X github.com/containerd/containerd/version.Version=$(shell grep 'github.com/containerd/containerd' go.mod | head -n1 | awk '{print $$6}')
GO_LDFLAGS += -extldflags '${GO_EXTLDFLAGS}'

default: in-docker-build                 ## Build using docker environment (default target)
	@echo "Run make help for info about other make targets"

ci: in-docker-.ci                        ## Run CI locally

ci-shell: clean .dapper                  ## Launch a shell in the CI environment to troubleshoot. Runs clean first
	@echo
	@echo '######################################################'
	@echo '# Run "make dapper-ci" to reproduce CI in this shell #'
	@echo '######################################################'
	@echo
	./.dapper -f Dockerfile --target dapper -s

dapper-ci: .ci                           ## Used by Drone CI, does the same as "ci" but in a Drone way

build:                                   ## Build using host go tools
	$(GO) build ${DEBUG_GO_GCFLAGS} ${GO_GCFLAGS} ${GO_BUILDFLAGS} -o ${GOBIN}/${PROG} -ldflags "${GO_LDFLAGS}" ${GO_TAGS}

build-debug:                             ## Debug build using host go tools
	$(MAKE) GODEBUG=y build

package:                                 ## Build final docker image for push
	docker build --build-arg ARCH=${GOARCH} --tag ${IMAGE}:${TAG} .

bin/golangci-lint:
	curl -fsL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s ${GOLANGCI_VERSION}

validate:                                ## Run go fmt/vet
	$(GO) fmt ./...
	$(GO) vet ./...

validate-ci: validate bin/golangci-lint  ## Run more validation for CI
	[ "${GOARCH}" != "amd64" ] || ./bin/golangci-lint run

run: build-debug
	./bin/${PROG} server

bin/dlv:
	$(GO) build -o bin/dlv github.com/go-delve/delve/cmd/dlv

remote-debug: build-debug bin/dlv        ## Run with remote debugging listening on :2345
	./bin/dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./bin/${PROG} server

.dev-shell-build:
	docker build -t ${PROG}-dev --target shell -f Dockerfile --target dapper .

clean-cache:                             ## Clean up docker base caches used for development
	docker rm -fv ${PROG}-dev-shell
	docker volume rm ${PROG}-cache ${PROG}-pkg

clean:                                   ## Clean up workspace
	rm -rf bin dist

dev-shell: .dev-shell-build              ## Launch a development shell to run test builds
	docker run --rm --name ${PROG}-dev-shell -ti -v $${HOME}:$${HOME} -v ${PROG} -w $$(pwd) --privileged --net=host -v ${PROG}-pkg:/go/pkg -v ${PROG}-cache:/root/.cache/go-build ${PROG}-dev bash

dev-shell-enter:                         ## Enter the development shell on another terminal
	docker exec -it ${PROG}-dev-shell bash

artifacts: build
	mkdir -p dist/artifacts
	cp ${GOBIN}/${PROG} dist/artifacts/${RELEASE}
	cp ${GOBIN}/${PROG} bin/${PROG}

.ci: validate-ci artifacts

in-docker-%: .dapper                     ## Advanced: wraps any target in Docker environment, for example: in-docker-build-debug
	mkdir -p bin/ dist/
	./.dapper -f Dockerfile --target dapper make $*

./.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/v0.5.0/dapper-$$(uname -s)-$$(uname -m) > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
