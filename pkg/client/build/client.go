package build

import (
	"context"
	"strings"

	"github.com/moby/buildkit/client"
	client2 "github.com/rancher/k3c/pkg/client"
)

type Client interface {
	Build(ctx context.Context, contextDir string, opts *Opts) (string, error)
	Close() error
}

type buildkitClient struct {
	c      client2.Client
	client *client.Client
}

func (b *buildkitClient) Close() error {
	return b.client.Close()
}

func New(ctx context.Context, address string) (Client, error) {
	if address == "" {
		address = client2.DefaultEndpoint
	}

	c, err := client2.New(ctx, address)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(address, "unix://") {
		address = "unix://" + address
	}

	client, err := client.New(ctx, address, client.WithFailFast())
	if err != nil {
		return nil, err
	}

	return &buildkitClient{
		c:      c,
		client: client,
	}, nil
}
