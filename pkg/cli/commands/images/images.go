package images

import (
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/client/action"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "images [OPTIONS] [REPOSITORY[:TAG]]",
		Short:                 "List images",
		DisableFlagsInUseLine: true,
	})
}

type CommandSpec struct {
	action.ListImages
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}

	return s.ListImages.Invoke(cmd.Context(), k8s, args)
}
