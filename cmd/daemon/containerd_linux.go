package daemon

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
	"k8s.io/klog"
)

func Containerd() {
	app := newApp()
	app.Name = "containerd"
	app.HelpName = app.Name
	app.Before = func(clx *cli.Context) error {
		klog.InitFlags(nil)
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", app.Name, err)
		os.Exit(1)
	}
}
