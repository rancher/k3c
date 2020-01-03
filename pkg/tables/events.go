package tables

import (
	"github.com/rancher/k3c/pkg/table"
	"github.com/urfave/cli/v2"
)

type EventData struct {
	ID   string
	Name string
}

func NewEvents(cli *cli.Context) table.Writer {
	cols := [][]string{
		{"", "{{.Name}} {{.ID}}"},
	}

	w := table.NewWriter(cols, config(cli, ""))
	return w
}
