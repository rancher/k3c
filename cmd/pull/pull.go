package pull

import (
	"fmt"
	"os"

	"github.com/rancher/k3c/cmd/create"
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/urfave/cli/v2"
)

type Pull struct {
	Q_Quiet bool `usage:"Suppress verbose output"`
}

func (p *Pull) Run(ctx *cli.Context) error {
	if err := validate.NArgs(ctx, "pull", 1, 1); err != nil {
		return err
	}

	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	out := os.Stdout
	if p.Q_Quiet {
		out = nil
	}

	id, err := create.PullImage(ctx.Context, client, ctx.Args().First(), out, true)
	if err != nil {
		return err
	}

	fmt.Println(id)
	return nil
}
