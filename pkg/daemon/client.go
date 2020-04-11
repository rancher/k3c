package daemon

import (
	"context"
	"os"
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
	"github.com/containerd/containerd/services"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/cri/pkg/constants"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/daemon/config"
	"github.com/rancher/k3c/pkg/daemon/volume"
	"github.com/rancher/k3c/pkg/endpointconn"
	"github.com/rancher/k3c/pkg/kicker"
	"github.com/rancher/k3c/pkg/pushstatus"
	"google.golang.org/grpc"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type Daemon struct {
	*volume.Manager

	logPath string
	cClient *containerd.Client
	runtime criv1.RuntimeServiceClient
	image   criv1.ImageServiceClient
	gcKick  kicker.Kicker

	lock     sync.Mutex
	pushJobs map[string]pushstatus.Tracker
}

func (c *Daemon) Start(ctx context.Context, cfg config.K3Config, address string) error {
	conn, err := endpointconn.Get(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	ccontainerdClient, err := containerd.NewWithConn(conn, containerd.WithDefaultNamespace(constants.K8sContainerdNamespace))
	if err != nil {
		return err
	}
	c.cClient = ccontainerdClient
	c.runtime = criv1.NewRuntimeServiceClient(conn)
	c.image = criv1.NewImageServiceClient(conn)

	volManager, err := volume.New(cfg.Volumes)
	if err != nil {
		defer c.Close()
		return err
	}
	c.Manager = volManager
	go c.gc(ctx)
	return nil
}

func (c *Daemon) Bootstrap(ic *plugin.InitContext) error {
	var (
		ctx = ic.Context
		cfg = ic.Config.(*config.K3Config)
	)
	if cfg.BootstrapImage == "" {
		log.G(ctx).Infof("K3C bootstrapping skipped")
		return nil
	}
	opts, err := getServicesOpts(ic)
	if err != nil {
		return err
	}
	client, err := containerd.New("", containerd.WithServices(opts...))
	if err != nil {
		return err
	}
	log := log.G(ctx).WithField("namespace", cfg.BootstrapNamespace).WithField("image", cfg.BootstrapImage)
	ctx = cns.WithNamespace(ctx, cfg.BootstrapNamespace)
	image, err := client.GetImage(ctx, cfg.BootstrapImage)
	if errdefs.IsNotFound(err) {
		log.Infof("K3C bootstrapping data...")
		image, err = client.Pull(ctx, cfg.BootstrapImage, containerd.WithPullUnpack)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	path, err := getInstallPath(ic)
	if err != nil {
		if path = os.Getenv("K3C_ROOT"); path == "" {
			return err
		}
		log.Warn(err)
	}
	if err := client.Install(ctx, image,
		containerd.WithInstallLibs,
		containerd.WithInstallReplace,
		containerd.WithInstallPath(path),
	); err != nil {
		return err
	}

	log.Infof("K3C bootstrapping done")
	return nil
}

func (c *Daemon) Close() error {
	if c.cClient == nil {
		return nil
	}
	err := c.cClient.Close()
	if err != nil {
		return err
	}
	c.cClient = nil
	return nil
}

func getInstallPath(ic *plugin.InitContext) (string, error) {
	plugins, err := ic.GetByType(plugin.InternalPlugin)
	if err != nil {
		return "", errors.Wrap(err, "failed to get plugins")
	}
	for _, plugin := range plugins {
		if plugin.Registration.ID == "opt" {
			if plugin.Registration.Disable {
				return "", errors.New("opt plugin disabled")
			}
			return plugin.Meta.Exports["path"], nil
		}
	}
	return "", errors.New("opt plugin unavailable")
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
