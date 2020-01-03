package push

import (
	"context"
	"os"

	"github.com/rancher/k3c/pkg/auth"
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/progress"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

type Push struct {
	Q_Quiet bool `usage:"Suppress verbose output"`
}

func (p *Push) Run(ctx *cli.Context) error {
	if err := validate.NArgs(ctx, "pull", 1, 1); err != nil {
		return err
	}

	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	eg := errgroup.Group{}
	subCtx, cancel := context.WithCancel(ctx.Context)
	defer eg.Wait()
	defer cancel()

	image := ctx.Args().First()

	eg.Go(func() error {
		if p.Q_Quiet {
			return nil
		}

		infos, err := client.PushProgress(subCtx, image)
		if err != nil {
			return err
		}

		return progress.Display(infos, os.Stdout)
	})

	return client.PushImage(ctx.Context, image, auth.Lookup(image))
}
