package tag

import (
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/urfave/cli/v2"
)

type Tag struct {
}

func (t *Tag) Run(ctx *cli.Context) error {
	if err := validate.NArgs(ctx, "tag", 2, -1); err != nil {
		return err
	}

	c, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	image := ctx.Args().First()
	return c.TagImage(ctx.Context, image, ctx.Args().Tail()...)
}
