package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cmd/ctr/commands"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/k3c/pkg/auth"
	"github.com/rancher/k3c/pkg/progress"
	"github.com/rancher/k3c/pkg/version"
	"github.com/sirupsen/logrus"
)

// Push server-side impl
func (i *Interface) Push(ctx context.Context, request *imagesv1.ImagePushRequest) (*imagesv1.ImagePushResponse, error) {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	img, err := i.Containerd.ImageService().Get(ctx, request.Image.Image)
	if err != nil {
		return nil, err
	}

	authorizer := docker.NewDockerAuthorizer(
		docker.WithAuthClient(http.DefaultClient),
		docker.WithAuthCreds(func(host string) (string, string, error) {
			return auth.Parse(request.Auth, host)
		}),
		docker.WithAuthHeader(http.Header{
			"User-Agent": []string{fmt.Sprintf("k3c/%s", version.Version)},
		}),
	)
	resolver := docker.NewResolver(docker.ResolverOptions{
		Tracker: commands.PushTracker,
		Hosts: docker.ConfigureDefaultRegistries(
			docker.WithAuthorizer(authorizer),
		),
	})
	tracker := progress.NewTracker(ctx, commands.PushTracker)
	i.pushes.Store(img.Name, tracker)
	handler := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		tracker.Add(remotes.MakeRefKey(ctx, desc))
		return nil, nil
	})
	err = i.Containerd.Push(ctx, img.Name, img.Target,
		containerd.WithResolver(resolver),
		containerd.WithImageHandler(handler),
	)
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImagePushResponse{
		Image: img.Name,
	}, nil
}

// PushProgress server-side impl
func (i *Interface) PushProgress(req *imagesv1.ImageProgressRequest, srv imagesv1.Images_PushProgressServer) error {
	ctx := namespaces.WithNamespace(srv.Context(), "k8s.io")
	defer i.pushes.Delete(req.Image)

	timeout := time.After(15 * time.Second)

	for {
		if tracker, tracking := i.pushes.Load(req.Image); tracking {
			for status := range tracker.(progress.Tracker).Status() {
				if err := srv.Send(&imagesv1.ImageProgressResponse{Status: status}); err != nil {
					logrus.Debugf("push-progress-error: %s -> %v", req.Image, err)
					return err
				}
			}
			logrus.Debugf("push-progress-done: %s", req.Image)
			return nil
		}
		select {
		case <-timeout:
			logrus.Debugf("push-progress-timeout: not tracking %s", req.Image)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
