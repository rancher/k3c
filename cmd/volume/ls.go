package volume

import (
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/tables"
	"github.com/urfave/cli/v2"
)

type Ls struct {
	Format  string `usage:"Pretty-print volumes using a Go template"`
	Q_Quiet bool   `usage:"Only display volume names"`
}

func (l *Ls) Run(ctx *cli.Context) error {
	c, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	volumes, err := c.ListVolumes(ctx.Context)
	if err != nil {
		return err
	}

	t := tables.NewVolumes(ctx)

	for _, volume := range volumes {
		t.Write(tables.VolumeData{
			Volume: volume,
		})
	}

	return t.Close()
}
