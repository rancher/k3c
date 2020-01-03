package client

import (
	"context"
	"io"
	"time"

	"github.com/rancher/k3c/pkg/endpointconn"
	"github.com/rancher/k3c/pkg/log"
	pb "github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/rancher/k3c/pkg/status"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type client struct {
	conn *grpc.ClientConn
	s    pb.ContainerServiceClient
}

func New(ctx context.Context, endpoint string) (Client, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	conn, err := endpointconn.Get(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &client{
		conn: conn,
		s:    pb.NewContainerServiceClient(conn),
	}, nil
}

func (c *client) ListPods(ctx context.Context) ([]v1.Pod, error) {
	resp, err := c.s.ListPods(ctx, &pb.ListPodsRequest{})
	var pods []v1.Pod
	for _, pod := range resp.GetPods() {
		pods = append(pods, *pod)
	}

	return pods, err
}

func (c *client) CreatePod(ctx context.Context, name string, opts *pb.PodOptions) (string, error) {
	resp, err := c.s.CreatePod(ctx, &pb.CreatePodRequest{
		Name: name,
		Opts: opts,
	})
	return resp.GetPodID(), err
}

func (c *client) CreateContainer(ctx context.Context, podID, image string, opts *pb.ContainerOptions) (string, error) {
	resp, err := c.s.CreateContainer(ctx, &pb.CreateContainerRequest{
		PodId: podID,
		Image: image,
		Opts:  opts,
	})
	return resp.GetContainerId(), err
}

func (c *client) RemoveContainer(ctx context.Context, containerID string) error {
	_, err := c.s.RemoveContainer(ctx, &pb.RemoveContainerRequest{
		ContainerId: containerID,
	})
	return err
}

func (c *client) StartContainer(ctx context.Context, containerID string) error {
	_, err := c.s.StartContainer(ctx, &pb.StartContainerRequest{
		ContainerId: containerID,
	})
	return err
}

func (c *client) StopContainer(ctx context.Context, containerID string, timeout int64) error {
	_, err := c.s.StopContainer(ctx, &pb.StopContainerRequest{
		ContainerId: containerID,
		Timeout:     timeout,
	})
	return err
}

func (c *client) GetContainer(ctx context.Context, name string) (*v1.Pod, *v1.Container, string, error) {
	resp, err := c.s.GetContainer(ctx, &pb.GetContainerRequest{
		Name: name,
	})
	if err == nil && resp.GetContainerId() == "" {
		return nil, nil, "", ErrContainerNotFound
	}
	return resp.GetPod(), resp.GetContainer(), resp.GetContainerId(), err
}

func (c *client) LogContainer(ctx context.Context, containerID string, opts *v1.PodLogOptions) (<-chan log.Entry, error) {
	resp, err := c.s.LogContainer(ctx, &pb.LogContainerRequest{
		Name: containerID,
		Opts: opts,
	})
	if err != nil {
		return nil, err
	}

	result := make(chan log.Entry)
	go func() {
		defer close(result)
		for {
			msg, err := resp.Recv()
			if err != nil {
				return
			}
			result <- log.Entry{
				Stderr:  msg.Stderr,
				Message: msg.Message,
			}
		}
	}()

	return result, nil
}

func (c *client) Attach(ctx context.Context, name string, opts *pb.AttachOptions) (*StreamResponse, error) {
	resp, err := c.s.Attach(ctx, &pb.AttachRequest{
		Name: name,
		Opts: opts,
	})
	return &StreamResponse{
		URL:   resp.GetUrl(),
		TTY:   resp.GetTty(),
		Stdin: resp.GetStdin(),
	}, err
}

func (c *client) Exec(ctx context.Context, name string, cmd []string, opts *pb.ExecOptions) (*StreamResponse, error) {
	resp, err := c.s.Exec(ctx, &pb.ExecRequest{
		Name: name,
		Cmd:  cmd,
		Opts: opts,
	})
	return &StreamResponse{
		URL:   resp.GetUrl(),
		TTY:   resp.GetTty(),
		Stdin: resp.GetStdin(),
	}, err
}

func (c *client) ListImages(ctx context.Context) (images []Image, err error) {
	resp, err := c.s.ListImages(ctx, &pb.ListImagesRequest{})
	for _, image := range resp.GetImages() {
		images = append(images, Image{
			ID:      image.Id,
			Tags:    image.Tags,
			Digests: image.Digests,
			Size:    image.Size_,
		})
	}
	return images, err
}

func (c *client) RemoveImage(ctx context.Context, image string) error {
	_, err := c.s.RemoveImage(ctx, &pb.RemoveImageRequest{
		Image: image,
	})
	return err
}

func toAuthConfig(authConfig *AuthConfig) *v1alpha2.AuthConfig {
	if authConfig == nil {
		return nil
	}
	return &v1alpha2.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}
}
func (c *client) PullImage(ctx context.Context, image string, authConfig *AuthConfig) (string, error) {
	resp, err := c.s.PullImage(ctx, &pb.PullImageRequest{
		Image:      image,
		AuthConfig: toAuthConfig(authConfig),
	})
	return resp.GetImage(), err
}

func (c *client) PushImage(ctx context.Context, image string, authConfig *AuthConfig) error {
	_, err := c.s.PushImage(ctx, &pb.PushImageRequest{
		Image:      image,
		AuthConfig: toAuthConfig(authConfig),
	})
	return err
}

func (c *client) PullProgress(ctx context.Context, image string) (<-chan []status.Info, error) {
	resp, err := c.s.PullProgress(ctx, &pb.InfoRequest{
		Image: image,
	})
	if err != nil {
		return nil, err
	}
	return stream(resp.Recv), nil
}

func (c *client) PushProgress(ctx context.Context, image string) (<-chan []status.Info, error) {
	resp, err := c.s.PushProgress(ctx, &pb.InfoRequest{
		Image: image,
	})
	if err != nil {
		return nil, err
	}
	return stream(resp.Recv), nil
}

type recv func() (*pb.InfoResponse, error)

func stream(fn recv) <-chan []status.Info {
	c := make(chan []status.Info)
	go func() {
		defer close(c)
		for {
			info, err := fn()
			if err != nil {
				logrus.Debugf("failed to stream pull progress info: %v", err)
				return
			}
			var infos []status.Info
			for _, info := range info.Info {
				infos = append(infos, status.Info{
					Ref:       info.Ref,
					Status:    info.Status,
					Offset:    info.Offset,
					Total:     info.Total,
					StartedAt: time.Unix(0, info.StartedAt),
					UpdatedAt: time.Unix(0, info.UpdatedAt),
				})
			}
			c <- infos
		}
	}()

	return c
}

func (c *client) GetImage(ctx context.Context, image string) (*Image, error) {
	resp, err := c.s.GetImage(ctx, &pb.GetImageRequest{
		Image: image,
	})
	if err == nil && resp.GetImage().GetId() == "" {
		return nil, ErrImageNotFound
	}
	return &Image{
		ID:      resp.GetImage().GetId(),
		Tags:    resp.GetImage().GetTags(),
		Digests: resp.GetImage().GetDigests(),
		Size:    resp.GetImage().GetSize_(),
	}, err
}

func (c *client) TagImage(ctx context.Context, image string, tags ...string) error {
	_, err := c.s.TagImage(ctx, &pb.TagImageRequest{
		Image: image,
		Tags:  tags,
	})
	return err
}

func (c *client) Close() error {
	return c.conn.Close()
}

func (c *client) Events(ctx context.Context) (<-chan status.Event, error) {
	events, err := c.s.Events(ctx, &pb.EventsRequest{})
	if err != nil {
		return nil, err
	}

	result := make(chan status.Event)
	go func() {
		defer close(result)

		for {
			event, err := events.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				logrus.Infof("error streaming events: %v", err)
				return
			}

			result <- status.Event{
				ID:   event.Id,
				Name: event.Name,
			}
		}
	}()

	return result, nil
}

func (c *client) CreateVolume(ctx context.Context, name string) (*Volume, error) {
	v, err := c.s.CreateVolume(ctx, &pb.CreateVolumeRequest{
		Name: name,
	})
	return &Volume{
		ID: v.GetVolume().GetId(),
	}, err
}

func (c *client) ListVolumes(ctx context.Context) (result []Volume, err error) {
	volumes, err := c.s.ListVolumes(ctx, &pb.ListVolumesRequest{})
	for _, volume := range volumes.GetVolumes() {
		result = append(result, Volume{
			ID: volume.GetId(),
		})
	}

	return result, err
}

func (c *client) RemoveVolume(ctx context.Context, name string, force bool) error {
	_, err := c.s.RemoveVolume(ctx, &pb.RemoveVolumeRequest{
		Name:  name,
		Force: force,
	})
	return err
}
