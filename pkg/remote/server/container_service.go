package server

import (
	"context"

	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/rancher/k3c/pkg/status"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type server struct {
	c client.Client
}

func NewContainerService(c client.Client) v1alpha1.ContainerServiceServer {
	return &server{
		c: c,
	}
}

func (s *server) ListPods(ctx context.Context, req *v1alpha1.ListPodsRequest) (*v1alpha1.ListPodsResponse, error) {
	pods, err := s.c.ListPods(ctx)
	resp := &v1alpha1.ListPodsResponse{
		Pods: make([]*v1.Pod, 0, len(pods)),
	}
	for i := range pods {
		resp.Pods = append(resp.Pods, &pods[i])
	}

	return resp, err
}

func (s *server) CreatePod(ctx context.Context, req *v1alpha1.CreatePodRequest) (*v1alpha1.CreatePodResponse, error) {
	id, err := s.c.CreatePod(ctx, req.GetName(), req.GetOpts())
	return &v1alpha1.CreatePodResponse{
		PodID: id,
	}, err
}

func (s *server) CreateContainer(ctx context.Context, req *v1alpha1.CreateContainerRequest) (*v1alpha1.CreateContainerResponse, error) {
	id, err := s.c.CreateContainer(ctx, req.GetPodId(), req.GetImage(), req.GetOpts())
	return &v1alpha1.CreateContainerResponse{
		ContainerId: id,
	}, err
}

func (s *server) StartContainer(ctx context.Context, req *v1alpha1.StartContainerRequest) (*v1alpha1.StartContainerResponse, error) {
	return &v1alpha1.StartContainerResponse{}, s.c.StartContainer(ctx, req.GetContainerId())
}

func (s *server) StopContainer(ctx context.Context, req *v1alpha1.StopContainerRequest) (*v1alpha1.StopContainerResponse, error) {
	return &v1alpha1.StopContainerResponse{}, s.c.StopContainer(ctx, req.GetContainerId(), req.GetTimeout())
}

func (s *server) RemoveContainer(ctx context.Context, req *v1alpha1.RemoveContainerRequest) (*v1alpha1.RemoveContainerResponse, error) {
	return &v1alpha1.RemoveContainerResponse{}, s.c.RemoveContainer(ctx, req.GetContainerId())
}

func (s *server) GetContainer(ctx context.Context, req *v1alpha1.GetContainerRequest) (*v1alpha1.GetContainerResponse, error) {
	pod, container, id, err := s.c.GetContainer(ctx, req.GetName())
	if err == client.ErrContainerNotFound {
		return &v1alpha1.GetContainerResponse{}, nil
	}
	return &v1alpha1.GetContainerResponse{
		Pod:         pod,
		Container:   container,
		ContainerId: id,
	}, err
}

func (s *server) LogContainer(req *v1alpha1.LogContainerRequest, resp v1alpha1.ContainerService_LogContainerServer) error {
	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	result, err := s.c.LogContainer(ctx, req.GetName(), req.GetOpts())
	if err != nil {
		return err
	}

	for msg := range result {
		if err := resp.Send(&v1alpha1.LogEntry{
			Stderr:  msg.Stderr,
			Message: msg.Message,
		}); err != nil {
			cancel()
		}
	}

	return nil
}

func (s *server) Attach(ctx context.Context, req *v1alpha1.AttachRequest) (*v1alpha1.StreamResponse, error) {
	stream, err := s.c.Attach(ctx, req.GetName(), req.GetOpts())
	if err != nil {
		return nil, err
	}
	return &v1alpha1.StreamResponse{
		Url:   stream.URL,
		Tty:   stream.TTY,
		Stdin: stream.Stdin,
	}, err
}

func (s *server) Exec(ctx context.Context, req *v1alpha1.ExecRequest) (*v1alpha1.StreamResponse, error) {
	stream, err := s.c.Exec(ctx, req.GetName(), req.GetCmd(), req.GetOpts())
	if err != nil {
		return nil, err
	}
	return &v1alpha1.StreamResponse{
		Url:   stream.URL,
		Tty:   stream.TTY,
		Stdin: stream.Stdin,
	}, nil
}

