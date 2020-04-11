package config

import (
	"path/filepath"

	"github.com/containerd/containerd/plugin"
	cri "github.com/containerd/cri/pkg/config"
)

var (
	DefaultCriSandboxImage = "docker.io/rancher/pause:3.1"
)

func DefaultCriConfig(address, root string) *cri.PluginConfig {
	// PluginConfig
	config := cri.DefaultConfig()
	config.SandboxImage = DefaultCriSandboxImage
	// .ContainerdConfig
	config.DefaultRuntimeName = "runc"
	config.Runtimes = map[string]cri.Runtime{
		config.DefaultRuntimeName: {Type: plugin.RuntimeRuncV2},
	}
	// .CniConfig
	config.NetworkPluginBinDir = filepath.Join(root, "bin")
	config.NetworkPluginConfDir = filepath.Join(root, "etc", "cni", "net.d")
	config.NetworkPluginMaxConfNum = 1
	return &config
}
