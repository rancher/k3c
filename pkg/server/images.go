package server

import (
	"context"

	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// List images server-side impl
func (i *Interface) List(ctx context.Context, req *imagesv1.ImageListRequest) (*imagesv1.ImageListResponse, error) {
	res, err := i.ImageService.ListImages(ctx, &criv1.ListImagesRequest{Filter: req.Filter})
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImageListResponse{
		Images: res.Images,
	}, nil
}

// Status of an image server-side impl (unused)
func (i *Interface) Status(ctx context.Context, req *imagesv1.ImageStatusRequest) (*imagesv1.ImageStatusResponse, error) {
	res, err := i.ImageService.ImageStatus(ctx, &criv1.ImageStatusRequest{Image: req.Image})
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImageStatusResponse{
		Image: res.Image,
	}, nil
}

// Remove image server-side impl
func (i *Interface) Remove(ctx context.Context, req *imagesv1.ImageRemoveRequest) (*imagesv1.ImageRemoveResponse, error) {
	_, err := i.ImageService.RemoveImage(ctx, &criv1.RemoveImageRequest{Image: req.Image})
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImageRemoveResponse{}, nil
}
