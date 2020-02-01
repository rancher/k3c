package wait

import (
	"strings"
	"time"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/tables"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/urfave/cli/v2"
	v1 "k8s.io/api/core/v1"
)

type Wait struct {
}

func (w *Wait) Run(ctx *cli.Context) error {
	var stoppedStatuses []v1.ContainerStatus

	if err := validate.NArgs(ctx, "wait", 1, -1); err != nil {
		return err
	}

	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	tick := time.Tick(500 * time.Millisecond)

	for _, containerID := range ctx.Args().Slice() {
		select {
		case <-tick:
			pod, _, _, err := client.GetContainer(ctx.Context, containerID)
			if err != nil {
				return err
			}
			for _, status := range pod.Status.ContainerStatuses {
				if !strings.Contains(status.ContainerID, containerID) {
					continue
				}

				if status.State.Terminated == nil {
					break
				}

				stoppedStatuses = append(stoppedStatuses, status)
			}

		}

	}
	t := tables.NewStoppedContainers(ctx)
	for _, status := range stoppedStatuses {
		t.Write(status)
	}
}
