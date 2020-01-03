package rm

import (
	"fmt"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/urfave/cli/v2"
)

type Rm struct {
	F_Force   bool `usage:"Force the removal of a running container (uses SIGKILL)"`
	V_Volumes bool `usage:"Remove the volumes associated with the container"`
}

func (r *Rm) Run(ctx *cli.Context) error {
	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	success := true
	for _, container := range ctx.Args().Slice() {
		if r.F_Force {
			// ignore error
			_ = client.StopContainer(ctx.Context, container, 0)
		}
		err := client.RemoveContainer(ctx.Context, container)
		if err == nil {
			fmt.Println(container)
		} else {
			success = false
			fmt.Printf("Error: %v: %s\n", err, container)
		}
	}

	if !success {
		return cli.Exit("", 1)
	}

	return nil
}
