package defaults

import "github.com/containerd/cri/pkg/constants"

const (
	// DefaultConfigFile is the default location of the containerd configuration file.
	DefaultConfigFile = "/etc/rancher/k3c/config.toml"

	// DefaultRootDir is the default location used by containerd to store persistent data.
	// Used in place of https://github.com/containerd/containerd/blob/v1.3.4/defaults/defaults_unix.go#L24
	DefaultRootDir = "/var/lib/rancher/k3c"

	// DefaultStateDir is the default location used by containerd to store ephemeral data.
	// Used in place of https://github.com/containerd/containerd/blob/v1.3.4/defaults/defaults_unix.go#L27
	DefaultStateDir = "/run/k3c"

	// DefaultAddress is the default location of the containerd unix socket.
	// Used in place of https://github.com/containerd/containerd/blob/v1.3.4/defaults/defaults_unix.go#L29.
	DefaultAddress = DefaultStateDir + "/k3c.sock"

	// PublicNamespace is the default namespace to use when communicating with containerd.
	// See https://github.com/containerd/containerd/blob/v1.3.4/namespaces/context.go#L37.
	PublicNamespace = constants.K8sContainerdNamespace

	// PrivateNamespace is the bootstrap namespace to use when installing via the containerd opt service
	// and building images.
	PrivateNamespace = "k3c.io"

	// DefaultBridgeName is the default name of the network bridge, i.e. docker0.
	DefaultBridgeName = "k3c0"

	// DefaultBridgeCIDR is the default address range of the network bridge.
	DefaultBridgeCIDR = "172.18.0.0/16"

	// DefaultPodLogsDir is the default location where the kublet will place pod logs
	DefaultPodLogsDir = "/var/log/pods"

	// DefaultSandboxImage is the default image that the k3c.io namespace will use
	DefaultSandboxImage = "docker.io/rancher/pause:3.1"
)
