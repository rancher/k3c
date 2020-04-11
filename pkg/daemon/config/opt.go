package config

import "github.com/containerd/containerd/services/opt"

func DefaultOptConfig(root string) *opt.Config {
	return &opt.Config{
		Path: root,
	}
}
