package config

import (
	"path/filepath"

	"github.com/containerd/containerd/plugin"
	cri "github.com/containerd/cri/pkg/config"
)

var (
	DefaultCriSandboxImage = "docker.io/rancher/pause:3.1"
)

func DefaultCriConfig(address, root string) *cri.Config {
	config := &cri.Config{
		PluginConfig:       cri.DefaultConfig(),
		ContainerdEndpoint: address,
		ContainerdRootDir:  root,
	}
	config.DefaultRuntimeName = "runc"
	config.Runtimes = map[string]cri.Runtime{
		config.DefaultRuntimeName: {Type: plugin.RuntimeRuncV2},
	}
	config.CniConfig.NetworkPluginBinDir = filepath.Join(root, "bin")
	config.CniConfig.NetworkPluginConfDir = filepath.Join(root, "etc", "cni", "net.d")
	config.CniConfig.NetworkPluginMaxConfNum = 1
	config.SandboxImage = DefaultCriSandboxImage
	return config
}
