package uninstall

import (
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/client/action"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "uninstall [OPTIONS]",
		Short:                 "Uninstall builder component(s)",
		DisableFlagsInUseLine: true,
	})
}

type CommandSpec struct {
	action.UninstallBuilder
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	err = s.UninstallBuilder.Namespace(ctx, k8s)
	if err != nil {
		return err
	}
	return s.UninstallBuilder.NodeRole(ctx, k8s)
}
