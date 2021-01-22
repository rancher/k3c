ifeq ($(GOARCH),)
GOARCH := $(shell go env GOARCH)
endif

ifeq ($(GOOS),)
GOOS := $(shell go env GOOS)
endif

ifneq ($(DRONE_TAG),)
TAG := $(DRONE_TAG)
endif

DOCKER_BUILDKIT ?= 1

ORG ?= rancher
PKG ?= github.com/rancher/k3c
TAG ?= v0.0.0-dev

ifeq ($(GO_BUILDTAGS),)
GO_BUILDTAGS := static_build,netgo,osusergo
#ifeq ($(GOOS),linux)
#GO_BUILDTAGS := $(GO_BUILDTAGS),seccomp,selinux
#endif
endif

GO_LDFLAGS ?= -w -extldflags=-static
GO_LDFLAGS += -X $(PKG)/pkg/version.GitCommit=$(shell git rev-parse HEAD)
GO_LDFLAGS += -X $(PKG)/pkg/version.Version=$(TAG)
GO_LDFLAGS += -X $(PKG)/pkg/server.DefaultAgentImage=docker.io/$(ORG)/k3c

GO ?= go
GOLANG ?= docker.io/library/golang:1.15-alpine
ifeq ($(GOOS),windows)
BINSUFFIX := .exe
endif
BIN ?= bin/k3c
BIN := $(BIN)$(BINSUFFIX)

.PHONY: build package validate ci publish
build: $(BIN)
package: | dist image-build
validate:
publish: | image-build image-push image-manifest
ci: | build package validate
.PHONY: $(BIN)
$(BIN):
	$(GO) build -ldflags "$(GO_LDFLAGS)" -tags "$(GO_BUILDTAGS)" -o $@ .

.PHONY: dist
dist:
	@mkdir -p dist/artifacts
	@make GOOS=$(GOOS) GOARCH=$(GOARCH) BIN=dist/artifacts/k3c-$(GOOS)-$(GOARCH)$(BINSUFFIX) -C .

.PHONY: clean
clean:
	rm -rf bin dist

.PHONY: image-build
image-build:
	DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) docker build \
		--build-arg GOLANG=$(GOLANG) \
		--build-arg ORG=$(ORG) \
		--build-arg PKG=$(PKG) \
		--build-arg TAG=$(TAG) \
		--tag $(ORG)/k3c:$(TAG) \
		--tag $(ORG)/k3c:$(TAG)-$(GOARCH) \
	.

.PHONY: image-push
image-push:
	docker push $(ORG)/k3c:$(TAG)-$(GOARCH)

.PHONY: image-manifest
image-manifest:
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create --amend \
		$(ORG)/k3c:$(TAG) \
		$(ORG)/k3c:$(TAG)-$(GOARCH)
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push \
		$(ORG)/k3c:$(TAG)

./.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/v0.5.0/dapper-$$(uname -s)-$$(uname -m) > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

dapper-%: .dapper
	@mkdir -p ./bin/ ./dist/
	env DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) ./.dapper -f Dockerfile --target dapper make $*