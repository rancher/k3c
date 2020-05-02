package build

import (
	"context"
	"strings"

	bkc "github.com/moby/buildkit/client"
	k3c "github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/defaults"
)

type Client interface {
	Build(ctx context.Context, contextDir string, opts *Opts) (string, error)
	Close() error
}

type buildkitClient struct {
	k3client k3c.Client
	bkclient *bkc.Client
}

func (b *buildkitClient) Close() error {
	return b.bkclient.Close()
}

func New(ctx context.Context, address string) (Client, error) {
	if address == "" {
		address = defaults.DefaultAddress
	}

	k3client, err := k3c.New(ctx, address)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(address, "unix://") {
		address = "unix://" + address
	}

	bkclient, err := bkc.New(ctx, address, bkc.WithFailFast())
	if err != nil {
		return nil, err
	}

	return &buildkitClient{
		k3client: k3client,
		bkclient: bkclient,
	}, nil
}
