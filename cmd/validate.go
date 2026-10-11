package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/open-rpc/openrpc-linter/metaschema"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate an OpenRPC document",
	Long:  "Validate an OpenRPC document against basic OpenRPC specification requirements. Defaults to 'openrpc.json' if no file is specified.",
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	filename := commandArgument(args, "openrpc.json")

	openrpc, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Error reading %s: %w", filename, err)
	}

	data, parseErr := jsonschema.UnmarshalJSON(bytes.NewReader(openrpc))
	if parseErr != nil {
		return fmt.Errorf("Error parsing JSON: %w", parseErr)
	}

	version, versionErr := metaschema.Version(data)
	if versionErr != nil {
		return fmt.Errorf("Error getting OpenRPC version: %w", versionErr)
	}
	meta, metaErr := metaschema.ForVersion(version)
	if metaErr != nil {
		return metaErr
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Validating OpenRPC document: %s (OpenRPC %s)\n", filename, version)

	schema, compileErr := meta.Compile()
	if compileErr != nil {
		return fmt.Errorf("Error compiling schema: %w", compileErr)
	}

	if err := schema.Validate(data); err != nil {
		return fmt.Errorf("❌ Validation failed: %w", err)
	}

	_, writeErr := fmt.Fprintln(cmd.OutOrStdout(), "✅ OpenRPC document is valid!")
	return writeErr
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
