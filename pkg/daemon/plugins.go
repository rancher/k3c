package daemon

import (
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/plugin"
	"github.com/rancher/k3c/pkg/daemon/config"
	"github.com/rancher/k3c/pkg/daemon/services/buildkit"
	"github.com/rancher/k3c/pkg/kicker"
	"github.com/rancher/k3c/pkg/pushstatus"
	"github.com/rancher/k3c/pkg/remote/server"
)

var (
	PluginRegistration = plugin.Registration{
		ID:     "k3c",
		Type:   plugin.GRPCPlugin,
		Config: config.DefaultK3Config(),
		Requires: []plugin.Type{
			plugin.InternalPlugin,
			plugin.ServicePlugin,
		},
		InitFn: PluginInitFunc,
	}
)

func init() {
	// registration order is important because k3c needs to come up after cri but before buildkit
	plugin.Register(&PluginRegistration)
	plugin.Register(&buildkit.PluginRegistration)
}

func PluginInitFunc(ic *plugin.InitContext) (interface{}, error) {
	ctx := ic.Context
	log.G(ctx).WithField(
		"address", ic.Address,
	).WithField(
		"root", ic.Root,
	).WithField(
		"state", ic.State,
	).Info("Init K3C Plugin")

	cfg := ic.Config.(*config.K3Config)
	log.G(ctx).Debugf("Init K3C Plugin with config %+v", cfg)

	ic.Meta.Exports["K3CVersion"] = "v1alpha1"
	ic.Meta.Platforms = append(ic.Meta.Platforms, platforms.DefaultSpec())

	log.G(ctx).WithField("bridge", cfg.BridgeName).WithField("cidr", cfg.BridgeCIDR).Info("Start K3C...")

	daemon := &Daemon{
		logPath:  cfg.PodLogs,
		pushJobs: map[string]pushstatus.Tracker{},
		gcKick:   kicker.New(ctx, true),
	}
	// bootstrap in the foreground so that buildkit will have the binaries it needs
	if err := daemon.Bootstrap(ic); err != nil {
		log.G(ctx).WithError(err).Error("K3C failed to bootstrap")
	}

	service := server.NewContainerService(daemon)
	// connect to the grpc socket in the background
	// because it isn't started until all plugins get a chance to register
	// TODO(dweomer): this could be avoided if cri provided service references the same way containerd does
	go func() {
		if err := daemon.Start(ctx, *cfg, ic.Address); err != nil {
			log.G(ctx).WithError(err).Fatal("K3C failed to start")
		}
		service.SetInitialized(true)
	}()
	return service, nil
}
