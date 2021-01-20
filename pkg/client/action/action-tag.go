package action

import (
	"context"

	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/k3c/pkg/client"
	"github.com/sirupsen/logrus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type TagImage struct {
}

func (s *TagImage) Invoke(ctx context.Context, k8s *client.Interface, image string, tags []string) error {
	return DoImages(ctx, k8s, func(ctx context.Context, imagesClient imagesv1.ImagesClient) error {
		req := &imagesv1.ImageTagRequest{
			Image: &criv1.ImageSpec{
				Image: image,
			},
			Tags: tags,
		}
		res, err := imagesClient.Tag(ctx, req)
		if err != nil {
			return err
		}
		logrus.Debugf("%#v", res)
		return nil
	})
}
