package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

const basicRulesYAML = `extends:
  - recommended
`

var initForce bool

type InitOptions struct {
	RulesFile string
	Output    io.Writer
	Force     bool
}

func RunInit(opts InitOptions) error {
	if opts.RulesFile == "" {
		opts.RulesFile = "rules.yml"
	}
	if opts.Output == nil {
		opts.Output = io.Discard
	}

	flags := os.O_WRONLY | os.O_CREATE
	if opts.Force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}

	file, err := os.OpenFile(opts.RulesFile, flags, 0644)
	if err != nil {
		if os.IsExist(err) {
			fmt.Fprintf(opts.Output, "%s already exists; use --force to overwrite\n", opts.RulesFile)
		}
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(basicRulesYAML); err != nil {
		return err
	}

	fmt.Fprintf(opts.Output, "Created %s\n", opts.RulesFile)
	return nil
}

var initCmd = &cobra.Command{
	Use:   "init [rules-file]",
	Short: "Create a basic rules.yml",
	Long:  "Create a basic rules.yml that extends the bundled recommended rules. Defaults to 'rules.yml' if no file is specified.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runInitCommand(cmd, args, os.Exit)
	},
}

func runInitCommand(cmd *cobra.Command, args []string, exit func(int)) {
	opts := InitOptions{
		RulesFile: commandArgument(args, "rules.yml"),
		Output:    cmd.OutOrStdout(),
		Force:     initForce,
	}

	execute(func() error { return RunInit(opts) }, exit)
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite an existing rules file")
	rootCmd.AddCommand(initCmd)
}
