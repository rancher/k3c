// +build !linux

package daemon

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func Command(version string) *cli.Command {
	return &cli.Command{
		Name:            "daemon",
		Usage:           "Run the container daemon",
		SkipFlagParsing: true,
		Hidden:          true,
		Action: func(clx *cli.Context) error {
			return fmt.Errorf("%s only supported on Linux", clx.Command.Name)
		},
	}
}
