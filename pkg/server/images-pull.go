package server

import (
	"context"
	"time"

	"github.com/containerd/containerd/namespaces"
	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
	"github.com/sirupsen/logrus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// Pull server-side impl
func (i *Interface) Pull(ctx context.Context, request *imagesv1.ImagePullRequest) (*imagesv1.ImagePullResponse, error) {
	req := &criv1.PullImageRequest{
		Image: request.Image,
	}
	res, err := i.ImageService.PullImage(ctx, req)
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImagePullResponse{
		Image: res.ImageRef,
	}, nil
}

// PullProgress server-side impl
func (i *Interface) PullProgress(req *imagesv1.ImageProgressRequest, srv imagesv1.Images_PullProgressServer) error {
	ctx := namespaces.WithNamespace(srv.Context(), "k8s.io")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			isr, err := i.ImageService.ImageStatus(ctx, &criv1.ImageStatusRequest{
				Image: &criv1.ImageSpec{
					Image: req.Image,
				},
			})
			if err != nil {
				logrus.Debugf("pull-progress-image-status-error: %v", err)
				return err
			}
			if isr.Image != nil {
				logrus.Debugf("pull-progress-image-status-done: %s", isr.Image)
				return nil
			}
			csl, err := i.Containerd.ContentStore().ListStatuses(ctx, "") // TODO is this filter too broad?
			if err != nil {
				logrus.Debugf("pull-progress-content-status-error: %v", err)
				return err
			}
			res := &imagesv1.ImageProgressResponse{}
			for _, s := range csl {
				status := "waiting"
				if s.Offset == s.Total {
					status = "unpacking"
				} else if s.Offset > 0 {
					status = "downloading"
				}
				res.Status = append(res.Status, imagesv1.ImageStatus{
					Status:    status,
					Ref:       s.Ref,
					Offset:    s.Offset,
					Total:     s.Total,
					StartedAt: s.StartedAt,
					UpdatedAt: s.UpdatedAt,
				})
			}
			if err = srv.Send(res); err != nil {
				logrus.Debugf("pull-progress-content-send-error: %v", err)
				return err
			}
		}
	}
}
