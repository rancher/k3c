// +build linux

package buildkit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/containers/v1"
	"github.com/containerd/containerd/api/services/diff/v1"
	"github.com/containerd/containerd/api/services/images/v1"
	"github.com/containerd/containerd/api/services/namespaces/v1"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/gc"
	"github.com/containerd/containerd/leases"
	"github.com/containerd/containerd/plugin"
	"github.com/containerd/containerd/services"
	"github.com/containerd/containerd/snapshots"
	"github.com/moby/buildkit/cache/metadata"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildkitd/config"
	"github.com/moby/buildkit/executor/containerdexecutor"
	"github.com/moby/buildkit/executor/oci"
	containerdsnapshot "github.com/moby/buildkit/snapshot/containerd"
	"github.com/moby/buildkit/util/leaseutil"
	"github.com/moby/buildkit/util/network/cniprovider"
	"github.com/moby/buildkit/util/network/netproviders"
	"github.com/moby/buildkit/util/resolver"
	"github.com/moby/buildkit/util/winlayers"
	"github.com/moby/buildkit/worker/base"
	"github.com/pkg/errors"
)

func newContainerdWorkerOpt(ic *plugin.InitContext) (base.WorkerOpt, error) {
	cfg := (ic.Config).(*config.Config)
	if cfg.Workers.Containerd.Enabled == nil || !*cfg.Workers.Containerd.Enabled {
		return base.WorkerOpt{}, errors.New("containerd worker not enabled")
	}

	servicesOpts, err := getContainerdServicesOpts(ic)
	if err != nil {
		return base.WorkerOpt{}, err
	}

	client, err := containerd.New("",
		containerd.WithDefaultNamespace(cfg.Workers.Containerd.Namespace),
		containerd.WithServices(servicesOpts...),
	)
	if err != nil {
		return base.WorkerOpt{}, err
	}

	snapshotterName := containerd.DefaultSnapshotter

	workerRoot := filepath.Join(ic.Root, fmt.Sprintf("containerd-%s", snapshotterName))
	if err := os.MkdirAll(workerRoot, 0700); err != nil {
		return base.WorkerOpt{}, errors.Wrapf(err, "failed to create %s", workerRoot)
	}

	workerLabels := base.Labels("containerd", snapshotterName)
	for k, v := range cfg.Workers.Containerd.Labels {
		workerLabels[k] = v
	}

	workerID, err := base.ID(workerRoot)
	if err != nil {
		return base.WorkerOpt{}, err
	}

	cniRoot := filepath.Clean(filepath.Join(cfg.Root, ".."))
	networkProviders, err := netproviders.Providers(netproviders.Opt{
		Mode: cfg.Workers.Containerd.NetworkConfig.Mode,
		CNI: cniprovider.Opt{
			Root:       cniRoot,
			ConfigPath: cfg.Workers.Containerd.NetworkConfig.CNIConfigPath,
			BinaryDir:  cfg.Workers.Containerd.NetworkConfig.CNIBinaryPath,
		},
	})
	if err != nil {
		return base.WorkerOpt{}, err
	}

	metadataStore, err := metadata.NewStore(filepath.Join(workerRoot, "metadata_v2.db"))
	if err != nil {
		return base.WorkerOpt{}, err
	}

	workerNS := cfg.Workers.Containerd.Namespace
	workerOpt := base.WorkerOpt{
		ID:            workerID,
		Labels:        workerLabels,
		Platforms:     ic.Meta.Platforms,
		MetadataStore: metadataStore,
		ContentStore:  client.ContentStore(),
		ImageStore:    client.ImageService(),
		LeaseManager:  client.LeasesService(),
		Executor: NamespacedExecutor(
			workerNS, containerdexecutor.New(client, workerRoot, "", networkProviders, getDNSConfig(cfg.DNS)),
		),
		Snapshotter: containerdsnapshot.NewSnapshotter(
			snapshotterName, client.SnapshotService(snapshotterName), workerNS, nil,
		),
		ResolveOptionsFunc: resolverFunc(cfg),
		GCPolicy:           getGCPolicy(cfg.Workers.Containerd.GCConfig, cfg.Root),
	}
	diffService := client.DiffService()
	workerOpt.Applier = winlayers.NewFileSystemApplierWithWindows(workerOpt.ContentStore, diffService)
	workerOpt.Differ = winlayers.NewWalkingDiffWithWindows(workerOpt.ContentStore, diffService)
	workerOpt.GarbageCollect = func(ctx context.Context) (gc.Stats, error) {
		l, err := workerOpt.LeaseManager.Create(ctx)
		if err != nil {
			return nil, nil
		}
		return nil, workerOpt.LeaseManager.Delete(ctx, leases.Lease{ID: l.ID}, leases.SynchronousDelete)
	}
	return workerOpt, nil
}

