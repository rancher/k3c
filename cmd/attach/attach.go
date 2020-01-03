package attach

import (
	"context"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/rancher/k3c/pkg/stream"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/urfave/cli/v2"
)

type Attach struct {
	NoStdin bool `usage:"Do not attach STDIN"`
}

func (a *Attach) Run(ctx *cli.Context) error {
	if err := validate.NArgs(ctx, "attach", 1, 1); err != nil {
		return err
	}

	c, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	return RunAttach(ctx.Context, c, ctx.Args().First(), &v1alpha1.AttachOptions{
		NoStdin: a.NoStdin,
	})
}

func RunAttach(ctx context.Context, c client.Client, containerID string, opts *v1alpha1.AttachOptions) error {
	resp, err := c.Attach(ctx, containerID, opts)
	if err != nil {
		return err
	}

	return stream.Stream(resp.Stdin, resp.TTY, resp.URL)
}
