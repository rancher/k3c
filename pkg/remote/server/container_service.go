package server

import (
	"context"
	"io"

	"github.com/containerd/containerd/plugin"
	"github.com/containerd/cri/pkg/atomic"
	"github.com/rancher/k3c/pkg/client"
	k3cv1 "github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/rancher/k3c/pkg/status"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type ContainerService interface {
	io.Closer
	plugin.Service
	k3cv1.ContainerServiceServer
	SetInitialized(bool)
}

type server struct {
	c client.Client
	// initialized indicates whether the server is initialized. All GRPC services
	// should return error before the server is initialized.
	initialized atomic.Bool
}

func NewContainerService(c client.Client) ContainerService {
	return &server{
		c:           c,
		initialized: atomic.NewBool(false),
	}
}

func (s *server) Close() error {
	logrus.Info("Stop K3C service")
	return s.c.Close()
}

func (s *server) Register(g *grpc.Server) error {
	logrus.Debugf("Register K3C ContainerService")
	k3cv1.RegisterContainerServiceServer(g, &wrapper{svc: s})
	return nil
}

func (s *server) SetInitialized(initialized bool) {
	if initialized {
		s.initialized.Set()
	} else {
		s.initialized.Unset()
	}
}

func (s *server) ListPods(ctx context.Context, req *k3cv1.ListPodsRequest) (*k3cv1.ListPodsResponse, error) {
	pods, err := s.c.ListPods(ctx)
	resp := &k3cv1.ListPodsResponse{
		Pods: make([]*corev1.Pod, 0, len(pods)),
	}
	for i := range pods {
		resp.Pods = append(resp.Pods, &pods[i])
	}

	return resp, err
}

func (s *server) CreatePod(ctx context.Context, req *k3cv1.CreatePodRequest) (*k3cv1.CreatePodResponse, error) {
	id, err := s.c.CreatePod(ctx, req.GetName(), req.GetOpts())
	return &k3cv1.CreatePodResponse{
		PodID: id,
	}, err
}

func (s *server) CreateContainer(ctx context.Context, req *k3cv1.CreateContainerRequest) (*k3cv1.CreateContainerResponse, error) {
	id, err := s.c.CreateContainer(ctx, req.GetPodId(), req.GetImage(), req.GetOpts())
	return &k3cv1.CreateContainerResponse{
		ContainerId: id,
	}, err
}

func (s *server) StartContainer(ctx context.Context, req *k3cv1.StartContainerRequest) (*k3cv1.StartContainerResponse, error) {
	return &k3cv1.StartContainerResponse{}, s.c.StartContainer(ctx, req.GetContainerId())
}

func (s *server) StopContainer(ctx context.Context, req *k3cv1.StopContainerRequest) (*k3cv1.StopContainerResponse, error) {
	return &k3cv1.StopContainerResponse{}, s.c.StopContainer(ctx, req.GetContainerId(), req.GetTimeout())
}

func (s *server) RemoveContainer(ctx context.Context, req *k3cv1.RemoveContainerRequest) (*k3cv1.RemoveContainerResponse, error) {
	return &k3cv1.RemoveContainerResponse{}, s.c.RemoveContainer(ctx, req.GetContainerId())
}

func (s *server) GetContainer(ctx context.Context, req *k3cv1.GetContainerRequest) (*k3cv1.GetContainerResponse, error) {
	pod, container, id, err := s.c.GetContainer(ctx, req.GetName())
	if err == client.ErrContainerNotFound {
		return &k3cv1.GetContainerResponse{}, nil
	}
	return &k3cv1.GetContainerResponse{
		Pod:         pod,
		Container:   container,
		ContainerId: id,
	}, err
}

func (s *server) LogContainer(req *k3cv1.LogContainerRequest, resp k3cv1.ContainerService_LogContainerServer) error {
	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	result, err := s.c.LogContainer(ctx, req.GetName(), req.GetOpts())
	if err != nil {
		return err
	}

	for msg := range result {
		if err := resp.Send(&k3cv1.LogEntry{
			Stderr:  msg.Stderr,
			Message: msg.Message,
		}); err != nil {
			cancel()
		}
	}

	return nil
}

func (s *server) Attach(ctx context.Context, req *k3cv1.AttachRequest) (*k3cv1.StreamResponse, error) {
	stream, err := s.c.Attach(ctx, req.GetName(), req.GetOpts())
	if err != nil {
		return nil, err
	}
	return &k3cv1.StreamResponse{
		Url:   stream.URL,
		Tty:   stream.TTY,
		Stdin: stream.Stdin,
	}, err
}

func (s *server) Exec(ctx context.Context, req *k3cv1.ExecRequest) (*k3cv1.StreamResponse, error) {
	stream, err := s.c.Exec(ctx, req.GetName(), req.GetCmd(), req.GetOpts())
	if err != nil {
		return nil, err
	}
	return &k3cv1.StreamResponse{
		Url:   stream.URL,
		Tty:   stream.TTY,
		Stdin: stream.Stdin,
	}, nil
}

func (s *server) ListImages(ctx context.Context, req *k3cv1.ListImagesRequest) (*k3cv1.ListImagesResponse, error) {
	images, err := s.c.ListImages(ctx)
	resp := &k3cv1.ListImagesResponse{
		Images: make([]*k3cv1.Image, 0, len(images)),
	}
	for _, image := range images {
		resp.Images = append(resp.Images, &k3cv1.Image{
			Id:      image.ID,
			Tags:    image.Tags,
			Digests: image.Digests,
			Size_:   image.Size,
		})
	}
	return resp, err
}

func (s *server) RemoveImage(ctx context.Context, req *k3cv1.RemoveImageRequest) (*k3cv1.RemoveImageResponse, error) {
	return &k3cv1.RemoveImageResponse{}, s.c.RemoveImage(ctx, req.GetImage())
}

func (s *server) PullImage(ctx context.Context, req *k3cv1.PullImageRequest) (*k3cv1.PullImageResponse, error) {
	id, err := s.c.PullImage(ctx, req.GetImage(), toAuthConfig(req.AuthConfig))
	return &k3cv1.PullImageResponse{
		Image: id,
	}, err
}

func toAuthConfig(authConfig *criv1.AuthConfig) *client.AuthConfig {
	if authConfig == nil {
		return nil
	}
	return &client.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}
}

