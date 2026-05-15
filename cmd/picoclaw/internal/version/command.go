package version

import (
	"github.com/spf13/cobra"

	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/cliui"
	"github.com/conglinyizhi/SylastraClaws/pkg/config"
)

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Show version information",
		Run: func(_ *cobra.Command, _ []string) {
			printVersion()
		},
	}

	return cmd
}

func printVersion() {
	build, goVer := config.FormatBuildInfo()
	cliui.PrintVersion(internal.Logo, "sylastraclaws "+config.FormatVersion(), build, goVer)
}
