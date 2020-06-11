package daemon

import (
	"os"
	"time"

	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/plugin"
	"github.com/rancher/k3c/pkg/daemon/config"
	"github.com/rancher/k3c/pkg/daemon/services/buildkit"
	"github.com/rancher/k3c/pkg/pushstatus"
	"github.com/rancher/k3c/pkg/remote/server"
)

var (
	Config             = config.DefaultK3Config()
	PluginRegistration = plugin.Registration{
		ID:     "k3c",
		Type:   plugin.GRPCPlugin,
		Config: Config,
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
	log.G(ic.Context).WithField(
		"address", ic.Address,
	).WithField(
		"root", ic.Root,
	).WithField(
		"state", ic.State,
	).Info("K3C init")

	cfg := ic.Config.(*config.K3Config)
	log.G(ic.Context).Debugf("K3C config %+v", *cfg)

	// exports
	ic.Meta.Exports["api-version"] = "v1alpha1"
	ic.Meta.Exports["bridge-name"] = cfg.BridgeName
	ic.Meta.Exports["bridge-cidr"] = cfg.BridgeCIDR
	ic.Meta.Exports["pod-logs-dir"] = cfg.PodLogs
	ic.Meta.Exports["volumes-dir"] = cfg.Volumes

	// platforms
	if len(ic.Meta.Platforms) == 0 {
		ic.Meta.Platforms = append(ic.Meta.Platforms, platforms.DefaultSpec())
	}

	daemon := &Daemon{
		logPath:  cfg.PodLogs,
		pushJobs: map[string]pushstatus.Tracker{},
	}
	// bootstrap in the foreground so that buildkit will have the binaries it needs
	if err := daemon.bootstrap(ic); err != nil {
		return nil, err
	}
	if err := daemon.start(ic); err != nil {
		return nil, err
	}
	service := server.NewContainerService(daemon)
	service.SetInitialized(true)
	log.G(ic.Context).WithField("bridge", cfg.BridgeName).WithField("cidr", cfg.BridgeCIDR).Info("K3C daemon")
	go func() {
		var (
			addr = config.Socket.Address
			gid  = config.Socket.GID
			uid  = config.Socket.UID
		)
		for {
			select {
			case <-time.After(100 * time.Millisecond):
				err := os.Chown(addr, uid, gid)
				if os.IsNotExist(err) {
					continue
				}
				log := log.G(ic.Context).WithField("address", addr).WithField("gid", gid).WithField("uid", uid)
				if err != nil {
					log.WithError(err).Warn("K3C socket")
				} else {
					log.Debug("K3C socket")
				}
				return
			case <-ic.Context.Done():
				return
			}
		}
	}()
	return service, nil
}