func (s *server) ListImages(ctx context.Context, req *v1alpha1.ListImagesRequest) (*v1alpha1.ListImagesResponse, error) {
	images, err := s.c.ListImages(ctx)
	resp := &v1alpha1.ListImagesResponse{
		Images: make([]*v1alpha1.Image, 0, len(images)),
	}
	for _, image := range images {
		resp.Images = append(resp.Images, &v1alpha1.Image{
			Id:      image.ID,
			Tags:    image.Tags,
			Digests: image.Digests,
			Size_:   image.Size,
		})
	}
	return resp, err
}

func (s *server) RemoveImage(ctx context.Context, req *v1alpha1.RemoveImageRequest) (*v1alpha1.RemoveImageResponse, error) {
	return &v1alpha1.RemoveImageResponse{}, s.c.RemoveImage(ctx, req.GetImage())
}

func (s *server) PullImage(ctx context.Context, req *v1alpha1.PullImageRequest) (*v1alpha1.PullImageResponse, error) {
	id, err := s.c.PullImage(ctx, req.GetImage(), toAuthConfig(req.AuthConfig))
	return &v1alpha1.PullImageResponse{
		Image: id,
	}, err
}

func toAuthConfig(authConfig *v1alpha2.AuthConfig) *client.AuthConfig {
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

func (s *server) PushImage(ctx context.Context, req *v1alpha1.PushImageRequest) (*v1alpha1.PushImageResponse, error) {
	return &v1alpha1.PushImageResponse{}, s.c.PushImage(ctx, req.GetImage(), toAuthConfig(req.GetAuthConfig()))
}

func (s *server) GetImage(ctx context.Context, req *v1alpha1.GetImageRequest) (*v1alpha1.GetImageResponse, error) {
	img, err := s.c.GetImage(ctx, req.GetImage())
	if err == client.ErrImageNotFound {
		return &v1alpha1.GetImageResponse{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &v1alpha1.GetImageResponse{
		Image: &v1alpha1.Image{
			Id:      img.ID,
			Tags:    img.Tags,
			Digests: img.Digests,
			Size_:   img.Size,
		},
	}, nil
}

func (s *server) TagImage(ctx context.Context, req *v1alpha1.TagImageRequest) (*v1alpha1.TagImageResponse, error) {
	return &v1alpha1.TagImageResponse{}, s.c.TagImage(ctx, req.GetImage(), req.GetTags()...)
}

func (s *server) PullProgress(req *v1alpha1.InfoRequest, resp v1alpha1.ContainerService_PullProgressServer) error {
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

func toInfoResponse(infos []status.Info) *v1alpha1.InfoResponse {
	resp := &v1alpha1.InfoResponse{}
	for _, info := range infos {
		resp.Info = append(resp.Info, &v1alpha1.Info{
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

func (s *server) PushProgress(req *v1alpha1.InfoRequest, resp v1alpha1.ContainerService_PushProgressServer) error {
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

func (s *server) Events(req *v1alpha1.EventsRequest, resp v1alpha1.ContainerService_EventsServer) error {
	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	events, err := s.c.Events(ctx)
	if err != nil {
		return err
	}

	var lastErr error
	for event := range events {
		if err := resp.Send(&v1alpha1.Event{
			Id:   event.ID,
			Name: event.Name,
		}); err != nil {
			lastErr = err
			cancel()
		}
	}

	return lastErr
}

func (s *server) ListVolumes(ctx context.Context, _ *v1alpha1.ListVolumesRequest) (*v1alpha1.ListVolumesResponse, error) {
	volumes, err := s.c.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}

	resp := &v1alpha1.ListVolumesResponse{}
	for _, volume := range volumes {
		resp.Volumes = append(resp.Volumes, &v1alpha1.Volume{
			Id: volume.ID,
		})
	}

	return resp, nil
}

func (s *server) CreateVolume(ctx context.Context, req *v1alpha1.CreateVolumeRequest) (*v1alpha1.CreateVolumeResponse, error) {
	v, err := s.c.CreateVolume(ctx, req.GetName())
	if err != nil {
		return nil, err
	}
	return &v1alpha1.CreateVolumeResponse{
		Volume: &v1alpha1.Volume{
			Id: v.ID,
		},
	}, nil
}

func (s *server) RemoveVolume(ctx context.Context, req *v1alpha1.RemoveVolumeRequest) (*v1alpha1.RemoveVolumeResponse, error) {
	return &v1alpha1.RemoveVolumeResponse{}, s.c.RemoveVolume(ctx, req.GetName(), req.GetForce())
}
