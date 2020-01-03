package daemon

import (
	"context"

	"github.com/rancher/k3c/pkg/daemon/services"
	"github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	remoteserver "github.com/rancher/k3c/pkg/remote/server"
	"google.golang.org/grpc"
)

type Opts = services.Opts

func Start(ctx context.Context, stateDir, rootDir string, opts *Opts) error {
	if opts == nil {
		opts = &Opts{}
	}

	containerdAddress, err := services.StartContainerd(ctx, stateDir, rootDir, opts)
	if err != nil {
		return err
	}

	daemon, err := newDaemon(ctx, containerdAddress)
	if err != nil {
		return err
	}

	cb := func(server *grpc.Server) error {
		v1alpha1.RegisterContainerServiceServer(server, remoteserver.NewContainerService(daemon))
		return nil
	}

	err = services.StartBuildkitd(ctx, containerdAddress, cb, stateDir, rootDir, opts)
	if err != nil {
		return err
	}

	return nil
}
