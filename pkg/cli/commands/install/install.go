package install

import (
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/client/action"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "install [OPTIONS]",
		Short:                 "Install builder component(s)",
		DisableFlagsInUseLine: true,
	})
}

type CommandSpec struct {
	action.InstallBuilder
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	// assert namespace
	err = s.InstallBuilder.Namespace(ctx, k8s)
	if err != nil {
		return err
	}
	// assert service
	err = s.InstallBuilder.Service(ctx, k8s)
	if err != nil {
		return err
	}
	// assert daemonset
	err = s.InstallBuilder.DaemonSet(ctx, k8s)
	if err != nil {
		return err
	}
	// label the node(s)
	return s.InstallBuilder.NodeRole(ctx, k8s)
}