func (s *server) PushImage(ctx context.Context, req *k3cv1.PushImageRequest) (*k3cv1.PushImageResponse, error) {
	return &k3cv1.PushImageResponse{}, s.c.PushImage(ctx, req.GetImage(), toAuthConfig(req.GetAuthConfig()))
}

func (s *server) GetImage(ctx context.Context, req *k3cv1.GetImageRequest) (*k3cv1.GetImageResponse, error) {
	img, err := s.c.GetImage(ctx, req.GetImage())
	if err == client.ErrImageNotFound {
		return &k3cv1.GetImageResponse{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &k3cv1.GetImageResponse{
		Image: &k3cv1.Image{
			Id:      img.ID,
			Tags:    img.Tags,
			Digests: img.Digests,
			Size_:   img.Size,
		},
	}, nil
}

func (s *server) TagImage(ctx context.Context, req *k3cv1.TagImageRequest) (*k3cv1.TagImageResponse, error) {
	return &k3cv1.TagImageResponse{}, s.c.TagImage(ctx, req.GetImage(), req.GetTags()...)
}

func (s *server) PullProgress(req *k3cv1.InfoRequest, resp k3cv1.ContainerService_PullProgressServer) error {
	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	c, err := s.c.PullProgress(ctx, req.GetImage())
	if err != nil {
		return err
	}
	var lastErr error
	for info := range c {
		if err := resp.Send(toInfoResponse(info)); err != nil {
			cancel()
			lastErr = err
		}
	}
	return lastErr
}

func toInfoResponse(infos []status.Info) *k3cv1.InfoResponse {
	resp := &k3cv1.InfoResponse{}
	for _, info := range infos {
		resp.Info = append(resp.Info, &k3cv1.Info{
			Ref:       info.Ref,
			Status:    info.Status,
			Offset:    info.Offset,
			Total:     info.Total,
			StartedAt: info.StartedAt.UnixNano(),
			UpdatedAt: info.UpdatedAt.UnixNano(),
		})
	}
	return resp
}

func (s *server) PushProgress(req *k3cv1.InfoRequest, resp k3cv1.ContainerService_PushProgressServer) error {
	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	c, err := s.c.PushProgress(ctx, req.GetImage())
	if err != nil {
		return err
	}

	var lastErr error
	for info := range c {
		if err := resp.Send(toInfoResponse(info)); err != nil {
			lastErr = err
			cancel()
		}
	}

	return lastErr
}

func (s *server) Events(req *k3cv1.EventsRequest, resp k3cv1.ContainerService_EventsServer) error {
	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	events, err := s.c.Events(ctx)
	if err != nil {
		return err
	}

	var lastErr error
	for event := range events {
		if err := resp.Send(&k3cv1.Event{
			Id:   event.ID,
			Name: event.Name,
		}); err != nil {
			lastErr = err
			cancel()
		}
	}

	return lastErr
}

func (s *server) ListVolumes(ctx context.Context, _ *k3cv1.ListVolumesRequest) (*k3cv1.ListVolumesResponse, error) {
	volumes, err := s.c.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}

	resp := &k3cv1.ListVolumesResponse{}
	for _, volume := range volumes {
		resp.Volumes = append(resp.Volumes, &k3cv1.Volume{
			Id: volume.ID,
		})
	}

	return resp, nil
}

func (s *server) CreateVolume(ctx context.Context, req *k3cv1.CreateVolumeRequest) (*k3cv1.CreateVolumeResponse, error) {
	v, err := s.c.CreateVolume(ctx, req.GetName())
	if err != nil {
		return nil, err
	}
	return &k3cv1.CreateVolumeResponse{
		Volume: &k3cv1.Volume{
			Id: v.ID,
		},
	}, nil
}

func (s *server) RemoveVolume(ctx context.Context, req *k3cv1.RemoveVolumeRequest) (*k3cv1.RemoveVolumeResponse, error) {
	return &k3cv1.RemoveVolumeResponse{}, s.c.RemoveVolume(ctx, req.GetName(), req.GetForce())
}
