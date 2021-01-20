package agent

import (
	"github.com/rancher/k3c/pkg/server/action"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "agent [OPTIONS]",
		Short:                 "Run the controller daemon",
		Hidden:                true,
		DisableFlagsInUseLine: true,
	})
}

type CommandSpec struct {
	action.Agent
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	return s.Agent.Run(cmd.Context())
}
