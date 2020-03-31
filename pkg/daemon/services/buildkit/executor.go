package buildkit

import (
	"context"
	"io"

	"github.com/containerd/containerd/namespaces"
	"github.com/moby/buildkit/cache"
	"github.com/moby/buildkit/executor"
)

type nsExecutor struct {
	n string
	w executor.Executor
}

func NamespacedExecutor(ns string, exec executor.Executor) executor.Executor {
	return &nsExecutor{
		n: ns, w: exec,
	}
}

func (n nsExecutor) Exec(ctx context.Context, meta executor.Meta, rootfs cache.Mountable, mounts []executor.Mount, stdin io.ReadCloser, stdout, stderr io.WriteCloser) error {
	return n.w.Exec(namespaces.WithNamespace(ctx, n.n), meta, rootfs, mounts, stdin, stdout, stderr)
}
