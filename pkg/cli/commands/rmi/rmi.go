package rmi

import (
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/client/action"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:   "rmi REF",
		Short: "Remove an image",
	})
}

type CommandSpec struct {
	action.RemoveImage
}

func (c *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("exactly one argument is required")
	}

	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	return c.RemoveImage.Invoke(cmd.Context(), k8s, args[0])
}
