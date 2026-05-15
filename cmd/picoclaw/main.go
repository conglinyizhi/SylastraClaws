// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/agent"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/auth"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/cliui"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/cron"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/firstrun"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/gateway"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/mcp"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/migrate"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/model"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/onboard"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/skills"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/status"
	"github.com/conglinyizhi/SylastraClaws/cmd/picoclaw/internal/version"
	"github.com/conglinyizhi/SylastraClaws/pkg/config"
	"github.com/conglinyizhi/SylastraClaws/pkg/updater"
)

var rootNoColor bool

func syncCliUIColor(root *cobra.Command) {
	no, _ := root.PersistentFlags().GetBool("no-color")
	cliui.Init(no || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb")
}

// earlyColorDisabled matches lipgloss/banner behavior from env and argv before Cobra parses flags.
func earlyColorDisabled() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return true
	}
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--no-color" || arg == "--no-color=true" || arg == "--no-color=1" {
			return true
		}
	}
	return false
}

// extractFirstRunValue returns the --first-run flag value from os.Args.
// Supports both:
//
//	--first-run <value>    (next arg)
//	--first-run=<value>    (equals form)
func extractFirstRunValue() string {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--first-run" || arg == "--first-run=true" {
			if i+1 < len(os.Args) && arg == "--first-run" {
				if !strings.HasPrefix(os.Args[i+1], "--") {
					return os.Args[i+1]
				}
			}
			// --first-run without value or --first-run=true both mean empty
			return ""
		}
		if strings.HasPrefix(arg, "--first-run=") {
			val := strings.TrimPrefix(arg, "--first-run=")
			if val == "true" || val == "" {
				return ""
			}
			return val
		}
	}
	return ""
}

func NewPicoclawCommand() *cobra.Command {
	short := fmt.Sprintf("%s PicoClaw — personal AI assistant", internal.Logo)
	long := fmt.Sprintf(`%s PicoClaw is a lightweight personal AI assistant.

Version: %s`, internal.Logo, config.FormatVersion())

	cmd := &cobra.Command{
		Use:   "picoclaw",
		Short: short,
		Long:  long,
		Example: `picoclaw version
picoclaw onboard
picoclaw --no-color status`,
		SilenceErrors: true,
		// Avoid plain UsageString() on stderr/stdout when a command fails; cliui
		// renders matching panels on stderr instead.
		SilenceUsage: true,
		PersistentPreRun: func(c *cobra.Command, _ []string) {
			syncCliUIColor(c.Root())
		},
	}

	cmd.PersistentFlags().BoolVar(&rootNoColor, "no-color", false,
		"Disable colors (boxed layout unchanged)")

	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		syncCliUIColor(c.Root())
		fmt.Fprint(c.OutOrStdout(), cliui.RenderCommandHelp(c))
	})

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		mcp.NewMCPCommand(),
		migrate.NewMigrateCommand(),
		skills.NewSkillsCommand(),
		model.NewModelCommand(),
		updater.NewUpdateCommand("picoclaw"),
		version.NewVersionCommand(),
	)

	return cmd
}

const (
	colorBlue = "\033[1;38;2;62;93;185m"
	colorRed  = "\033[1;38;2;213;70;70m"
	banner    = "\r\n" +
		colorBlue + "██████╗ ██╗ ██████╗ ██████╗ " + colorRed + " ██████╗██╗      █████╗ ██╗    ██╗\n" +
		colorBlue + "██╔══██╗██║██╔════╝██╔═══██╗" + colorRed + "██╔════╝██║     ██╔══██╗██║    ██║\n" +
		colorBlue + "██████╔╝██║██║     ██║   ██║" + colorRed + "██║     ██║     ███████║██║ █╗ ██║\n" +
		colorBlue + "██╔═══╝ ██║██║     ██║   ██║" + colorRed + "██║     ██║     ██╔══██║██║███╗██║\n" +
		colorBlue + "██║     ██║╚██████╗╚██████╔╝" + colorRed + "╚██████╗███████╗██║  ██║╚███╔███╔╝\n" +
		colorBlue + "╚═╝     ╚═╝ ╚═════╝ ╚═════╝ " + colorRed + " ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝\n " +
		"\033[0m\r\n"
	plainBanner = "\r\n" +
		"██████╗ ██╗ ██████╗ ██████╗  ██████╗██╗      █████╗ ██╗    ██╗\n" +
		"██╔══██╗██║██╔════╝██╔═══██╗██╔════╝██║     ██╔══██╗██║    ██║\n" +
		"██████╔╝██║██║     ██║   ██║██║     ██║     ███████║██║ █╗ ██║\n" +
		"██╔═══╝ ██║██║     ██║   ██║██║     ██║     ██╔══██║██║███╗██║\n" +
		"██║     ██║╚██████╗╚██████╔╝╚██████╗███████╗██║  ██║╚███╔███╔╝\n" +
		"╚═╝     ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝\n " +
		"\r\n"
)

func main() {
	cliui.Init(earlyColorDisabled())

	if earlyColorDisabled() {
		fmt.Print(plainBanner)
	} else {
		fmt.Printf("%s", banner)
	}

	// Handle --first-run before normal command execution
	if val := extractFirstRunValue(); val != "" {
		if err := firstrun.Run(val); err != nil && err != firstrun.ErrNotFirstRun {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	tzEnv := os.Getenv("TZ")
	if tzEnv != "" {
		fmt.Println("TZ environment:", tzEnv)
		zoneinfoEnv := os.Getenv("ZONEINFO")
		fmt.Println("ZONEINFO environment:", zoneinfoEnv)
		loc, err := time.LoadLocation(tzEnv)
		if err != nil {
			fmt.Println("Error loading time zone:", err)
		} else {
			fmt.Println("Time zone loaded successfully:", loc)
			time.Local = loc //nolint:gosmopolitan // We intentionally set local timezone from TZ env
		}
	}

	cmd := NewPicoclawCommand()
	last, err := cmd.ExecuteC()
	if err != nil {
		syncCliUIColor(cmd)
		fmt.Fprint(os.Stderr, cliui.FormatCLIError(err.Error(), last))
		os.Exit(1)
	}
}
