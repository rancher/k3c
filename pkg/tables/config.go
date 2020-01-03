package tables

import (
	"io"
	"os"

	"github.com/rancher/k3c/pkg/table"
	"github.com/urfave/cli/v2"
)

func config(cli *cli.Context, idColumn string) table.WriterConfig {
	return &writerConfig{
		cli:      cli,
		idColumn: idColumn,
	}
}

type writerConfig struct {
	cli      *cli.Context
	idColumn string
}

func (w *writerConfig) Quiet() bool {
	return w.cli.Bool("quiet")
}

func (w *writerConfig) NoTrunc() bool {
	return w.cli.Bool("no-trunc")
}

func (w *writerConfig) Format() string {
	return w.cli.String("format")
}

func (w *writerConfig) Writer() io.Writer {
	return os.Stdout
}

func (w *writerConfig) IDColumn() string {
	return w.idColumn
}
