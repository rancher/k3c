package config

import (
	"path/filepath"
)

var (
	DefaultDaemonConfigFile = "/etc/rancher/k3c/config.toml"
	DefaultDaemonRootDir    = "/var/lib/rancher/k3c"
	DefaultDaemonStateDir   = "/run/k3c"

	DefaultBridgeName         = "k3c0"
	DefaultBridgeCIDR         = "172.18.0.0/16"
	DefaultBootstrapNamespace = "k3c.io"
	DefaultPodLogsDir         = "/var/log/pods"
	DefaultVolumesDir         = filepath.Join(DefaultDaemonRootDir, "volumes")
)

type K3Config struct {
	BootstrapImage     string `toml:"bootstrap_image"`
	BootstrapNamespace string `toml:"bootstrap_namespace"`
	BridgeName         string `toml:"bridge_name"`
	BridgeCIDR         string `toml:"bridge_cidr"`
	PodLogs            string `toml:"pod_logs"`
	Volumes            string `toml:"volumes"`
}

func DefaultK3Config() *K3Config {
	return &K3Config{
		BridgeName:         DefaultBridgeName,
		BridgeCIDR:         DefaultBridgeCIDR,
		BootstrapNamespace: DefaultBootstrapNamespace,
		BootstrapImage:     "index.docker.io/dweomer/k3c-data:dev",
		PodLogs:            DefaultPodLogsDir,
		Volumes:            DefaultVolumesDir,
	}
}
