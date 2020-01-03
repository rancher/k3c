package cliclient

import (
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/client/build"
	"github.com/urfave/cli/v2"
)

func New(cli *cli.Context) (client.Client, error) {
	return client.New(cli.Context, "")
}

func NewBuilder(cli *cli.Context) (build.Client, error) {
	return build.New(cli.Context, "")
}
