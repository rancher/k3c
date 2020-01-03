package rmi

import (
	"context"
	"fmt"
	"strings"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/client"
	"github.com/urfave/cli/v2"
)

type Rmi struct {
}

func (r *Rmi) Run(ctx *cli.Context) error {
	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	success := true
	for _, image := range ctx.Args().Slice() {
		err := r.removeImage(ctx.Context, client, image)
		if err == nil {
			fmt.Println(image)
		} else {
			success = false
			fmt.Printf("Error: %v: %s\n", err, image)
		}
	}

	if !success {
		return cli.Exit("", 1)
	}

	return nil
}

func (r *Rmi) removeImage(ctx context.Context, client client.Client, imageName string) error {
	if strings.HasPrefix(imageName, "sha256:") {
		images, err := client.ListImages(ctx)
		if err != nil {
			return err
		}

		for _, image := range images {
			for _, digest := range image.Digests {
				if strings.HasSuffix(digest, "@"+imageName) {
					if err := client.RemoveImage(ctx, image.ID); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}
	return client.RemoveImage(ctx, imageName)
}
