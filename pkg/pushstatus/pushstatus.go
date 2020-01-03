package pushstatus

import (
	"context"
	"sync"
	"time"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/rancher/k3c/pkg/status"
)

// most of this is copied from containerd ctr tool

type Tracker interface {
	Add(ref string)
	Status() <-chan []status.Info
}

type tracker struct {
	*pushjobs
	status chan []status.Info
}

func (t *tracker) Status() <-chan []status.Info {
	return t.status
}

func NewTracker(ctx context.Context, statusTracker docker.StatusTracker) Tracker {
	ongoing := newPushJobs(statusTracker)

	var (
		result = make(chan []status.Info)
	)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)

		defer ticker.Stop()
		defer close(result)

		for {
			select {
			case <-ticker.C:
				result <- ongoing.status()
			case <-ctx.Done():
			}
		}
	}()

	return &tracker{
		pushjobs: ongoing,
		status:   result,
	}
}

type pushjobs struct {
	jobs    map[string]struct{}
	ordered []string
	tracker docker.StatusTracker
	mu      sync.Mutex
}

func newPushJobs(tracker docker.StatusTracker) *pushjobs {
	return &pushjobs{
		jobs:    make(map[string]struct{}),
		tracker: tracker,
	}
}

func (j *pushjobs) Add(ref string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if _, ok := j.jobs[ref]; ok {
		return
	}
	j.ordered = append(j.ordered, ref)
	j.jobs[ref] = struct{}{}
}

func (j *pushjobs) status() []status.Info {
	j.mu.Lock()
	defer j.mu.Unlock()

	statuses := make([]status.Info, 0, len(j.jobs))
	for _, name := range j.ordered {
		si := status.Info{
			Ref: name,
		}

		status, err := j.tracker.GetStatus(name)
		if err != nil {
			si.Status = "waiting"
		} else {
			si.Offset = status.Offset
			si.Total = status.Total
			si.StartedAt = status.StartedAt
			si.UpdatedAt = status.UpdatedAt
			if status.Offset >= status.Total {
				if status.UploadUUID == "" {
					si.Status = "done"
				} else {
					si.Status = "committing"
				}
			} else {
				si.Status = "uploading"
			}
		}
		statuses = append(statuses, si)
	}

	return statuses
}
