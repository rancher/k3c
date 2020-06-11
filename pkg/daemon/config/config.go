package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/services/server/config"
	buildkit "github.com/moby/buildkit/cmd/buildkitd/config"
	"github.com/rancher/k3c/pkg/defaults"
)

var (
	DefaultDaemonConfigFile = defaults.DefaultConfigFile
	DefaultDaemonRootDir    = defaults.DefaultRootDir
	DefaultDaemonStateDir   = defaults.DefaultStateDir
	DefaultDaemonAddress    = DefaultDaemonStateDir + filepath.Base(defaults.DefaultAddress)

	DefaultBridgeName     = defaults.DefaultBridgeName
	DefaultBridgeCIDR     = defaults.DefaultBridgeCIDR
	DefaultBootstrapImage = "docker.io/rancher/k3c:latest"

	DefaultSandboxImage = defaults.DefaultSandboxImage
	DefaultPodLogsDir   = defaults.DefaultPodLogsDir
	DefaultVolumesDir   = filepath.Join(DefaultDaemonRootDir, "volumes")
)

var (
	Socket config.GRPCConfig
)

type K3Config struct {
	BootstrapSkip  bool   `toml:"bootstrap_skip"`
	BootstrapImage string `toml:"bootstrap_image"`
	BridgeName     string `toml:"bridge_name"`
	BridgeCIDR     string `toml:"bridge_cidr"`
	PodLogs        string `toml:"pod_logs"`
	Volumes        string `toml:"volumes"`
}

func DefaultBuildkitConfig() *buildkit.Config {
	config := buildkit.Config{}
	containerdEnabled := true
	config.Workers.Containerd = buildkit.ContainerdConfig{
		Enabled: &containerdEnabled,
		Platforms: []string{
			platforms.DefaultString(),
		},
		Namespace: defaults.PrivateNamespace,
	}
	config.Workers.Containerd.NetworkConfig.Mode = "host"
	return &config
}

func DefaultK3Config() *K3Config {
	config := &K3Config{
		BridgeName:     DefaultBridgeName,
		BridgeCIDR:     DefaultBridgeCIDR,
		BootstrapImage: DefaultBootstrapImage,
		PodLogs:        DefaultPodLogsDir,
		Volumes:        DefaultVolumesDir,
	}
	return config
}

func DefaultCniConf(bridge, cidr string) map[string]interface{} {
	return map[string]interface{}{
		"cniVersion":  "0.3.1",
		"type":        "bridge",
		"name":        "k3c-net",
		"bridge":      bridge,
		"isGateway":   true,
		"ipMasq":      true,
		"promiscMode": true,
		"ipam": map[string]interface{}{
			"type":   "host-local",
			"subnet": cidr,
			"routes": []map[string]interface{}{
				{
					"dst": "0.0.0.0/0",
				},
			},
		},
	}
}

func DefaultCniConflist(bridge, cidr string) map[string]interface{} {
	return map[string]interface{}{
		"cniVersion": "0.3.1",
		"name":       "k3c-net",
		"plugins": []map[string]interface{}{
			DefaultCniConf(bridge, cidr),
			{
				"type": "portmap",
				"capabilities": map[string]interface{}{
					"portMappings": true,
				},
			},
		},
	}
}

func WriteFileJSON(path string, data interface{}, mode os.FileMode) error {
	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return err
	}
	return ioutil.WriteFile(path, buf.Bytes(), mode)
}

func WriteFileTOML(path string, data interface{}, mode os.FileMode) error {
	buf := bytes.Buffer{}
	if err := toml.NewEncoder(&buf).Encode(data); err != nil {
		return err
	}
	return ioutil.WriteFile(path, buf.Bytes(), mode)
}
