package config

import (
	"path/filepath"

	"github.com/containerd/cri/pkg/constants"
	buildkit "github.com/moby/buildkit/cmd/buildkitd/config"
	"github.com/moby/buildkit/util/binfmt_misc"
)

func DefaultBuildkitConfig(address, root string) *buildkit.Config {
	config := &buildkit.Config{}
	enabled := true
	config.Workers.Containerd = buildkit.ContainerdConfig{
		Enabled:   &enabled,
		Address:   address,
		Platforms: binfmt_misc.SupportedPlatforms(),
		Namespace: constants.K8sContainerdNamespace,
		NetworkConfig: buildkit.NetworkConfig{
			Mode:          "cni",
			CNIBinaryPath: filepath.Clean(filepath.Join(root, "bin")),
			CNIConfigPath: filepath.Clean(filepath.Join(root, "etc", "cni", "net.d", "90-k3c.json")),
		},
	}
	return config
}
