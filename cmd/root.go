package cmd

import (
	_ "embed"
	"io"
	"os"

	"github.com/spf13/cobra"
)

//go:embed SKILL.md
var skillMarkdown string

var rootCmd = newRootCommand()

// cyclo-allow(side_effect_density): Construct the Cobra command and bind help or skill output at the CLI boundary.
func newRootCommand() *cobra.Command {
	var skill bool
	command := &cobra.Command{
		Use:   "openrpc-linter",
		Short: "A linter for OpenRPC documents",
		Long:  "Fast, extensible linter for OpenRPC documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !skill {
				return cmd.Help()
			}
			_, err := io.WriteString(cmd.OutOrStdout(), skillMarkdown)
			return err
		},
	}
	command.Flags().BoolVar(&skill, "skill", false, "Print the Markdown skill instructions")
	return command
}

func Execute() {
	execute(rootCmd.Execute, os.Exit)
}

// cyclo-allow(side_effect_density): Execute the root command and delegate a failed process exit to the injected exit function.
func execute(run func() error, exit func(int)) {
	if err := run(); err != nil {
		exit(1)
	}
}
