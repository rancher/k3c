package client

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/log"
	pb "github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/rancher/k3c/pkg/status"
	v1 "k8s.io/api/core/v1"
)

var (
	ErrContainerNotFound = errors.New("container not found")
	ErrImageNotFound     = errors.New("image not found")
)

type Client interface {
	ListPods(ctx context.Context) ([]v1.Pod, error)
	CreatePod(ctx context.Context, name string, opts *pb.PodOptions) (string, error)
	CreateContainer(ctx context.Context, podID, image string, opts *pb.ContainerOptions) (string, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string, timeout int64) error
	RemoveContainer(ctx context.Context, containerID string) error
	LogContainer(ctx context.Context, containerID string, opts *v1.PodLogOptions) (<-chan log.Entry, error)

	GetContainer(ctx context.Context, name string) (*v1.Pod, *v1.Container, string, error)
	Attach(ctx context.Context, name string, opts *pb.AttachOptions) (*StreamResponse, error)
	Exec(ctx context.Context, containerName string, cmd []string, opts *pb.ExecOptions) (*StreamResponse, error)

	ListImages(ctx context.Context) (images []Image, err error)
	RemoveImage(ctx context.Context, image string) error
	PullImage(ctx context.Context, image string, authConfig *AuthConfig) (string, error)
	PullProgress(ctx context.Context, image string) (<-chan []status.Info, error)
	PushImage(ctx context.Context, image string, authConfig *AuthConfig) error
	PushProgress(ctx context.Context, image string) (<-chan []status.Info, error)
	GetImage(ctx context.Context, image string) (*Image, error)
	TagImage(ctx context.Context, image string, tags ...string) error

	CreateVolume(ctx context.Context, name string) (*Volume, error)
	ListVolumes(ctx context.Context) ([]Volume, error)
	RemoveVolume(ctx context.Context, name string, force bool) error

	Events(ctx context.Context) (<-chan status.Event, error)

	Close() error
}

type Volume struct {
	ID string
}

type StreamResponse struct {
	URL   string
	TTY   bool
	Stdin bool
}

type Image struct {
	ID string
	// Other names by which this image is known.
	Tags []string
	// Digests by which this image is known.
	Digests []string
	// Size of the image in bytes. Must be > 0.
	Size uint64
}

type AuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth,omitempty"`
	Email         string `json:"email,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"`
	IdentityToken string `json:"identitytoken,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}
