package daemon

import (
	"github.com/rancher/k3c/pkg/daemon"
	"github.com/rancher/k3c/pkg/version"
	"github.com/urfave/cli/v2"
)

type Daemon struct {
	ContainerdAddr string `usage:"Use external containerd at this address"`
	BuildkitdAddr  string `usage:"Use external buildkitd at this address"`
	RootDir        string `usage:"Path used for persistent state" default:"/var/lib/rancher/k3c"`
	BootstrapImage string `usage:"Bootstrap image" default:"index.docker.io/rancher/k3c-data"`
	StateDir       string `usage:"Path used for ephemeral runtime state" default:"/run/k3c"`
	Config         string `usage:"Config toml used for k3c/containerd/buildkitd" default:"/etc/rancher/k3c/config.toml"`
	Bridge         string `usage:"Name of the bridge to create for networking" default:"k3c0"`
	BridgeCidr     string `usage:"Bridge CIDR" default:"172.18.0.0/16"`
	G_Group        string `usage:"System group to assign the socket go"`
}

func (d *Daemon) Run(ctx *cli.Context) (err error) {
	opts := &daemon.Opts{
		ExtraConfig:    d.Config,
		BootstrapImage: d.BootstrapImage,
		BridgeName:     d.Bridge,
		BridgeCIDR:     d.BridgeCidr,
		Group:          d.G_Group,
	}

	if opts.BootstrapImage == "index.docker.io/rancher/k3c-data" {
		opts.BootstrapImage = "index.docker.io/rancher/k3c-data:" + version.Version
	}

	err = daemon.Start(ctx.Context, d.StateDir, d.RootDir, opts)
	if err != nil {
		return err
	}

	<-ctx.Context.Done()
	return ctx.Context.Err()
}
