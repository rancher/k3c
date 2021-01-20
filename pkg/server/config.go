package server

import (
	"context"
	"fmt"
	"time"

	"github.com/containerd/containerd"
	buildkit "github.com/moby/buildkit/client"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/version"
	"google.golang.org/grpc"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	defaultAgentPort     = 1233
	defaultAgentImage    = "docker.io/rancher/k3c"
	defaultBuildkitImage = "docker.io/moby/buildkit:v0.8.1"

//	defaultBuildkitPort      = 1234
//	defaultBuildkitAddress   = "unix:///run/buildkit/buildkitd.sock"
//	defaultBuildkitNamespace = "buildkit"
//	defaultContainerdAddress = "/run/k3s/containerd/containerd.sock"
)

var (
	DefaultAgentPort     = defaultAgentPort
	DefaultAgentImage    = defaultAgentImage
	DefaultBuildkitImage = defaultBuildkitImage

//	DefaultBuildkitPort      = defaultBuildkitPort
//	DefaultBuildkitAddress   = defaultBuildkitAddress
//	DefaultBuildkitNamespace = defaultBuildkitNamespace
//	DefaultContainerdAddress = defaultContainerdAddress
//	DefaultListenAddress     = fmt.Sprintf("tcp://0.0.0.0:%d", defaultAgentPort)
)

type Config struct {
	AgentImage        string `usage:"Image to run the agent w/ missing tag inferred from version" default:"docker.io/rancher/k3c"`
	AgentPort         int    `usage:"Port that the agent will listen on" default:"1233"`
	BuildkitImage     string `usage:"BuildKit image for running buildkitd" default:"docker.io/moby/buildkit:v0.8.1"`
	BuildkitNamespace string `usage:"BuildKit namespace in containerd (not 'k8s.io')" default:"buildkit"`
	BuildkitPort      int    `usage:"BuildKit service port" default:"1234"`
	BuildkitSocket    string `usage:"BuildKit socket address" default:"unix:///run/buildkit/buildkitd.sock"`
	ContainerdSocket  string `usage:"Containerd socket address" default:"/run/k3s/containerd/containerd.sock"`
}

func (c *Config) GetAgentImage() string {
	if c.AgentImage == "" {
		c.AgentImage = DefaultAgentImage
	}
	// TODO assumes default agent image is tag-less
	if c.AgentImage == DefaultAgentImage {
		return fmt.Sprintf("%s:%s", c.AgentImage, version.Version)
	}
	return c.AgentImage
}

func (c *Config) GetBuildkitImage() string {
	if c.BuildkitImage == "" {
		c.BuildkitImage = DefaultBuildkitImage
	}
	return c.BuildkitImage
}

func (c *Config) Interface(ctx context.Context, config *client.Config) (*Interface, error) {
	k8s, err := config.Interface()
	if err != nil {
		return nil, err
	}
	server := Interface{
		Kubernetes: k8s,
	}

	server.Buildkit, err = buildkit.New(ctx, c.BuildkitSocket)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix://%s", c.ContainerdSocket), grpc.WithInsecure(), grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
	if err != nil {
		server.Close()
		return nil, err
	}
	server.Containerd, err = containerd.NewWithConn(conn,
		containerd.WithDefaultNamespace(c.BuildkitNamespace),
		containerd.WithTimeout(5*time.Second),
	)
	if err != nil {
		server.Close()
		return nil, err
	}
	server.RuntimeService = criv1.NewRuntimeServiceClient(conn)
	server.ImageService = criv1.NewImageServiceClient(conn)

	return &server, nil
}
