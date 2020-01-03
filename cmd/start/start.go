package start

import (
	"fmt"

	"github.com/rancher/k3c/cmd/attach"
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/urfave/cli/v2"
)

type Start struct {
	A_Attach      bool `usage:"Attach STDOUT/STDERR and forward signals"`
	I_Interactive bool `usage:"Attach container's STDIN"`
}

func (s *Start) Run(ctx *cli.Context) error {
	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	var lastErr error
	for _, id := range ctx.Args().Slice() {
		if err := client.StartContainer(ctx.Context, id); err != nil {
			fmt.Printf("Error: %v: %s\n", err, id)
			lastErr = err
		}
	}

	if lastErr != nil {
		return cli.Exit("", 1)
	}

	if s.A_Attach && ctx.NArg() > 0 {
		return attach.RunAttach(ctx.Context, client, ctx.Args().First(), &v1alpha1.AttachOptions{
			NoStdin: !s.I_Interactive,
		})
	}

	return nil
}
