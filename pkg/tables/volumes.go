package tables

import (
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/table"
	"github.com/urfave/cli/v2"
)

type VolumeData struct {
	Volume client.Volume
}

func NewVolumes(cli *cli.Context) table.Writer {
	cols := [][]string{
		{"VOLUME NAME", "Volume.ID"},
	}

	w := table.NewWriter(cols, config(cli, "VOLUME NAME"))
	return w
}
