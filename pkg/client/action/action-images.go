package action

import (
	"context"
	"fmt"

	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/k3c/pkg/client"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type ListImages struct {
	All     bool   `usage:"Ignored (compatibility)" short:"a"`
	Digests bool   `usage:"Show digests"`
	Filter  string `usage:"Filter output based on conditions provided" short:"f"`
	Format  string `usage:"Pretty-print images using a Go template"`
	NoTrunc bool   `usage:"Don't truncate output"`
	Quiet   bool   `usage:"Only show image IDs" short:"q"`
}

func (s *ListImages) Invoke(ctx context.Context, k8s *client.Interface, names []string) error {
	return DoImages(ctx, k8s, func(ctx context.Context, imagesClient imagesv1.ImagesClient) error {
		// TODO tabular listing
		req := &imagesv1.ImageListRequest{}
		// TODO filtering not working as expected
		if len(names) > 0 {
			req.Filter = &criv1.ImageFilter{
				Image: &criv1.ImageSpec{
					Image: names[0],
				},
			}
		}
		res, err := imagesClient.List(ctx, req)
		if err != nil {
			return err
		}
		for _, image := range res.Images {
			for _, tag := range image.RepoTags {
				fmt.Println(tag)
			}
		}
		return nil
	})
}
