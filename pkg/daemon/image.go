package daemon

import (
	"context"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/cri"
	"github.com/containerd/cri/pkg/server"
	"github.com/docker/distribution/reference"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/pushstatus"
	"github.com/rancher/k3c/pkg/status"
	"github.com/sirupsen/logrus"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

var (
	closed = make(chan []status.Info)
)

func init() {
	close(closed)
}

func (c *Daemon) RemoveImage(ctx context.Context, id string) error {
	_, err := c.image.RemoveImage(ctx, &pb.RemoveImageRequest{
		Image: &pb.ImageSpec{
			Image: id,
		},
	})
	return err
}

func (c *Daemon) ListImages(ctx context.Context) (images []client.Image, err error) {
	resp, err := c.image.ListImages(ctx, &pb.ListImagesRequest{})
	if err != nil {
		return nil, err
	}

	for _, image := range resp.Images {
		images = append(images, imageToImage(image))
	}

	return
}

func imageToImage(image *pb.Image) client.Image {
	return client.Image{
		ID:      image.Id,
		Tags:    image.RepoTags,
		Digests: image.RepoDigests,
		Size:    image.Size_,
	}
}

func (c *Daemon) resolveImage(ctx context.Context, image string) (images.Image, error) {
	img, err := c.GetImage(ctx, image)
	if err != nil {
		return images.Image{}, err
	}

	if len(img.Digests) == 0 {
		return images.Image{}, errors.Wrapf(client.ErrImageNotFound, "image %s does not have a valid digest", img.ID)
	}

	imageService := c.cClient.ImageService()
	return imageService.Get(ctx, img.Digests[0])
}

func (c *Daemon) TagImage(ctx context.Context, image string, tags ...string) error {
	imageService := c.cClient.ImageService()
	cImage, err := c.resolveImage(ctx, image)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		normalized, err := reference.ParseDockerRef(tag)
		if err != nil {
			return err
		}

		cImage.Name = normalized.String()
		// Attempt to create the image first
		if _, err = imageService.Create(ctx, cImage); err != nil {
			return err
		}
	}

	return nil
}

func (c *Daemon) GetImage(ctx context.Context, image string) (*client.Image, error) {
	resp, err := c.image.ImageStatus(ctx, &pb.ImageStatusRequest{
		Image: &pb.ImageSpec{
			Image: image,
		},
		Verbose: true,
	})
	if err != nil {
		return nil, err
	} else if resp.Image == nil {
		return nil, client.ErrImageNotFound
	}

	result := imageToImage(resp.Image)
	return &result, nil
}

func (c *Daemon) PullProgress(ctx context.Context, image string) (<-chan []status.Info, error) {
	done, err := c.imagePulled(ctx, image)
	if err != nil || done {
		return nil, err
	}

	result := make(chan []status.Info)
	go func() {
		t := time.NewTicker(time.Millisecond * 250)
		defer close(result)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				status, done, err := c.imageProgress(ctx, image)
				result <- status
				if err != nil {
					logrus.Errorf("Failed to get image status for %s: %v", image, err)
					return
				}
				if done {
					return
				}
			}
		}
	}()

	return result, nil
}

func (c *Daemon) imagePulled(ctx context.Context, image string) (bool, error) {
	resp, err := c.image.ImageStatus(ctx, &pb.ImageStatusRequest{
		Image: &pb.ImageSpec{
			Image: image,
		},
	})
	if err != nil {
		return false, err
	}

	return resp.Image != nil, nil
}

func (c *Daemon) imageProgress(ctx context.Context, image string) (result []status.Info, done bool, err error) {
	if done, err := c.imagePulled(ctx, image); err != nil {
		return nil, false, err
	} else if done {
		return nil, true, nil
	}

	statuses, err := c.cClient.ContentStore().ListStatuses(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, s := range statuses {
		message := "waiting"
		if s.Offset == s.Total {
			message = "unpacking"
		} else if s.Offset > 0 {
			message = "downloading"
		}
		result = append(result, status.Info{
			Ref:       s.Ref,
			Status:    message,
			Offset:    s.Offset,
			Total:     s.Total,
			StartedAt: s.StartedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}

	return result, false, nil
}

func toAuth(authConfig *client.AuthConfig) *pb.AuthConfig {
	if authConfig == nil {
		return nil
	}

	return &pb.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}
}

func (c *Daemon) PullImage(ctx context.Context, image string, authConfig *client.AuthConfig) (string, error) {
	resp, err := c.image.PullImage(ctx, &pb.PullImageRequest{
		Image: &pb.ImageSpec{
			Image: image,
		},
		Auth:          toAuth(authConfig),
		SandboxConfig: nil,
	})
	if err != nil {
		return "", err
	}
	return resp.ImageRef, nil
}

func (c *Daemon) PushProgress(ctx context.Context, image string) (<-chan []status.Info, error) {
	timeout := time.After(time.Minute)

	for {
		c.lock.Lock()
		t := c.pushJobs[image]
		c.lock.Unlock()

		if t != nil {
			return t.Status(), nil
		}

		select {
		case <-timeout:
			return closed, nil
		case <-ctx.Done():
			return closed, nil
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (c *Daemon) PushImage(ctx context.Context, image string, authConfig *client.AuthConfig) error {
	resolver := cri.Resolver.GetResolver(toAuth(authConfig))
	tracker := pushstatus.NewTracker(ctx, server.Tracker)

	c.lock.Lock()
	c.pushJobs[image] = tracker
	c.lock.Unlock()

	defer func() {
		c.lock.Lock()
		delete(c.pushJobs, image)
		c.lock.Unlock()
	}()

	cImage, err := c.resolveImage(ctx, image)
	if err != nil {
		return err
	}

	jobHandler := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		tracker.Add(remotes.MakeRefKey(ctx, desc))
		return nil, nil
	})

	return c.cClient.Push(ctx, cImage.Name, cImage.Target,
		containerd.WithResolver(resolver),
		containerd.WithImageHandler(jobHandler),
	)
}
