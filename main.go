package main

import (
	"os"

	"github.com/containerd/containerd/pkg/seed"
	"github.com/rancher/k3c/pkg/cli/app"
	"github.com/sirupsen/logrus"
)

func main() {
	seed.WithTimeAndRand()
	app := app.New()
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
