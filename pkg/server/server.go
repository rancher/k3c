package server

import (
	"github.com/containerd/containerd"
	buildkit "github.com/moby/buildkit/client"
	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/k3c/pkg/client"
	"github.com/sirupsen/logrus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

var _ imagesv1.ImagesServer = &Interface{}

type Interface struct {
	Kubernetes     *client.Interface
	Buildkit       *buildkit.Client
	Containerd     *containerd.Client
	RuntimeService criv1.RuntimeServiceClient
	ImageService   criv1.ImageServiceClient
}

// Close the Interface connections to various backends.
func (i *Interface) Close() {
	if i.Buildkit != nil {
		if err := i.Buildkit.Close(); err != nil {
			logrus.Warnf("error closing connection to buildkit: %v", err)
		}
	}
	if i.Containerd != nil {
		// this will close the underlying grpc connection making the cri runtime/images clients inoperable as well
		if err := i.Containerd.Close(); err != nil {
			logrus.Warnf("error closing connection to containerd: %v", err)
		}
	}
}
