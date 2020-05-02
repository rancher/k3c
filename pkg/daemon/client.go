package daemon

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/containers/v1"
	"github.com/containerd/containerd/api/services/diff/v1"
	"github.com/containerd/containerd/api/services/images/v1"
	"github.com/containerd/containerd/api/services/namespaces/v1"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/leases"
	"github.com/containerd/containerd/log"
	cns "github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/plugin"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/services"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/cri/pkg/server"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/daemon/config"
	"github.com/rancher/k3c/pkg/daemon/volume"
	k3c "github.com/rancher/k3c/pkg/defaults"
	"github.com/rancher/k3c/pkg/kicker"
	"github.com/rancher/k3c/pkg/pushstatus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type Daemon struct {
	ctd *containerd.Client
	crt criv1.RuntimeServiceServer
	img criv1.ImageServiceServer
	vol *volume.Manager

	logPath  string
	gck      kicker.Kicker
	lock     sync.Mutex
	pushJobs map[string]pushstatus.Tracker
	tracker  docker.StatusTracker

	getResolver func(context.Context, *client.AuthConfig) (remotes.Resolver, error)
}

func (c *Daemon) CreateVolume(ctx context.Context, name string) (*client.Volume, error) {
	return c.vol.CreateVolume(ctx, name)
}

func (c *Daemon) ListVolumes(ctx context.Context) ([]client.Volume, error) {
	return c.vol.ListVolumes(ctx)
}

func (c *Daemon) RemoveVolume(ctx context.Context, name string, force bool) error {
	return c.vol.RemoveVolume(ctx, name, force)
}

func (c *Daemon) start(ic *plugin.InitContext) error {
	var (
		ctx = cns.WithNamespace(ic.Context, k3c.DefaultNamespace)
	)
	plugins, err := ic.GetByType(plugin.GRPCPlugin)
	if err != nil {
		return err
	}
	criPlugin, ok := plugins["cri"]
	if !ok {
		return errors.New("cannot find CRI plugin")
	}
	criService, err := criPlugin.Instance()
	if err != nil {
		return err
	}

	svc, ok := criService.(server.CRIService)
	if !ok {
		return errors.Errorf("unexpected instance type %T", criService)
	}
	c.getResolver = func(ctx context.Context, authConfig *client.AuthConfig) (remotes.Resolver, error) {
		return svc.GetResolver(ctx, toAuth(authConfig), c.tracker)
	}
	c.crt = svc
	c.img = svc
	c.gck = kicker.New(ctx, true)
	c.vol, err = volume.New(ic.Config.(*config.K3Config).Volumes)
	if err != nil {
		return err
	}

	go c.gc(ctx)

	return nil
}

func (c *Daemon) bootstrap(ic *plugin.InitContext) error {
	var (
		ctx = cns.WithNamespace(ic.Context, k3c.BootstrapNamespace)
		cfg = ic.Config.(*config.K3Config)
	)

	// setup containerd client
	opts, err := getServicesOpts(ic)
	if err != nil {
		return err
	}
	c.ctd, err = containerd.New("", containerd.WithServices(opts...))
	if err != nil {
		return err
	}

	// mark namespace as managed
	err = c.ctd.NamespaceService().SetLabel(ctx, k3c.DefaultNamespace, "io.cri-containerd", "managed")
	if err != nil {
		return err
	}

	if cfg.BootstrapImage == "" || cfg.BootstrapSkip {
		log.G(ctx).Infof("K3C bootstrap skipped")
		return nil
	}

	logrus := log.G(ctx).WithField("namespace", cfg.BootstrapNamespace).WithField("image", cfg.BootstrapImage)

	image, err := c.ctd.GetImage(ctx, cfg.BootstrapImage)
	if errdefs.IsNotFound(err) {
		logrus.Infof("K3C bootstrapping ...")
		image, err = c.ctd.Pull(ctx, cfg.BootstrapImage, containerd.WithPullUnpack)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := c.ctd.Install(ctx, image,
		containerd.WithInstallLibs,
		containerd.WithInstallReplace,
		containerd.WithInstallPath(filepath.Join(ic.Root, "..")),
	); err != nil {
		return err
	}

	logrus.Infof("K3C bootstrapped")
	return nil
}

func (c *Daemon) Close() error {
	if c.ctd == nil {
		return nil
	}
	err := c.ctd.Close()
	if err != nil {
		return err
	}
	c.ctd = nil
	return nil
}

func getServicesOpts(ic *plugin.InitContext) ([]containerd.ServicesOpt, error) {
	plugins, err := ic.GetByType(plugin.ServicePlugin)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plugins")
	}

	opts := []containerd.ServicesOpt{
		containerd.WithEventService(ic.Events),
	}

	for s, fn := range map[string]func(interface{}) containerd.ServicesOpt{
		services.ContentService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithContentStore(s.(content.Store))
		},
		services.ImagesService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithImageService(s.(images.ImagesClient))
		},
		services.SnapshotsService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithSnapshotters(s.(map[string]snapshots.Snapshotter))
		},
		services.ContainersService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithContainerService(s.(containers.ContainersClient))
		},
		services.TasksService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithTaskService(s.(tasks.TasksClient))
		},
		services.DiffService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithDiffService(s.(diff.DiffClient))
		},
		services.NamespacesService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithNamespaceService(s.(namespaces.NamespacesClient))
		},
		services.LeasesService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithLeasesService(s.(leases.Manager))
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