// getContainerdServicesOpts get service options from plugin context.
func getContainerdServicesOpts(ic *plugin.InitContext) ([]containerd.ServicesOpt, error) {
	cfg := (ic.Config).(*config.Config)
	plugins, err := ic.GetByType(plugin.ServicePlugin)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get service plugin")
	}

	opts := []containerd.ServicesOpt{
		containerd.WithEventService(ic.Events),
	}

	for s, fn := range map[string]func(interface{}) containerd.ServicesOpt{
		services.ContentService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithContentStore(containerdsnapshot.NewContentStore(s.(content.Store), cfg.Workers.Containerd.Namespace))
		},
		services.ImagesService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithImageService(&nsImagesClient{
				n: cfg.Workers.Containerd.Namespace,
				w: s.(images.ImagesClient),
			})
		},
		services.SnapshotsService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithSnapshotters(s.(map[string]snapshots.Snapshotter))
		},
		services.ContainersService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithContainerService(&nsContainersClient{
				n: cfg.Workers.Containerd.Namespace,
				w: s.(containers.ContainersClient),
			})
		},
		services.TasksService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithTaskService(&nsTasksClient{
				n: cfg.Workers.Containerd.Namespace,
				w: s.(tasks.TasksClient),
			})
		},
		services.DiffService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithDiffService(&nsDiffClient{
				n: cfg.Workers.Containerd.Namespace,
				w: s.(diff.DiffClient),
			})
		},
		services.NamespacesService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithNamespaceService(s.(namespaces.NamespacesClient))
		},
		services.LeasesService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithLeasesService(leaseutil.WithNamespace(s.(leases.Manager), cfg.Workers.Containerd.Namespace))
		},
	} {
		p := plugins[s]
		if p == nil {
			return nil, errors.Errorf("service %q not found", s)
		}
		i, err := p.Instance()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance of service %q", s)
		}
		if i == nil {
			return nil, errors.Errorf("instance of service %q not found", s)
		}
		opts = append(opts, fn(i))
	}
	return opts, nil
}

func resolverFunc(cfg *config.Config) resolver.ResolveOptionsFunc {
	m := map[string]resolver.RegistryConf{}
	for k, v := range cfg.Registries {
		m[k] = resolver.RegistryConf{
			Mirrors:   v.Mirrors,
			PlainHTTP: v.PlainHTTP,
		}
	}
	return resolver.NewResolveOptionsFunc(m)
}

func getDNSConfig(cfg *config.DNSConfig) *oci.DNSConfig {
	var dns *oci.DNSConfig
	if cfg != nil {
		dns = &oci.DNSConfig{
			Nameservers:   cfg.Nameservers,
			Options:       cfg.Options,
			SearchDomains: cfg.SearchDomains,
		}
	}
	return dns
}

func getGCPolicy(cfg config.GCConfig, root string) []client.PruneInfo {
	if cfg.GC != nil && !*cfg.GC {
		return nil
	}
	if len(cfg.GCPolicy) == 0 {
		cfg.GCPolicy = config.DefaultGCPolicy(root, cfg.GCKeepStorage)
	}
	out := make([]client.PruneInfo, 0, len(cfg.GCPolicy))
	for _, rule := range cfg.GCPolicy {
		out = append(out, client.PruneInfo{
			Filter:       rule.Filters,
			All:          rule.All,
			KeepBytes:    rule.KeepBytes,
			KeepDuration: time.Duration(rule.KeepDuration) * time.Second,
		})
	}
	return out
}
