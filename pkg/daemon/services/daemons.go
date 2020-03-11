package services

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/services/opt"
	"github.com/containerd/containerd/services/server/config"
	criconfig "github.com/containerd/cri/pkg/config"
	"github.com/containerd/cri/pkg/constants"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/daemon/services/embed/buildkitd"
	embedcontainerd "github.com/rancher/k3c/pkg/daemon/services/embed/containerd"
	"github.com/sirupsen/logrus"
)

type Opts struct {
	ExtraConfig    string
	BootstrapImage string
	BridgeName     string
	BridgeCIDR     string
	Group          string
}

func StartContainerd(ctx context.Context, stateDir, rootDir string, opts *Opts) (containerdAddress string, err error) {
	if opts == nil {
		opts = &Opts{}
	}

	return startContainerd(stateDir, rootDir, opts)
}

func StartBuildkitd(ctx context.Context, containerdAddress string, serverCB buildkitd.ServerCallback, stateDir, rootDir string, opts *Opts) error {
	if opts == nil {
		opts = &Opts{}
	}

	client, err := newClient(ctx, containerdAddress)
	if err != nil {
		return err
	}

	if err := bootstrapData(ctx, client, opts.BootstrapImage); err != nil {
		return err
	}

	return startBuildkitd(containerdAddress, serverCB, stateDir, rootDir, opts)
}

func newClient(ctx context.Context, address string) (*containerd.Client, error) {
	var (
		c   *containerd.Client
		err error
	)

	for {
		c, err = containerd.New(address, containerd.WithDefaultNamespace(constants.K8sContainerdNamespace))
		if err == nil {
			break
		}

		select {
		case <-time.After(250 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	for {
		_, err := c.Version(ctx)
		if err == nil {
			return c, nil
		}
		select {
		case <-time.After(250 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func startContainerd(stateDir, rootDir string, opts *Opts) (string, error) {
	address, config, err := setupConfig(stateDir, rootDir, opts)
	if err != nil {
		return "", err
	}

	go func() {
		embedcontainerd.Run("containerd", "-c", config)
	}()

	return address, nil
}

func bootstrapData(ctx context.Context, c *containerd.Client, imageName string) error {
	if imageName == "" {
		return nil
	}

	logrus.Infof("Bootstrapping data...")
	image, err := c.GetImage(ctx, imageName)
	if errdefs.IsNotFound(err) {
		logrus.Infof("Pulling %s...", imageName)
		image, err = c.Pull(ctx, imageName, containerd.WithPullUnpack)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if err := c.Install(ctx, image, containerd.WithInstallLibs, containerd.WithInstallReplace); err != nil {
		return err
	}

	logrus.Infof("Bootstrapping done")
	return nil
}

func setupConfig(stateDir, rootDir string, opts *Opts) (string, string, error) {
	cfg := &config.Config{
		Version: 1,
		Root:    filepath.Join(rootDir, "containerd"),
		State:   filepath.Join(stateDir, "containerd"),
		GRPC: config.GRPCConfig{
			Address:        filepath.Join(stateDir, "containerd", "containerd.sock"),
			MaxRecvMsgSize: defaults.DefaultMaxRecvMsgSize,
			MaxSendMsgSize: defaults.DefaultMaxSendMsgSize,
		},
		DisabledPlugins: []string{},
		RequiredPlugins: []string{},
	}

	if _, err := os.Stat(opts.ExtraConfig); err == nil {
		cfg.Imports = []string{opts.ExtraConfig}
	}

	cni := cniDir(rootDir)
	plugins := pluginConfig(rootDir, cni)
	containerdConfig := filepath.Join(rootDir, "etc", "containerd", "config.toml")

	if err := writeTOMLToFile(containerdConfig, cfg, plugins); err != nil {
		return "", "", errors.Wrap(err, "write containerd conf")
	}

	if err := writeJSONToFile(filepath.Join(cni, "10-k3c.conflist"), cniPlugins(opts)); err != nil {
		return "", "", errors.Wrap(err, "write cni conf")
	}

	return cfg.GRPC.Address, containerdConfig, nil
}

func cniDir(rootDir string) string {
	return filepath.Join(rootDir, "etc", "cni", "net.d")
}

func pluginConfig(root, cni string) map[string]interface{} {
	bin := filepath.Join(root, "bin")

	cri := criconfig.DefaultConfig()
	r := cri.ContainerdConfig.Runtimes["runc"]
	r.Type = "io.containerd.runc.v2"
	cri.ContainerdConfig.Runtimes["runc"] = r
	cri.CniConfig.NetworkPluginBinDir = bin
	cri.CniConfig.NetworkPluginConfDir = cni

	return map[string]interface{}{
		"cri": cri,
		"opt": &opt.Config{
			Path: root,
		},
	}
}

func writeTOMLToFile(target string, data, plugins interface{}) error {
	buf := &bytes.Buffer{}
	if err := toml.NewEncoder(buf).Encode(data); err != nil {
		return err
	}

	mapData := map[string]interface{}{}
	if err := toml.Unmarshal(buf.Bytes(), &mapData); err != nil {
		return err
	}
	mapData["plugins"] = plugins

	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return errors.Wrapf(err, "mkdir %s", filepath.Dir(target))
	}

	of, err := os.Create(target)
	if err != nil {
		return err
	}
	defer of.Close()

	return toml.NewEncoder(of).Encode(mapData)
}

func writeJSONToFile(target string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return errors.Wrapf(err, "mkdir %s", filepath.Dir(target))
	}

	of, err := os.Create(target)
	if err != nil {
		return err
	}
	defer of.Close()

	return json.NewEncoder(of).Encode(data)
}

func cniPlugin(opts *Opts) map[string]interface{} {
	type m map[string]interface{}

	cidr := opts.BridgeCIDR
	if cidr == "" {
		cidr = "172.18.0.0/16"
	}
	bridge := opts.BridgeName
	if bridge == "" {
		bridge = "k3c0"
	}
	return map[string]interface{}{
		"cniVersion":  "0.3.1",
		"type":        "bridge",
		"name":        "k3c-net",
		"bridge":      bridge,
		"isGateway":   true,
		"ipMasq":      true,
		"promiscMode": true,
		"ipam": m{
			"type":   "host-local",
			"subnet": cidr,
			"routes": []m{
				{
					"dst": "0.0.0.0/0",
				},
			},
		},
	}
}

func cniPlugins(opts *Opts) map[string]interface{} {
	type m map[string]interface{}

	return m{
		"cniVersion": "0.3.1",
		"name":       "k3c-net",
		"plugins": []m{
			cniPlugin(opts),
			{
				"type": "portmap",
				"capabilities": m{
					"portMappings": true,
				},
			},
		},
	}
}
