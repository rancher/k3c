package cliclient

import (
	"os"

	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/client/build"
	"github.com/urfave/cli/v2"
)

func New(cli *cli.Context) (client.Client, error) {
	return client.New(cli.Context, os.Getenv("K3C_ADDRESS"))
}

func NewBuilder(cli *cli.Context) (build.Client, error) {
	return build.New(cli.Context, os.Getenv("K3C_ADDRESS"))
}
