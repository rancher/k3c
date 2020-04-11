package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/cmd/containerd/command"
	"github.com/containerd/containerd/pkg/timeout"
	"github.com/containerd/containerd/plugin"
	containerd "github.com/containerd/containerd/services/server/config"
	cri "github.com/containerd/cri/pkg/config"
	buildkit "github.com/moby/buildkit/cmd/buildkitd/config"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/daemon"
	"github.com/rancher/k3c/pkg/daemon/config"
	bkplugin "github.com/rancher/k3c/pkg/daemon/services/buildkit"
	"github.com/sirupsen/logrus"
	cliv1 "github.com/urfave/cli"
	cliv2 "github.com/urfave/cli/v2"
	"k8s.io/klog"
)

func Command(version string) *cliv2.Command {
	app := command.App()
	app.Name = "k3c daemon"
	app.Usage = "containerd++ (cri, buildkit and k3c)"
	app.Description = `
k3c daemon is containerd work-alike presenting the CRI, BuildKit, K3C and
containerd APIs all on a single gRPC socket. It is meant to be a drop-in
replacement for a CRI-enabled containerd with additional functionality on
the backend to support the Docker work-alike frontend of k3c.`
	app.HelpName = app.Name
	app.Version = fmt.Sprintf("k3c version %s (%s)", version, app.Version)

	for i := range app.Flags {
		flag := app.Flags[i]
		logrus.Debugf("%+v", flag)
		switch n := flag.GetName(); n {
		case "address,a":
			if sf, ok := flag.(cliv1.StringFlag); ok {
				sf.Value = filepath.Join(config.DefaultDaemonStateDir, "k3c.sock")
				app.Flags[i] = sf
			} else {
				logrus.Warnf("unexpected type for flag %q = %#v", flag.GetName(), flag)
			}
		case "config,c":
			if sf, ok := flag.(cliv1.StringFlag); ok {
				sf.Value = config.DefaultDaemonConfigFile
				app.Flags[i] = sf
			} else {
				logrus.Warnf("unexpected type for flag %q = %#v", flag.GetName(), flag)
			}
		case "root":
			if sf, ok := flag.(cliv1.StringFlag); ok {
				sf.Value = config.DefaultDaemonRootDir
				app.Flags[i] = sf
			} else {
				logrus.Warnf("unexpected type for flag %q = %#v", flag.GetName(), flag)
			}
		case "state":
			if sf, ok := flag.(cliv1.StringFlag); ok {
				sf.Value = config.DefaultDaemonStateDir
				app.Flags[i] = sf
			} else {
				logrus.Warnf("unexpected type for flag %q = %#v", flag.GetName(), flag)
			}
		}
	}

	app.Before = func(before cliv1.BeforeFunc) cliv1.BeforeFunc {
		return func(clx *cliv1.Context) error {
			var (
				root    = clx.GlobalString("root")
				state   = clx.GlobalString("state")
				address = clx.GlobalString("address")
				file    = clx.GlobalString("config")
				conf    = config.DefaultContainerdConfig(root, state, address)
			)
			// sidestep the merge during containerd.LoadConfig
			required := conf.RequiredPlugins
			conf.RequiredPlugins = nil
			if err := containerd.LoadConfig(file, conf); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
				conf.RequiredPlugins = required
			}
			if conf.Version == 1 {
				return fmt.Errorf("unsupported configuration version: %d", conf.Version)
			}
			if err := os.Setenv("K3C_ADDRESS", conf.GRPC.Address); err != nil {
				logrus.Warn(err)
			}
			if err := os.Setenv("K3C_ROOT", conf.Root); err != nil {
				logrus.Warn(err)
			}
			if err := os.Setenv("K3C_STATE", conf.State); err != nil {
				logrus.Warn(err)
			}
			k3cfg := config.DefaultK3Config()
			err := writeConfig(conf, file, config.DefaultCniConf(k3cfg), config.DefaultCniConflist(k3cfg),
				&plugin.Registration{
					ID:     daemon.PluginRegistration.ID,
					Type:   plugin.GRPCPlugin,
					Config: k3cfg,
				},
				&plugin.Registration{
					ID:     "cri",
					Type:   plugin.GRPCPlugin,
					Config: config.DefaultCriConfig(address, root),
				},
				&plugin.Registration{
					ID:     bkplugin.PluginRegistration.ID,
					Type:   plugin.GRPCPlugin,
					Config: config.DefaultBuildkitConfig(address, root),
				},
				&plugin.Registration{
					ID:     "opt",
					Type:   plugin.InternalPlugin,
					Config: config.DefaultOptConfig(root),
				},
			)
			if err != nil {
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

func writeConfig(cfg *containerd.Config, path string, cniConf, cniConfList map[string]interface{}, plugins ...*plugin.Registration) error {
	config := &command.Config{
		Config:  cfg,
		Plugins: make(map[string]interface{}, len(plugins)),
	}
	if len(plugins) != 0 {
		config.Plugins = make(map[string]interface{})
		for _, p := range plugins {
			if p.Config == nil {
				continue
			}
			pc, err := config.Decode(p)
			if err != nil {
				return err
			}
			config.Plugins[p.URI()] = pc
			switch p.ID {
			case "k3c":
				// TODO(dweomer): nothing to do here ... yet
			case "cri":
				xfg, ok := p.Config.(*cri.Config)
				if !ok {
					return fmt.Errorf("unexpected config type for plugin %q: %T", p.ID, p.Config)
				}
				if err := os.MkdirAll(xfg.CniConfig.NetworkPluginConfDir, 0700); err != nil {
					return errors.Wrapf(err, "mkdir %s", xfg.CniConfig.NetworkPluginConfDir)
				}
				if err := jsonEncodeThenClose(filepath.Join(xfg.CniConfig.NetworkPluginConfDir, "90-k3c.conflist"), cniConfList); err != nil {
					return err
				}
			case "buildkit":
				xfg, ok := p.Config.(*buildkit.Config)
				if !ok {
					return fmt.Errorf("unexpected config type for plugin %q: %T", p.ID, p.Config)
				}
				if err := os.MkdirAll(filepath.Dir(xfg.Workers.Containerd.CNIConfigPath), 0700); err != nil {
					return errors.Wrapf(err, "mkdir %s", filepath.Dir(xfg.Workers.Containerd.CNIConfigPath))
				}
				if err := jsonEncodeThenClose(xfg.Workers.Containerd.CNIConfigPath, cniConf); err != nil {
					return err
				}
			}
		}
	}

	timeouts := timeout.All()
	config.Timeouts = make(map[string]string)
	for k, v := range timeouts {
		config.Timeouts[k] = v.String()
	}

	// for the time being, keep the defaultConfig's version set at 1 so that
	// when a config without a version is loaded from disk and has no version
	// set, we assume it's a v1 config.  But when generating new configs via
	// this command, generate the v2 config
	config.Config.Version = 2

	// remove overridden Plugins type to avoid duplication in output
	config.Config.Plugins = nil

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return errors.Wrapf(err, "mkdir %s", filepath.Dir(path))
	}
	w, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = config.WriteTo(w)
	return err
}

func jsonEncodeThenClose(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(data)

}
