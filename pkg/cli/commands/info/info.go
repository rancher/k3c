package info

import (
	"github.com/pkg/errors"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:   "info [OPTIONS]",
		Short: "Display builder information",
	})
}

type CommandSpec struct {
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	return errors.New("not implemented") // TODO
}
