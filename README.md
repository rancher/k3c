k3c - Classic Docker (Build) for a Kubernetes world
===========================================

***STATUS: EXPERIMENT - Let us know what you think***

***NOTE: the original experiment started on `master` while the next-gen work will be on `main`***

`k3c` brings the Classic &trade; Docker images manipulation UX to your
`k3s` development workflow. It is designed to enable the rapid feedback
when developing and testing local container images in `k3s` and `rke2`.
Currently `k3s`, the [lightweight Kubernetes distribution](https://github.com/k3s-io/k3s),
provides a great solution for Kubernetes from dev to production.  While
`k3s` satisifies the Kubernetes runtime needs, one still needs to run
`docker` (or a docker-like tool) to actually develop and build the container
images.  `k3c` is intended to replace `docker` for just the functionality
needed for building and manipulating images in the Kubernetes ecosystem.

## A familiar UX

There really is nothing better than the classic Docker UX of `build/push/pull/tag`.
This tool copies the same UX as classic Docker (think Docker v1.12). The intention
is to follow the same style but not be a 100% drop in replacement.  Behaviour and
arguments have been changed to better match the behavior of the Kubernetes ecosystem.

## A single binary

`k3c`, similar to `k3s` and old school docker, is packaged as a single binary, because nothing
is easier for distribution than a static binary.

## Built on Kubernetes Tech (and others)

Fundamentally `k3c` is a built on the [Container Runtime Interface (CRI)](https://github.com/kubernetes/cri-api),
[containerd](https://github.com/containerd/containerd), and [buildkit](https://github.com/moby/buildkit).

## Architecture

`k3c` enables building `k3s`-local images by installing a DaemonSet Pod that runs both `buildkitd` and `k3c agent`
and exposing the gRPC endpoints for these active agents in your cluster via a Service. Once installed, the `k3c` CLI
can inspect your installation and communicate with the backend daemons for image building and manipulation with merely
the KUBECONFIG that was available when invoking `k3c install`. When building `k3c` will talk directly to the `buildkit`
service but all other interactions with the underlying containerd/CRI are mediated by the `k3c agent` (primarily
because the `containerd` client code assumes a certain level of co-locality with the `containerd` installation).

## Building

```bash
# more to come on this front but builds are currently a very manual affair
# git clone --branch=trunk https://github.com/rancher/k3c.git ~/Projects/rancher/k3c
# cd ~/Projects/rancher/k3c
go generate # only necessary when modifying the gRPC protobuf IDL, see Dockerfile for pre-reqs
go build -ldflags '-w -extldflags=-static' -tags="seccomp,selinux,static_build,netgo,osusergo" .
# only needed until @dweomer gets the automated builds publishing again
docker build --tag your/image:tag .
docker push your/image:tag
```

## Running

Have a working `k3s` installation with a working `$HOME/.kube/config` or `$KUBECONFIG`, then:

```bash
# Installation on a single-node cluster
./k3c install --agent-image=docker.io/your/image:tag
```

```bash
# Installation on a multi-node cluster, targeting a Node named "my-builder-node"
./k3c install --agent-image=docker.io/your/image:tag --selector k3s.io/hostname=my-builder-node

```

`k3c` currently works against a single builder node so you must specify a narrow selector when
installing on multi-node clusters. Upon successful installation this node will acquire the "builder" role.

Build images like you would with `docker`

```
$ ./k3c --help
Usage:
  k3c [flags]
  k3c [command]

Available Commands:
  build       Build an image
  help        Help about any command
  images      List images
  install     Install builder component(s)
  pull        Pull an image
  push        Push an image
  rmi         Remove an image
  tag         Tag an image
  uninstall   Uninstall builder component(s)

Flags:
  -x, --context string      kubeconfig context for authentication
      --debug               
      --debug-level int     
  -h, --help                help for k3c
  -k, --kubeconfig string   kubeconfig for authentication
  -n, --namespace string    namespace (default "k3c")
  -v, --version             version for k3c

Use "k3c [command] --help" for more information about a command.
```

## Roadmap

- Automated builds for clients on MacOS (amd64/arm64), Windows (amd64), and Linux client/server (amd64/arm64/arm).

# License

Copyright (c) 2020-2021 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

