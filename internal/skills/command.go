package skills

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/conglinyizhi/SylastraClaws/internal"
	"github.com/conglinyizhi/SylastraClaws/pkg/skills"
)

type deps struct {
	workspace    string
	skillsLoader *skills.SkillsLoader
}

func NewSkillsCommand() *cobra.Command {
	var d deps

	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage skills",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}

			d.workspace = cfg.WorkspacePath()

			globalDir := filepath.Dir(internal.GetConfigPath())
			globalSkillsDir := filepath.Join(globalDir, "skills")
			builtinSkillsDir := filepath.Join(globalDir, "sylastraclaws", "skills")
			d.skillsLoader = skills.NewSkillsLoader(d.workspace, globalSkillsDir, builtinSkillsDir)

			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	loaderFn := func() (*skills.SkillsLoader, error) {
		if d.skillsLoader == nil {
			return nil, fmt.Errorf("skills loader is not initialized")
		}
		return d.skillsLoader, nil
	}

	workspaceFn := func() (string, error) {
		if d.workspace == "" {
			return "", fmt.Errorf("workspace is not initialized")
		}
		return d.workspace, nil
	}

	cmd.AddCommand(
		newListCommand(loaderFn),
		newInstallCommand(),
		newInstallBuiltinCommand(workspaceFn),
		newListBuiltinCommand(),
		newRemoveCommand(),
		newSearchCommand(),
		newShowCommand(loaderFn),
	)

	return cmd
}

func newListCommand(loaderFn func() (*skills.SkillsLoader, error)) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List installed skills",
		Example: "sylastraclaws skills list",
		RunE: func(_ *cobra.Command, _ []string) error {
			loader, err := loaderFn()
			if err != nil {
				return err
			}
			skillsListCmd(loader)
			return nil
		},
	}
}

func newRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove installed skill",
		Args:    cobra.ExactArgs(1),
		Example: "sylastraclaws skills remove weather",
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return err
			}
			return skillsRemoveFromWorkspace(cfg.WorkspacePath(), cfg.Tools.Skills, args[0])
		},
	}
}

func newSearchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search available skills",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := ""
			if len(args) == 1 {
				query = args[0]
			}
			skillsSearchCmd(query)
			return nil
		},
	}
}

func newShowCommand(loaderFn func() (*skills.SkillsLoader, error)) *cobra.Command {
	return &cobra.Command{
		Use:     "show",
		Short:   "Show skill details",
		Args:    cobra.ExactArgs(1),
		Example: "sylastraclaws skills show weather",
		RunE: func(_ *cobra.Command, args []string) error {
			loader, err := loaderFn()
			if err != nil {
				return err
			}
			skillsShowCmd(loader, args[0])
			return nil
		},
	}
}

func newInstallBuiltinCommand(workspaceFn func() (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:     "install-builtin",
		Short:   "Install all builtin skills to workspace",
		Example: "sylastraclaws skills install-builtin",
		RunE: func(_ *cobra.Command, _ []string) error {
			workspace, err := workspaceFn()
			if err != nil {
				return err
			}
			skillsInstallBuiltinCmd(workspace)
			return nil
		},
	}
}

func newListBuiltinCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list-builtin",
		Short:   "List available builtin skills",
		Example: "sylastraclaws skills list-builtin",
		Run: func(_ *cobra.Command, _ []string) {
			skillsListBuiltinCmd()
		},
	}
}
