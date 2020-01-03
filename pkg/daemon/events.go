package daemon

import (
	"context"
	"fmt"

	eventtypes "github.com/containerd/containerd/api/events"
	"github.com/containerd/typeurl"
	"github.com/rancher/k3c/pkg/status"
	"github.com/sirupsen/logrus"
)

func (d *Daemon) Events(ctx context.Context) (<-chan status.Event, error) {
	events, errs := d.cClient.EventService().Subscribe(ctx)
	result := make(chan status.Event)

	// consume and drop errs
	go func() {
		for range errs {
		}
	}()

	go func() {
		defer close(result)
		for event := range events {
			evt, err := typeurl.UnmarshalAny(event.Event)
			if err != nil {
				logrus.Errorf("unmarshal event: %v", err)
				return
			}

			switch e := evt.(type) {
			case *eventtypes.ContainerCreate:
				result <- status.Event{
					ID:   e.ID,
					Name: "container.create",
				}
			case *eventtypes.ContainerDelete:
				result <- status.Event{
					ID:   e.ID,
					Name: "container.delete",
				}
			case *eventtypes.TaskStart:
				result <- status.Event{
					ID:   fmt.Sprintf("%s/%d", e.ContainerID, e.Pid),
					Name: "task.start",
				}
			case *eventtypes.TaskExit:
				result <- status.Event{
					ID:   fmt.Sprintf("%s/%d", e.ContainerID, e.Pid),
					Name: "task.exit",
				}
			}
		}
	}()

	return result, nil
}
