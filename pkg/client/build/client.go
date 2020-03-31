package build

import (
	"context"
	"strings"

	bkc "github.com/moby/buildkit/client"
	k3c "github.com/rancher/k3c/pkg/client"
)

type Client interface {
	Build(ctx context.Context, contextDir string, opts *Opts) (string, error)
	Close() error
}

type buildkitClient struct {
	c      k3c.Client
	client *bkc.Client
}

func (b *buildkitClient) Close() error {
	return b.client.Close()
}

func New(ctx context.Context, address string) (Client, error) {
	if address == "" {
		address = k3c.DefaultEndpoint
	}

	c, err := k3c.New(ctx, address)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(address, "unix://") {
		address = "unix://" + address
	}

	client, err := bkc.New(ctx, address, bkc.WithFailFast())
	if err != nil {
		return nil, err
	}

	return &buildkitClient{
		c:      c,
		client: client,
	}, nil
}
