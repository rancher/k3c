package cli

import (
	"github.com/rancher/k3c/pkg/cli/commands/agent"
	"github.com/rancher/k3c/pkg/cli/commands/build"
	"github.com/rancher/k3c/pkg/cli/commands/images"
	"github.com/rancher/k3c/pkg/cli/commands/info"
	"github.com/rancher/k3c/pkg/cli/commands/install"
	"github.com/rancher/k3c/pkg/cli/commands/pull"
	"github.com/rancher/k3c/pkg/cli/commands/push"
	"github.com/rancher/k3c/pkg/cli/commands/rmi"
	"github.com/rancher/k3c/pkg/cli/commands/tag"
	"github.com/rancher/k3c/pkg/cli/commands/uninstall"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/version"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

var (
	DebugConfig wrangler.DebugConfig
)

func Main() *cobra.Command {
	root := wrangler.Command(&App{}, cobra.Command{
		Use:     "k3c",
		Version: version.FriendlyVersion(),
	})
	root.AddCommand(
		agent.Command(),
		info.Command(),
		images.Command(),
		install.Command(),
		uninstall.Command(),
		build.Command(),
		pull.Command(),
		push.Command(),
		rmi.Command(),
		tag.Command(),
	)
	return root
}

type App struct {
	wrangler.DebugConfig
	client.Config
}

func (s *App) Run(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}

func (s *App) PersistentPre(_ *cobra.Command, _ []string) error {
	s.MustSetupDebug()
	DebugConfig = s.DebugConfig
	client.DefaultConfig = s.Config
	return nil
}
