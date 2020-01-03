package logs

import (
	"os"

	timetypes "github.com/docker/docker/api/types/time"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/urfave/cli/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Logs struct {
	F_Follow     bool   `usage:"Follow log output"`
	Since        string `usage:"Show logs since timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes)"`
	Tail         int    `usage:"Number of lines to show from the end of the logs (default 'all')"`
	T_Timestamps bool   `usage:"Show timestamps"`
}

func (l *Logs) Run(ctx *cli.Context) error {
	if err := validate.NArgs(ctx, "logs", 1, 1); err != nil {
		return err
	}

	c, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	var since *metav1.Time
	if l.Since != "" {
		s, n, err := timetypes.ParseTimestamps(l.Since, 0)
		if err != nil {
			return errors.Wrapf(err, "parsing %s", l.Since)
		}
		t := metav1.Unix(s, n)
		since = &t
	}

	var tail *int64
	if l.Tail > 0 {
		v := int64(l.Tail)
		tail = &v
	}

	lines, err := c.LogContainer(ctx.Context, ctx.Args().First(), &v1.PodLogOptions{
		Follow:     l.F_Follow,
		SinceTime:  since,
		Timestamps: l.T_Timestamps,
		TailLines:  tail,
	})
	if err != nil {
		return err
	}

	for msg := range lines {
		if msg.Stderr {
			os.Stderr.Write(msg.Message)
		} else {
			os.Stdout.Write(msg.Message)
		}
	}

	return nil
}
