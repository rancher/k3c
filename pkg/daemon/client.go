package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/rancher/k3c/pkg/kicker"

	"github.com/containerd/containerd"
	"github.com/rancher/k3c/pkg/daemon/volume"
	"github.com/rancher/k3c/pkg/endpointconn"
	"github.com/rancher/k3c/pkg/pushstatus"
	"google.golang.org/grpc"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

var (
	Timeout                    = 2 * time.Second
	DefaultContainerdEndpoint  = "/run/k3c/containerd/containerd.sock"
	DefaultContainerdNamespace = "k3c.io"
	DefaultLogs                = "/var/log/pods"
	DefaultVolumeDir           = "/var/lib/rancher/k3c/volumes"
)

type Daemon struct {
	*volume.Manager

	namespace string
	logPath   string
	conn      *grpc.ClientConn
	cClient   *containerd.Client
	runtime   pb.RuntimeServiceClient
	image     pb.ImageServiceClient
	gcKick    kicker.Kicker

	lock     sync.Mutex
	pushJobs map[string]pushstatus.Tracker
}

func newDaemon(ctx context.Context, endpoint string) (*Daemon, error) {
	if endpoint == "" {
		endpoint = DefaultContainerdEndpoint
	}
	runtimeConn, err := endpointconn.Get(ctx, endpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	c, err := containerdClient(endpoint)
	if err != nil {
		return nil, err
	}

	volManager, err := volume.New(DefaultVolumeDir)
	if err != nil {
		return nil, err
	}

	d := &Daemon{
		Manager:   volManager,
		cClient:   c,
		namespace: "k3c",
		logPath:   DefaultLogs,
		conn:      runtimeConn,
		runtime:   pb.NewRuntimeServiceClient(runtimeConn),
		image:     pb.NewImageServiceClient(runtimeConn),
		pushJobs:  map[string]pushstatus.Tracker{},
		gcKick:    kicker.New(ctx, true),
	}
	go d.gc(ctx)
	return d, nil
}

func containerdClient(endpoint string) (*containerd.Client, error) {
	if endpoint == "" {
		endpoint = DefaultContainerdEndpoint
	}
	return containerd.New(endpoint, containerd.WithDefaultNamespace(DefaultContainerdNamespace))
}

func (c *Daemon) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	if err != nil {
		return err
	}
	c.conn = nil
	return nil
}
