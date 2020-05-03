DOCKER = "docker"

default: docker-ci package

docker-%:
	./scripts/wrap $*

build:
	./scripts/build

test:
	./scripts/test

validate:
	./scripts/validate

package:
	./scripts/package

clean:
	rm -rf bin dist

ci: build test validate

release: ci
