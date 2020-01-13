package inspect

import (
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/tables"
	"github.com/urfave/cli/v2"
)

type Inspect struct {
	Format string `usage:"Pretty-print images using a Go template" default:"json"`
}

func (i *Inspect) Run(ctx *cli.Context) error {
	c, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	t := tables.NewInspect(ctx)

	for _, arg := range ctx.Args().Slice() {
		container, _, _, err := c.GetContainer(ctx.Context, arg)
		if err == nil {
			t.Write(container)
			continue
		}

		image, err := c.GetImage(ctx.Context, arg)
		if err == nil {
			t.Write(image)
		}
	}

	return t.Close()
}
