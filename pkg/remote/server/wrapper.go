package server

import (
	"context"
	"errors"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	api "github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
)

var _ api.ContainerServiceServer = &wrapper{svc: nil}

type wrapper struct {
	svc *server
}

// checkInitialized returns error if the server is not fully initialized.
// GRPC service request handlers should return error before server is fully
// initialized.
// NOTE: All following functions MUST check initialized at the beginning.
func (w *wrapper) checkInitialized() error {
	if w.svc.initialized.IsSet() {
		return nil
	}
	return errors.New("server is not initialized yet")
}

func (w *wrapper) ListPods(ctx context.Context, req *api.ListPodsRequest) (res *api.ListPodsResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Trace("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("ListPods failed")
		} else {
			log.G(ctx).Tracef("ListPods returns %+v", res.GetPods())
		}
	}()
	res, err = w.svc.ListPods(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) CreatePod(ctx context.Context, req *api.CreatePodRequest) (res *api.CreatePodResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("CreatePod failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.CreatePod(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) Attach(ctx context.Context, req *api.AttachRequest) (res *api.StreamResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("Attach failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.Attach(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) Exec(ctx context.Context, req *api.ExecRequest) (res *api.StreamResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("Exec failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.Exec(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) CreateContainer(ctx context.Context, req *api.CreateContainerRequest) (res *api.CreateContainerResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("CreateContainer failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.CreateContainer(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) StartContainer(ctx context.Context, req *api.StartContainerRequest) (res *api.StartContainerResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("StartContainer failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.StartContainer(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) StopContainer(ctx context.Context, req *api.StopContainerRequest) (res *api.StopContainerResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("StopContainer failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.StopContainer(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) RemoveContainer(ctx context.Context, req *api.RemoveContainerRequest) (res *api.RemoveContainerResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("RemoveContainer failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.RemoveContainer(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) GetContainer(ctx context.Context, req *api.GetContainerRequest) (res *api.GetContainerResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("GetContainer failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.GetContainer(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) LogContainer(req *api.LogContainerRequest, res api.ContainerService_LogContainerServer) (err error) {
	ctx := res.Context()
	if err := w.checkInitialized(); err != nil {
		return err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("LogContainer failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	err = w.svc.LogContainer(req, res)
	return errdefs.ToGRPC(err)
}

func (w *wrapper) ListImages(ctx context.Context, req *api.ListImagesRequest) (res *api.ListImagesResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("ListImages failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.ListImages(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) RemoveImage(ctx context.Context, req *api.RemoveImageRequest) (res *api.RemoveImageResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("RemoveImage failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.RemoveImage(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) PullImage(ctx context.Context, req *api.PullImageRequest) (res *api.PullImageResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("PullImage failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.PullImage(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) PushImage(ctx context.Context, req *api.PushImageRequest) (res *api.PushImageResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("PushImage failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.PushImage(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) GetImage(ctx context.Context, req *api.GetImageRequest) (res *api.GetImageResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("GetImage failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.GetImage(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) TagImage(ctx context.Context, req *api.TagImageRequest) (res *api.TagImageResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("TagImage failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.TagImage(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) PullProgress(req *api.InfoRequest, res api.ContainerService_PullProgressServer) (err error) {
	ctx := res.Context()
	if err := w.checkInitialized(); err != nil {
		return err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("PullProgress failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	err = w.svc.PullProgress(req, res)
	return errdefs.ToGRPC(err)
}

func (w *wrapper) PushProgress(req *api.InfoRequest, res api.ContainerService_PushProgressServer) (err error) {
	ctx := res.Context()
	if err := w.checkInitialized(); err != nil {
		return err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("PushProgress failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	err = w.svc.PushProgress(req, res)
	return errdefs.ToGRPC(err)
}

func (w *wrapper) ListVolumes(ctx context.Context, req *api.ListVolumesRequest) (res *api.ListVolumesResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("ListVolumes failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.ListVolumes(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) CreateVolume(ctx context.Context, req *api.CreateVolumeRequest) (res *api.CreateVolumeResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("CreateVolume failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.CreateVolume(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) RemoveVolume(ctx context.Context, req *api.RemoveVolumeRequest) (res *api.RemoveVolumeResponse, err error) {
	if err := w.checkInitialized(); err != nil {
		return nil, err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("RemoveVolume failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	res, err = w.svc.RemoveVolume(ctx, req)
	return res, errdefs.ToGRPC(err)
}

func (w *wrapper) Events(req *api.EventsRequest, res api.ContainerService_EventsServer) (err error) {
	ctx := res.Context()
	if err := w.checkInitialized(); err != nil {
		return err
	}
	log.G(ctx).Tracef("%+v", req)
	defer func() {
		if err != nil {
			log.G(ctx).WithError(err).Error("Events failed")
		} else {
			log.G(ctx).Tracef("%+v", res)
		}
	}()
	err = w.svc.Events(req, res)
	return errdefs.ToGRPC(err)
}
