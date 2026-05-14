package auth

import "github.com/spf13/cobra"

func NewAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication (login, logout, status)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newLoginCommand(),
		newLogoutCommand(),
		newStatusCommand(),
		newModelsCommand(),
		newWeixinCommand(),
		newWeComCommand(),
	)

	return cmd
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current auth status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return authStatusCmd()
		},
	}
}

func newLogoutCommand() *cobra.Command {
	var provider string

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return authLogoutCmd(provider)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider to logout from (openai, anthropic); empty = all")

	return cmd
}

func newModelsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "Show available models",
		RunE: func(_ *cobra.Command, _ []string) error {
			return authModelsCmd()
		},
	}
}
