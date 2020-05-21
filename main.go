package main

import (
	"os"
	"path/filepath"

	"github.com/containerd/containerd/pkg/seed"
	"github.com/rancher/k3c/cmd/daemon"
	"github.com/rancher/k3c/pkg/cli/app"
	"github.com/sirupsen/logrus"
)

func main() {
	seed.WithTimeAndRand()
	self := filepath.Base(os.Args[0])
	switch self {
	case "containerd":
		daemon.Containerd()
	default:
		app := app.New()
		if err := app.Run(os.Args); err != nil {
			logrus.Fatal(err)
		}
	}
}
