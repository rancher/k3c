package daemon

import (
	"context"
	"io"
	"path/filepath"
	"sync"

	"github.com/containerd/containerd"
	eventspb "github.com/containerd/containerd/api/events"
	containerspb "github.com/containerd/containerd/api/services/containers/v1"
	diffpb "github.com/containerd/containerd/api/services/diff/v1"
	imagespb "github.com/containerd/containerd/api/services/images/v1"
	namespacespb "github.com/containerd/containerd/api/services/namespaces/v1"
	taskspb "github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/events/exchange"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/leases"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/plugin"
	"github.com/containerd/containerd/services"
	"github.com/containerd/containerd/snapshots"
	criutil "github.com/containerd/cri/pkg/containerd/util"
	"github.com/containerd/cri/pkg/server"
	"github.com/containerd/typeurl"
	prototypes "github.com/gogo/protobuf/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/daemon/config"
	"github.com/rancher/k3c/pkg/daemon/volume"
	"github.com/rancher/k3c/pkg/defaults"
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
		ctx = criutil.WithUnlisted(namespaces.WithNamespace(ic.Context, defaults.PublicNamespace), defaults.PrivateNamespace)
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
	c.crt = svc
	c.img = svc
	c.gck = kicker.New(ctx, true)
	c.vol, err = volume.New(ic.Config.(*config.K3Config).Volumes)
	if err != nil {
		return err
	}

	go c.gc(ctx)
	go c.syncImages(namespaces.WithNamespace(ic.Context, defaults.PrivateNamespace), ic.Events)

	return nil
}

func (c *Daemon) bootstrap(ic *plugin.InitContext) error {
	var (
		ctx = namespaces.WithNamespace(ic.Context, k3c.PrivateNamespace)
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

	if cfg.BootstrapImage == "" || cfg.BootstrapSkip {
		log.G(ctx).Infof("K3C bootstrap skipped")
		return nil
	}

	logrus := log.G(ctx).WithField("namespace", defaults.PrivateNamespace).WithField("image", cfg.BootstrapImage)

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

// syncImages listens for ImageCreate events in the private namespace and copies them to the public namespace
// based on the assumption that ImageCreate events are from images built by buildkit
func (c *Daemon) syncImages(ctx context.Context, ex *exchange.Exchange) {
	evtch, errch := ex.Subscribe(ctx, `topic~="/images/"`)
	for {
		select {
		case err, ok := <-errch:
			if !ok {
				return
			}
			log.G(ctx).WithError(err).Error("image sync listener")
		case evt, ok := <-evtch:
			if !ok {
				return
			}
			if evt.Namespace != defaults.PrivateNamespace {
				continue
			}
			if err := c.handleEvent(ctx, evt.Event); err != nil {
				log.G(ctx).WithError(err).Error("image sync handler")
			}
		case <-ctx.Done():
			log.G(ctx).WithError(ctx.Err()).Error("image sync handler")
			return
		}
	}
}

func (c *Daemon) handleEvent(ctx context.Context, any *prototypes.Any) error {
	evt, err := typeurl.UnmarshalAny(any)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal any")
	}

	switch e := evt.(type) {
	case *eventspb.ImageCreate:
		log.G(ctx).WithField("event", "image.create").Debug(e.Name)
		return c.handleImageCreate(ctx, e.Name)
	}

	return nil
}

func (c *Daemon) handleImageCreate(ctx context.Context, name string) error {
	imageStore := c.ctd.ImageService()
	img, err := imageStore.Get(ctx, name)
	if err != nil {
		return err
	}
	contentStore := c.ctd.ContentStore()
	otherContext := namespaces.WithNamespace(ctx, defaults.PublicNamespace)
	var copy images.HandlerFunc = func(ctx context.Context, desc ocispec.Descriptor) (subdescs []ocispec.Descriptor, err error) {
		log.G(ctx).WithField("media-type", desc.MediaType).Debug(desc.Digest)
		info, err := contentStore.Info(ctx, desc.Digest)
		if err != nil {
			return subdescs, err
		}
		if _, err = contentStore.Info(otherContext, desc.Digest); err != nil && !errdefs.IsNotFound(err) {
			return subdescs, err
		}
		ra, err := contentStore.ReaderAt(ctx, desc)
		if err != nil {
			return subdescs, err
		}
		defer ra.Close()
		r := content.NewReader(ra)
		w, err := contentStore.Writer(otherContext, content.WithRef(img.Name))
		if err != nil {
			return subdescs, err
		}
		defer w.Close()
		if _, err = io.Copy(w, r); err != nil {
			return subdescs, err
		}
		if err = w.Commit(otherContext, 0, w.Digest(), content.WithLabels(info.Labels)); err != nil && errdefs.IsAlreadyExists(err) {
			return subdescs, nil
		}
		return subdescs, err
	}
	err = images.Walk(ctx, images.Handlers(images.ChildrenHandler(contentStore), copy), img.Target)
	if err != nil {
		return err
	}
	_, err = imageStore.Create(otherContext, img)
	if errdefs.IsAlreadyExists(err) {
		return nil
	}
	return err
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
			return containerd.WithImageService(s.(imagespb.ImagesClient))
		},
		services.SnapshotsService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithSnapshotters(s.(map[string]snapshots.Snapshotter))
		},
		services.ContainersService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithContainerService(s.(containerspb.ContainersClient))
		},
		services.TasksService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithTaskService(s.(taskspb.TasksClient))
		},
		services.DiffService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithDiffService(s.(diffpb.DiffClient))
		},
		services.NamespacesService: func(s interface{}) containerd.ServicesOpt {
			return containerd.WithNamespaceService(s.(namespacespb.NamespacesClient))
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
