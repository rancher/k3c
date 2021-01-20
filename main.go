//go:generate protoc --gofast_out=plugins=grpc:. -I=./vendor:. pkg/apis/services/images/v1alpha1/images.proto

package main

import (
	"github.com/containerd/containerd/pkg/seed"
	"github.com/rancher/k3c/pkg/cli"
	command "github.com/rancher/wrangler-cli"

	// Add non-default auth providers
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func init() {
	seed.WithTimeAndRand()
}

func main() {
	command.Main(cli.Main())
}
