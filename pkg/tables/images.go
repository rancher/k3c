package tables

import (
	"encoding/json"

	"github.com/docker/go-units"
	"github.com/rancher/k3c/pkg/table"
	"github.com/urfave/cli/v2"
)

type ImageData struct {
	ID     string
	Tag    string
	Digest string
	Repo   string
	Size   uint64
}

func NewImages(cli *cli.Context, digests bool) table.Writer {
	cols := [][]string{
		{"REPOSITORY", "Repo"},
		{"TAG", "Tag"},
	}

	if digests {
		cols = append(cols, []string{"DIGEST", "{{.Digest}}"})
	}

	cols = append(cols,
		[]string{"IMAGE ID", "{{.ID | id}}"},
		[]string{"SIZE", "{{.Size | formatSize}}"},
	)

	w := table.NewWriter(cols, config(cli, "IMAGE ID"))
	w.AddFormatFunc("formatSize", formatSize)
	return w
}

func formatSize(size json.Number) (string, error) {
	i, err := size.Int64()
	if err != nil {
		return "", err
	}
	return units.HumanSizeWithPrecision(float64(i), 3), nil

}
