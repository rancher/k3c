package build

import (
	"errors"
	"os"

	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/client/action"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "build [OPTIONS] PATH",
		Short:                 "Build an image",
		DisableFlagsInUseLine: true,
	})
}

type CommandSpec struct {
	action.BuildImage
}

func (c *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("exactly one argument is required")
	}

	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	path := args[0]
	if path == "" || path == "." {
		path, err = os.Getwd()
	}
	if err != nil {
		return err
	}
	return c.BuildImage.Invoke(cmd.Context(), k8s, path)
}
