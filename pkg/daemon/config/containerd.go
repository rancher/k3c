package config

import (
	"fmt"

	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/plugin"
	containerd "github.com/containerd/containerd/services/server/config"
)

var (
	DefaultContainerdMaxRecvMsgSize = defaults.DefaultMaxRecvMsgSize
	DefaultContainerdMaxSendMsgSize = defaults.DefaultMaxSendMsgSize
)

func DefaultContainerdConfig(root, state, address string) *containerd.Config {
	config := &containerd.Config{
		Version: 2,
		Root:    root,
		State:   state,
		GRPC: containerd.GRPCConfig{
			Address:        address,
			MaxRecvMsgSize: DefaultContainerdMaxRecvMsgSize,
			MaxSendMsgSize: DefaultContainerdMaxSendMsgSize,
		},
		DisabledPlugins: []string{},
		RequiredPlugins: []string{
			fmt.Sprintf("%s.%s", plugin.GRPCPlugin, "cri"),
			fmt.Sprintf("%s.%s", plugin.GRPCPlugin, "k3c"),
		},
	}
	return config
}
