// +build !linux

package daemon

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

type Daemon struct {
}

func (d *Daemon) Run(ctx *cli.Context) error {
	return fmt.Errorf("daemon only supported on Linux")
}
