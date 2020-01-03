package exec

import (
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/rancher/k3c/pkg/stream"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/urfave/cli/v2"
)

type Exec struct {
	I_Interactive bool `usage:"Keep STDIN open even if not attached"`
	T_Tty         bool `usage:"Allocate a pseudo-TTY"`
}

func (e *Exec) Run(ctx *cli.Context) error {
	if err := validate.NArgs(ctx, "exec", 2, -1); err != nil {
		return err
	}

	c, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	container := ctx.Args().First()
	cmd := ctx.Args().Tail()

	resp, err := c.Exec(ctx.Context, container, cmd, &v1alpha1.ExecOptions{
		Tty:   e.T_Tty,
		Stdin: e.I_Interactive,
	})
	if err != nil {
		return err
	}

	return stream.Stream(resp.Stdin, resp.TTY, resp.URL)
}
