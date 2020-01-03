package volume

import (
	"fmt"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/urfave/cli/v2"
)

type Rm struct {
	F_Force bool `usage:"Force the removal of one or more volumes"`
}

func (r *Rm) Run(ctx *cli.Context) error {
	c, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	success := true
	for _, v := range ctx.Args().Slice() {
		err := c.RemoveVolume(ctx.Context, v, r.F_Force)
		if err == nil {
			fmt.Println(v)
		} else {
			success = false
			fmt.Printf("Error: %v: %s\n", err, v)
		}
	}

	if !success {
		return cli.Exit("", 1)
	}

	return nil
}
