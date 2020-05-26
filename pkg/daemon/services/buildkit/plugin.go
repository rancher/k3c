package buildkit

import (
	"path/filepath"

	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/plugin"
	"github.com/containerd/cri"
	"github.com/moby/buildkit/cache/remotecache"
	inlineremotecache "github.com/moby/buildkit/cache/remotecache/inline"
	localremotecache "github.com/moby/buildkit/cache/remotecache/local"
	registryremotecache "github.com/moby/buildkit/cache/remotecache/registry"
	buildkit "github.com/moby/buildkit/cmd/buildkitd/config"
	"github.com/moby/buildkit/control"
	"github.com/moby/buildkit/frontend"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/frontend/gateway"
	"github.com/moby/buildkit/frontend/gateway/forwarder"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/solver/bboltcachestorage"
	"github.com/moby/buildkit/worker"
	"github.com/moby/buildkit/worker/base"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/daemon/config"
)

var (
	Config             = config.DefaultBuildkitConfig()
	PluginRegistration = plugin.Registration{
		ID:     "buildkit",
		Type:   plugin.GRPCPlugin,
		Config: Config,
		Requires: []plugin.Type{
			plugin.RuntimePlugin,
			plugin.ServicePlugin,
		},
		InitFn: PluginInitFunc,
	}
)

func PluginInitFunc(ic *plugin.InitContext) (interface{}, error) {
	log.G(ic.Context).WithField(
		"address", ic.Address,
	).WithField(
		"root", ic.Root,
	).WithField(
		"state", ic.State,
	).Info("BuildKit init")
	cfg := ic.Config.(*buildkit.Config)
	cfg.Workers.Containerd.Address = ic.Address
	if cfg.Workers.Containerd.NetworkConfig.Mode == "cni" {
		cfg.Workers.Containerd.NetworkConfig.CNIBinaryPath = cri.Config.NetworkPluginBinDir
		cfg.Workers.Containerd.NetworkConfig.CNIConfigPath = filepath.Join(cri.Config.NetworkPluginConfDir, "90-k3c.json")

		plugins, err := ic.GetByType(plugin.GRPCPlugin)
		if err != nil {
			return nil, err
		}
		k3cPlugin, ok := plugins["k3c"]
		if !ok {
			return nil, errors.New("failed to find k3c plugin")
		}
		var bridgeName, bridgeCIDR string
		if bridgeName, ok = k3cPlugin.Meta.Exports["bridge-name"]; !ok {
			bridgeName = config.DefaultBridgeName
		}
		if bridgeCIDR, ok = k3cPlugin.Meta.Exports["bridge-cidr"]; !ok {
			bridgeCIDR = config.DefaultBridgeCIDR
		}
		if err := config.WriteFileJSON(cfg.Workers.Containerd.NetworkConfig.CNIConfigPath, config.DefaultCniConf(bridgeName, bridgeCIDR), 0600); err != nil {
			return nil, err
		}
	}
	cfg.Root = ic.Root
	log.G(ic.Context).Debugf("BuildKit config %+v", *cfg)

	// exports
	ic.Meta.Exports["root"] = cfg.Root

	// platforms
	if len(ic.Meta.Platforms) == 0 {
		ic.Meta.Platforms = append(ic.Meta.Platforms, platforms.DefaultSpec())
	}

	controllerOpt := control.Opt{
		WorkerController: &worker.Controller{},
		Entitlements:     cfg.Entitlements,
	}

	defaultWorkerOpt, err := newContainerdWorkerOpt(ic)
	if err != nil {
		return nil, err
	}
	defaultWorker, err := base.NewWorker(defaultWorkerOpt)
	if err != nil {
		return nil, err
	}
	if err := controllerOpt.WorkerController.Add(defaultWorker); err != nil {
		return nil, err
	}

	controllerOpt.SessionManager, err = session.NewManager()
	if err != nil {
		return nil, err
	}

	controllerOpt.CacheKeyStorage, err = bboltcachestorage.NewStore(filepath.Join(cfg.Root, "cache.db"))
	if err != nil {
		return nil, err
	}

	resolverFn := resolverFunc(cfg)
	controllerOpt.Frontends = map[string]frontend.Frontend{
		"dockerfile.v0": forwarder.NewGatewayForwarder(controllerOpt.WorkerController, dockerfile.Build),
		"gateway.v0":    gateway.NewGatewayFrontend(controllerOpt.WorkerController),
	}
	controllerOpt.ResolveCacheExporterFuncs = map[string]remotecache.ResolveCacheExporterFunc{
		"registry": registryremotecache.ResolveCacheExporterFunc(controllerOpt.SessionManager, resolverFn),
		"local":    localremotecache.ResolveCacheExporterFunc(controllerOpt.SessionManager),
		"inline":   inlineremotecache.ResolveCacheExporterFunc(),
	}
	controllerOpt.ResolveCacheImporterFuncs = map[string]remotecache.ResolveCacheImporterFunc{
		"registry": registryremotecache.ResolveCacheImporterFunc(controllerOpt.SessionManager, defaultWorker.ContentStore(), resolverFn),
		"local":    localremotecache.ResolveCacheImporterFunc(controllerOpt.SessionManager),
	}
	return control.NewController(controllerOpt)
}
