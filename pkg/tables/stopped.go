package tables

import (
	"github.com/rancher/k3c/pkg/table"
	"github.com/urfave/cli/v2"
	v1 "k8s.io/api/core/v1"
)

type ContainerStopped struct {
	Status v1.ContainerStatus
}

func NewStoppedContainers(cli *cli.Context) table.Writer {
	cols := [][]string{
		{"ContainerID", "ContainerStopped.ContainerID"},
		{"Exit Code", "ContainerStopped.State.Terminated.ExitCode"},
	}

	w := table.NewWriter(cols, config(cli, "VOLUME NAME"))
	return w
}
