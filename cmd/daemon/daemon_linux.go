package daemon

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rancher/k3c/pkg/daemon/services/buildkit"

	"github.com/containerd/containerd/cmd/containerd/command"
	"github.com/containerd/containerd/plugin"
	"github.com/rancher/k3c/pkg/daemon"
	"github.com/rancher/k3c/pkg/daemon/config"
	k3c "github.com/rancher/k3c/pkg/defaults"
	"github.com/sirupsen/logrus"
	cliv1 "github.com/urfave/cli"
	cliv2 "github.com/urfave/cli/v2"
	"k8s.io/klog"
)

func Command() *cliv2.Command {
	app := command.App()
	app.Name = "k3c daemon"
	app.Usage = "containerd++ (cri, buildkit and k3c)"
	app.Description = `
k3c daemon is containerd work-alike presenting the CRI, BuildKit, K3C and
containerd APIs all on a single gRPC socket. It is meant to be a drop-in
replacement for a CRI-enabled containerd with additional functionality on
the backend to support the Docker work-alike frontend of k3c.`
	app.HelpName = app.Name

	for i := range app.Flags {
		flag := app.Flags[i]
		logrus.Debugf("%+v", flag)
		switch n := flag.GetName(); n {
		case "address,a":
			if sf, ok := flag.(cliv1.StringFlag); ok {
				sf.Value = filepath.Join(config.DefaultDaemonStateDir, "k3c.sock")
				sf.EnvVar = "K3C_ADDRESS"
				app.Flags[i] = sf
			} else {
				logrus.Warnf("unexpected type for flag %q = %#v", flag.GetName(), flag)
			}
		case "config,c":
			if sf, ok := flag.(cliv1.StringFlag); ok {
				sf.Value = config.DefaultDaemonConfigFile
				sf.EnvVar = "K3C_CONFIG"
				app.Flags[i] = sf
			} else {
				logrus.Warnf("unexpected type for flag %q = %#v", flag.GetName(), flag)
			}
		case "root":
			if sf, ok := flag.(cliv1.StringFlag); ok {
				sf.Value = config.DefaultDaemonRootDir
				sf.EnvVar = "K3C_ROOT"
				app.Flags[i] = sf
			} else {
				logrus.Warnf("unexpected type for flag %q = %#v", flag.GetName(), flag)
			}
		case "state":
			if sf, ok := flag.(cliv1.StringFlag); ok {
				sf.Value = config.DefaultDaemonStateDir
				sf.EnvVar = "K3C_STATE"
				app.Flags[i] = sf
			} else {
				logrus.Warnf("unexpected type for flag %q = %#v", flag.GetName(), flag)
			}
		}
	}
	app.Flags = append(app.Flags, []cliv1.Flag{
		cliv1.StringFlag{
			Name:        "bridge-name",
			Value:       config.DefaultBridgeName,
			EnvVar:      "K3C_BRIDGE_NAME",
			Destination: &daemon.Config.BridgeName,
		},
		cliv1.StringFlag{
			Name:        "bridge-cidr",
			Value:       config.DefaultBridgeCIDR,
			EnvVar:      "K3C_BRIDGE_CIDR",
			Destination: &daemon.Config.BridgeCIDR,
		},
		cliv1.BoolFlag{
			Name:        "bootstrap-skip",
			EnvVar:      "K3C_BOOTSTRAP_SKIP",
			Usage:       "skip bootstrap install (default: false)",
			Destination: &daemon.Config.BootstrapSkip,
		},
		cliv1.StringFlag{
			Name:        "bootstrap-image",
			Value:       config.DefaultBootstrapImage,
			EnvVar:      "K3C_BOOTSTRAP_IMAGE",
			Usage:       "containerd-style image ref to install",
			Destination: &daemon.Config.BootstrapImage,
		},
		cliv1.StringFlag{
			Name:        "bootstrap-namespace",
			Value:       config.DefaultBootstrapNamespace,
			EnvVar:      "K3C_BOOTSTRAP_NAMESPACE",
			Destination: &daemon.Config.BootstrapNamespace,
			Hidden:      true,
		},
		cliv1.StringFlag{
			Name:        "sandbox-image",
			Value:       config.DefaultSandboxImage,
			EnvVar:      "K3C_SANDBOX_IMAGE",
			Usage:       "containerd-style image ref for sandboxes",
			Destination: &daemon.Config.Namespace.SandboxImage,
		},
	}...)

	app.Before = func(before cliv1.BeforeFunc) cliv1.BeforeFunc {
		return func(clx *cliv1.Context) error {
			// setup env
			for i := range clx.App.Flags {
				var (
					f = clx.App.Flags[i]
					n = f.GetName()
					e string
				)
				switch t := f.(type) {
				case cliv1.BoolFlag:
					e = t.EnvVar
				case cliv1.StringFlag:
					e = t.EnvVar
				}
				if e != "" {
					if err := os.Setenv(e, clx.GlobalString(n)); err != nil {
						return err
					}
				}
			}
			// setup cfg
			var (
				root = clx.GlobalString("root")
				path = filepath.Join(clx.GlobalString("root"), fmt.Sprintf("%s.cri", plugin.GRPCPlugin), "namespaces", k3c.DefaultNamespace, "config.toml")
			)
			daemon.Config.Namespace.NetworkPluginBinDir = filepath.Join(root, "bin")
			daemon.Config.Namespace.NetworkPluginConfDir = filepath.Join(root, "etc", "cni", "net.d")
			buildkit.Config.Workers.Containerd.NetworkConfig.Mode = "cni"
			buildkit.Config.Workers.Containerd.NetworkConfig.CNIBinaryPath = filepath.Join(root, "bin")
			buildkit.Config.Workers.Containerd.NetworkConfig.CNIConfigPath = filepath.Join(root, "etc", "cni", "net.d", "90-k3c.json")

			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return err
			}
			if err := config.WriteFileTOML(path, &daemon.Config.Namespace, 0600); err != nil {
				return err
			}
			if err := os.MkdirAll(daemon.Config.Namespace.NetworkPluginConfDir, 0700); err != nil {
				return err
			}
			if err := config.WriteFileJSON(buildkit.Config.Workers.Containerd.NetworkConfig.CNIConfigPath, config.DefaultCniConf(daemon.Config.BridgeName, daemon.Config.BridgeCIDR), 0600); err != nil {
				return err
			}
			if err := config.WriteFileJSON(filepath.Join(daemon.Config.Namespace.NetworkPluginConfDir, "90-k3c.conflist"), config.DefaultCniConflist(daemon.Config.BridgeName, daemon.Config.BridgeCIDR), 0600); err != nil {
				return err
			}

			if before != nil {
				return before(clx)
			}
			return nil
		}
	}(app.Before)

	return &cliv2.Command{
		Name:            "daemon",
		Usage:           "Run the container daemon",
		Description:     app.Description,
		SkipFlagParsing: true,

		Before: func(clx *cliv2.Context) error {
			klog.InitFlags(nil)
			return nil
		},

		Action: func(clx *cliv2.Context) error {
			args := []string{app.Name}
			args = append(args, clx.Args().Slice()...)
			return app.Run(args)
		},
	}
}
