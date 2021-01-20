package progress

import (
	"io"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/containerd/containerd/cmd/ctr/commands/content"
	"github.com/containerd/containerd/pkg/progress"
	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
)

func Display(sch <-chan []imagesv1.ImageStatus, out io.Writer) (err error) {
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

	for sc := range sch {
		pw.Flush()
		if len(sc) == 0 {
			continue
		}

		var status []content.StatusInfo
		for _, s := range sc {
			status = append(status, content.StatusInfo{
				Ref:       s.Ref,
				Status:    s.Status,
				Offset:    s.Offset,
				Total:     s.Total,
				StartedAt: s.StartedAt,
				UpdatedAt: s.UpdatedAt,
			})
		}

		sort.Slice(status, func(i, j int) bool {
			return status[i].Ref < status[j].Ref
		})

		tw := tabwriter.NewWriter(pw, 1, 8, 1, ' ', 0)
		content.Display(tw, status, start)
		tw.Flush()
	}

	return nil
}
