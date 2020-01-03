package volume

import (
	"fmt"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/urfave/cli/v2"
)

type Create struct {
}

func (r *Create) Run(ctx *cli.Context) error {
	if err := validate.NArgs(ctx, "volume create", 0, 1); err != nil {
		return err
	}

	c, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	v, err := c.CreateVolume(ctx.Context, ctx.Args().First())
	if err != nil {
		return err
	}

	fmt.Println(v.ID)
	return nil
}
