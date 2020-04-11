package buildkit

import (
	"os"
	"path/filepath"

	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/plugin"
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
)

var (
	PluginRegistration = plugin.Registration{
		ID:     "buildkit",
		Type:   plugin.GRPCPlugin,
		Config: &buildkit.Config{},
		Requires: []plugin.Type{
			plugin.RuntimePlugin,
			plugin.ServicePlugin,
		},
		InitFn: PluginInitFunc,
	}
)

func PluginInitFunc(ic *plugin.InitContext) (interface{}, error) {
	ctx := ic.Context
	log.G(ctx).WithField(
		"address", ic.Address,
	).WithField(
		"root", ic.Root,
	).WithField(
		"state", ic.State,
	).Info("Init BuildKit Plugin")

	cfg := ic.Config.(*buildkit.Config)
	log.G(ctx).Debugf("Init BuildKit Plugin with config %+v", cfg)

	ic.Meta.Exports["root"] = ic.Root
	ic.Meta.Platforms = append(ic.Meta.Platforms, platforms.DefaultSpec())

	if err := os.MkdirAll(ic.Root, 0711); err != nil {
		return nil, err
	}
	cfg.Root = ic.Root

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
