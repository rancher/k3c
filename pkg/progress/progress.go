package progress

import (
	"io"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/containerd/containerd/cmd/ctr/commands/content"
	"github.com/containerd/containerd/pkg/progress"
	"github.com/rancher/k3c/pkg/status"
)

func Display(c <-chan []status.Info, out io.Writer) (err error) {
	start := time.Now()

	pw := progress.NewWriter(out)
	defer func() {
		pw.Flush()
		if err == nil {
			pw.Write([]byte("\n"))
		} else {
			pw.Write([]byte(err.Error()))
		}
		pw.Flush()
	}()

	for status := range c {
		pw.Flush()
		if len(status) == 0 {
			continue
		}

		var contentStatus []content.StatusInfo
		for _, status := range status {
			contentStatus = append(contentStatus, content.StatusInfo(status))
		}

		sort.Slice(contentStatus, func(i, j int) bool {
			return contentStatus[i].Ref < contentStatus[j].Ref
		})

		tw := tabwriter.NewWriter(pw, 1, 8, 1, ' ', 0)
		content.Display(tw, contentStatus, start)
		tw.Flush()
	}

	return nil
}
