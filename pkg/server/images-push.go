package server

import (
	"context"

	"github.com/containerd/containerd/namespaces"
	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
)

// Push server-side impl
func (i *Interface) Push(ctx context.Context, request *imagesv1.ImagePushRequest) (*imagesv1.ImagePushResponse, error) {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	panic("implement me")
}

// PushProgress server-side impl
func (i *Interface) PushProgress(request *imagesv1.ImageProgressRequest, server imagesv1.Images_PushProgressServer) error {
	panic("implement me")
}
