package run

import (
	"fmt"

	"github.com/rancher/k3c/cmd/attach"
	"github.com/rancher/k3c/cmd/create"
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/urfave/cli/v2"
)

type Run struct {
	create.Create

	D_Detach bool `usage:"Run container in background and print container ID"`
}

func (r *Run) Run(ctx *cli.Context) error {
	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	id, err := r.Create.Create(ctx, client, !r.D_Detach && r.I_Interactive)
	if err != nil {
		return err
	}

	if r.D_Detach {
		err := client.StartContainer(ctx.Context, id)
		fmt.Println(id)
		return err
	}

	return attach.RunAttach(ctx.Context, client, id, nil)
}
