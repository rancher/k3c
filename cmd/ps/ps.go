package ps

import (
	"encoding/json"
	"sort"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/tables"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	v1 "k8s.io/api/core/v1"
)

type Ps struct {
	A_All   bool   `usage:"Show all images (default hides intermediate images)"`
	Format  string `usage:"Pretty-print images using a Go template"`
	NoTrunc bool   `usage:"Don't truncate output"`
	Q_Quiet bool   `usage:"Only show numeric IDs"`
}

func (p *Ps) Run(ctx *cli.Context) error {
	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	pods, err := client.ListPods(ctx.Context)
	if err != nil {
		return err
	}

	table := tables.NewPods(ctx)

	sort.Slice(pods, func(i, j int) bool {
		return pods[i].CreationTimestamp.After(pods[j].CreationTimestamp.Time)
	})

	for _, pod := range pods {
		statuses := map[string]v1.ContainerStatus{}
		for _, status := range pod.Status.ContainerStatuses {
			statuses[status.Name] = status
		}
		for _, container := range pod.Spec.Containers {
			infoStrings := map[string]string{}
			info := map[string]interface{}{}
			key := "info.k3c.io/" + container.Name
			if err := json.Unmarshal([]byte(pod.Annotations[key]), &infoStrings); err != nil {
				logrus.Errorf("failed to parse container info for %s/%s", pod.Name, container.Name)
			}
			delete(pod.Annotations, key)

			for k, v := range infoStrings {
				info[k] = v
				if k == "info" {
					data := map[string]interface{}{}
					if err := json.Unmarshal([]byte(v), &data); err == nil {
						delete(info, k)
						for k, v := range data {
							info[k] = v
						}
					}
				}
			}

			data := tables.ContainerData{
				Pod:       pod,
				Container: container,
				Status:    statuses[container.Name],
				Info:      info,
			}

			if !p.A_All && data.Status.State.Running == nil {
				continue
			}
			table.Write(data)
		}
	}

	return table.Close()
}
