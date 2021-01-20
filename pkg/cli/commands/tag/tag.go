package tag

import (
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/client/action"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:   "tag SOURCE_REF TARGET_REF [TARGET_REF, ...]",
		Short: "Tag an image",
	})
}

type CommandSpec struct {
	action.TagImage
}

func (c *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return errors.New("at least two arguments are required")
	}

	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	return c.TagImage.Invoke(cmd.Context(), k8s, args[0], args[1:])
}
