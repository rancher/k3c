package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rancher/k3c/cmd/attach"
	"github.com/rancher/k3c/cmd/build"
	"github.com/rancher/k3c/cmd/create"
	"github.com/rancher/k3c/cmd/daemon"
	"github.com/rancher/k3c/cmd/events"
	"github.com/rancher/k3c/cmd/exec"
	"github.com/rancher/k3c/cmd/images"
	"github.com/rancher/k3c/cmd/inspect"
	"github.com/rancher/k3c/cmd/logs"
	"github.com/rancher/k3c/cmd/ps"
	"github.com/rancher/k3c/cmd/pull"
	"github.com/rancher/k3c/cmd/push"
	"github.com/rancher/k3c/cmd/rm"
	"github.com/rancher/k3c/cmd/rmi"
	"github.com/rancher/k3c/cmd/run"
	"github.com/rancher/k3c/cmd/start"
	"github.com/rancher/k3c/cmd/stop"
	"github.com/rancher/k3c/cmd/tag"
	"github.com/rancher/k3c/cmd/volume"
	"github.com/rancher/k3c/pkg/clibuilder"
	"github.com/rancher/k3c/pkg/version"
	"github.com/rancher/norman/v2/pkg/debug"
	"github.com/urfave/cli/v2"
)

var (
	Name  = filepath.Base(os.Args[0])
	Debug debug.Config
)

func New() *cli.App {
	app := cli.NewApp()
	app.Name = Name
	app.Usage = "Lightweight local container platform"
	app.Version = fmt.Sprintf("%s (%s)", version.Version, version.GitCommit)

	app.Flags = []cli.Flag{}
	app.Flags = append(app.Flags, debug.FlagsV2(&Debug)...)

	app.Before = func(*cli.Context) error {
		Debug.MustSetupDebug()
		return nil
	}

	app.Commands = []*cli.Command{
		command(&create.Create{},
			"Create a new container",
			"IMAGE [COMMAND] [ARG...]"),
		command(&attach.Attach{},
			"Attach local standard input, output, and error streams to a running container",
			"CONTAINER"),
		command(&stop.Stop{},
			"Stop one or more running containers",
			"CONTAINER [CONTAINER...]"),
		command(&start.Start{},
			"Start one or more stopped containers",
			"CONTAINER [CONTAINER...]"),
		command(&logs.Logs{},
			"Fetch the logs of a container",
			"CONTAINER"),
		command(&rm.Rm{},
			"Remove one or more containers",
			"CONTAINER [CONTAINER...]"),
		command(&exec.Exec{},
			"Run a command in a running container",
			"CONTAINER COMMAND [ARG...]"),
		command(&run.Run{},
			"Run a command in a new container",
			"IMAGE [COMMAND] [ARG...]"),
		command(&ps.Ps{},
			"List containers",
			""),
		command(&build.Build{},
			"Build an image from a Dockerfile",
			"PATH | URL"),
		command(&images.Images{},
			"List images",
			"[REPOSITORY[:TAG]]"),
		clibuilder.Command(&tag.Tag{}, cli.Command{
			Usage:       "Create a tag TARGET_IMAGE that refers to SOURCE_IMAGE",
			Description: Name + " tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]",
		}),
		command(&pull.Pull{},
			"Pull an image or a repository from a registry",
			"NAME[:TAG|@DIGEST]"),
		command(&rmi.Rmi{},
			"Remove one or more images",
			"IMAGE [IMAGE...]"),
		command(&push.Push{},
			"Push an image or a repository to a registry",
			"NAME[:TAG]"),
		command(&inspect.Inspect{},
			"Return low-level information on k3c objects",
			"NAME|ID [NAME|ID...]"),
		command(&events.Events{},
			"Get real time events from the server",
			""),
		daemon.Command(),
		{
			Name:    "volume",
			Aliases: []string{"volumes", "v"},
			Usage:   "Manage volumes",
			Subcommands: []*cli.Command{
				command(&volume.Ls{},
					"List volumes",
					""),
				command(&volume.Rm{},
					"Remove one or more volumes",
					"VOLUME [VOLUME...]"),
				command(&volume.Create{},
					"Create a volume",
					"[VOLUME]"),
			},
		},
	}
	return app
}

func command(obj clibuilder.Runnable, usage, desc string) *cli.Command {
	return clibuilder.Command(obj, cli.Command{
		Usage:       usage,
		Description: fmt.Sprintf("%s %s [OPTIONS] %s", Name, clibuilder.CommandName(obj), desc),
	})
}
