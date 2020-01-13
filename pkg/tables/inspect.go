package tables

import (
	"github.com/rancher/k3c/pkg/table"
	"github.com/urfave/cli/v2"
)

func NewInspect(cli *cli.Context) table.Writer {
	cols := [][]string{
		{"", "{{. | json}}"},
	}

	w := table.NewWriter(cols, config(cli, ""))
	return w
}
