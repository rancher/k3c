package action

import (
	"context"
	"fmt"
	"net"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/typeurl"
	"github.com/gogo/protobuf/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/server"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func (s *Agent) Run(ctx context.Context) error {
	backend, err := s.Interface(ctx, &client.DefaultConfig)
	if err != nil {
		return err
	}
	defer backend.Close()

	go s.syncImageContent(namespaces.WithNamespace(ctx, s.BuildkitNamespace), backend.Containerd)
	go s.listenAndServe(ctx, backend)

	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Agent) listenAndServe(ctx context.Context, backend *server.Interface) error {
	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%d", s.AgentPort))
	if err != nil {
		return err
	}
	defer listener.Close()

	server := grpc.NewServer()
	imagesv1.RegisterImagesServer(server, backend)
	defer server.Stop()
	return server.Serve(listener)
}

func (s *Agent) syncImageContent(ctx context.Context, ctr *containerd.Client) {
	events, errors := ctr.EventService().Subscribe(ctx, `topic~="/images/"`)
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-errors:
			if !ok {
				return
			}
			logrus.Errorf("sync-image-content: %v", err)
		case evt, ok := <-events:
			if !ok {
				return
			}
			if evt.Namespace != s.BuildkitNamespace {
				continue
			}
			if err := handleImageEvent(ctx, ctr, evt.Event); err != nil {
				logrus.Errorf("sync-image-content: handling %#v returned %v", evt, err)
			}
		}
	}
}

func handleImageEvent(ctx context.Context, ctr *containerd.Client, any *types.Any) error {
	evt, err := typeurl.UnmarshalAny(any)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal any")
	}

	switch e := evt.(type) {
	case *events.ImageCreate:
		logrus.Debugf("image-create: %s", e.Name)
		return copyImageContent(ctx, ctr, e.Name, func(x context.Context, s images.Store, i images.Image) error {
			_, err := s.Create(x, i)
			if errdefs.IsAlreadyExists(err) {
				_, err = s.Update(x, i)
			}
			return err
		})
	case *events.ImageUpdate:
		logrus.Debugf("image-update: %s", e.Name)
		return copyImageContent(ctx, ctr, e.Name, func(x context.Context, s images.Store, i images.Image) error {
			_, err := s.Create(x, i)
			if errdefs.IsAlreadyExists(err) {
				_, err = s.Update(x, i)
			}
			return err
		})
	}

	return nil
}

func copyImageContent(ctx context.Context, ctr *containerd.Client, name string, fn func(context.Context, images.Store, images.Image) error) error {
	imageStore := ctr.ImageService()
	img, err := imageStore.Get(ctx, name)
	if err != nil {
		return err
	}
	contentStore := ctr.ContentStore()
	toCtx := namespaces.WithNamespace(ctx, "k8s.io")
	handler := images.Handlers(images.ChildrenHandler(contentStore), copyImageContentFunc(toCtx, contentStore, img))
	if err = images.Walk(ctx, handler, img.Target); err != nil {
		return err
	}
	return fn(toCtx, imageStore, img)
}

func copyImageContentFunc(toCtx context.Context, contentStore content.Store, img images.Image) images.HandlerFunc {
	return func(fromCtx context.Context, desc ocispec.Descriptor) (children []ocispec.Descriptor, err error) {
		logrus.Debugf("copy-image-content: media-type=%v, digest=%v", desc.MediaType, desc.Digest)
		info, err := contentStore.Info(fromCtx, desc.Digest)
		if err != nil {
			return children, err
		}
		ra, err := contentStore.ReaderAt(fromCtx, desc)
		if err != nil {
			return children, err
		}
		defer ra.Close()
		wopts := []content.WriterOpt{content.WithRef(img.Name)}
		if _, err := contentStore.Info(toCtx, desc.Digest); errdefs.IsNotFound(err) {
			// if the image does not already exist in the target namespace we supply the descriptor here so as to
			// ensure that it is created with proper size information. if the image already exist the size for the digest
			// for the to-be updated image is sourced from what is passed to content.Copy
			wopts = append(wopts, content.WithDescriptor(desc))
		}
		w, err := contentStore.Writer(toCtx, wopts...)
		if err != nil {
			return children, err
		}
		defer w.Close()
		err = content.Copy(toCtx, w, content.NewReader(ra), desc.Size, desc.Digest, content.WithLabels(info.Labels))
		if err != nil && errdefs.IsAlreadyExists(err) {
			return children, nil
		}
		return children, err
	}
}
