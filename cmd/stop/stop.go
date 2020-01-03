package stop

import (
	"fmt"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/urfave/cli/v2"
)

type Stop struct {
	T_Time int `usage:"Seconds to wait for stop before killing it" default:"10"`
}

func (s *Stop) Run(ctx *cli.Context) error {
	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	var lastErr error
	for _, id := range ctx.Args().Slice() {
		if err := client.StopContainer(ctx.Context, id, int64(s.T_Time)); err != nil {
			fmt.Printf("Error: %v: %s\n", err, id)
			lastErr = err
		}
	}

	if lastErr != nil {
		return cli.Exit("", 1)
	}

	return nil
}
