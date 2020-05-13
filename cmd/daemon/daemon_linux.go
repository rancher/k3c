package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd/cmd/containerd/command"
	"github.com/containerd/containerd/plugin"
	"github.com/containerd/containerd/services/opt"
	"github.com/containerd/cri"
	criconfig "github.com/containerd/cri/pkg/config"
	"github.com/rancher/k3c/pkg/daemon"
	"github.com/rancher/k3c/pkg/daemon/config"
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
				sf.Destination = &opt.PluginConfig.Path
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
		cliv1.StringFlag{
			Name:        "bootstrap-image",
			Value:       config.DefaultBootstrapImage,
			EnvVar:      "K3C_BOOTSTRAP_IMAGE",
			Usage:       "containerd-style image ref to install",
			Destination: &daemon.Config.BootstrapImage,
		},
	}...)
	defaultBootstrapSkip := true
	requiredExecutables := map[string]string{
		"bridge":                  "", // cni
		"containerd-shim":         "", // containerd
		"containerd-shim-runc-v1": "", // buildkit
		"containerd-shim-runc-v2": "", // cri
		"host-local":              "", // cni
		"iptables":                "", // cni, buildkit
		"loopback":                "", // cni
		"portmap":                 "", // cni
		"runc":                    "", // cri, buildkit
		"socat":                   "", // cri
	}
	for bin := range requiredExecutables {
		if path, err := exec.LookPath(bin); err != nil {
			defaultBootstrapSkip = false
		} else {
			requiredExecutables[bin] = path
		}
	}
	if defaultBootstrapSkip {
		app.Flags = append(app.Flags, cliv1.BoolTFlag{
			Name:        "bootstrap-skip",
			EnvVar:      "K3C_BOOTSTRAP_SKIP",
			Usage:       "skip bootstrap install (default: true)",
			Destination: &daemon.Config.BootstrapSkip,
		})
	} else {
		app.Flags = append(app.Flags, cliv1.BoolFlag{
			Name:        "bootstrap-skip",
			EnvVar:      "K3C_BOOTSTRAP_SKIP",
			Usage:       "skip bootstrap install (default: false)",
			Destination: &daemon.Config.BootstrapSkip,
		})
	}
	app.Flags = append(app.Flags, []cliv1.Flag{
		cliv1.StringFlag{
			Name:        "cni-bin",
			EnvVar:      "K3C_CNI_BIN",
			Destination: &cri.Config.NetworkPluginBinDir,
		},
		cliv1.StringFlag{
			Name:        "cni-netconf",
			EnvVar:      "K3C_CNI_NETCONF",
			Destination: &cri.Config.NetworkPluginConfDir,
		},
		cliv1.StringFlag{
			Name:        "sandbox-image",
			Value:       config.DefaultSandboxImage,
			EnvVar:      "K3C_SANDBOX_IMAGE",
			Usage:       "containerd-style image ref for sandboxes",
			Destination: &cri.Config.SandboxImage,
		},
	}...)
	app.Before = func(before cliv1.BeforeFunc) cliv1.BeforeFunc {
		return func(clx *cliv1.Context) error {
			// setup env
			for i := range clx.App.Flags {
				var (
					f = clx.App.Flags[i]
					n = strings.SplitN(f.GetName(), `,`, 2)[0]
					e string
				)
				switch t := f.(type) {
				case cliv1.BoolFlag:
					e = t.EnvVar
				case cliv1.BoolTFlag:
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
			)
			if cniBin := clx.GlobalString("cni-bin"); cniBin == "" {
				cri.Config.NetworkPluginBinDir = filepath.Join(root, "bin")
			} else {
				cri.Config.NetworkPluginBinDir = cniBin
			}
			if cniNetConf := clx.GlobalString("cni-netconf"); cniNetConf == "" {
				cri.Config.NetworkPluginConfDir = filepath.Join(root, "etc", "cni", "net.d")
			} else {
				cri.Config.NetworkPluginConfDir = cniNetConf
			}
			cri.Config.DefaultRuntimeName = "runc"
			cri.Config.Runtimes = map[string]criconfig.Runtime{
				cri.Config.DefaultRuntimeName: {
					Type: plugin.RuntimeRuncV2,
				},
			}
			if err := os.MkdirAll(cri.Config.NetworkPluginBinDir, 0700); err != nil {
				return err
			}
			for exe, loc := range requiredExecutables {
				if defaultBootstrapSkip {
					// the symlinking is to make buildkit happy
					if err := os.Symlink(loc, filepath.Join(cri.Config.NetworkPluginBinDir, exe)); err != nil {
						logrus.WithError(err).Warn("k3s bootstrap skip")
					}
				} else if loc == "" {
					logrus.WithField("executable", exe).Warn("k3c bootstrap check: missing")
				} else {
					logrus.WithField("executable", exe).WithField("location", loc).Debug("k3c bootstrap check: found")
				}
			}
			if err := os.MkdirAll(cri.Config.NetworkPluginConfDir, 0700); err != nil {
				return err
			}
			if err := config.WriteFileJSON(filepath.Join(cri.Config.NetworkPluginConfDir, "90-k3c.conflist"), config.DefaultCniConflist(daemon.Config.BridgeName, daemon.Config.BridgeCIDR), 0600); err != nil {
				return err
			}
			if err := os.Setenv("PATH", fmt.Sprintf("%s%c%s", os.Getenv("PATH"), os.PathListSeparator, filepath.Join(root, "bin", "aux"))); err != nil {
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
