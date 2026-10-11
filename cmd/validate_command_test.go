package cmd

import (
	"github.com/open-rpc/openrpc-linter/metaschema"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"io"
	"os"
	"testing"
)

func TestValidateDefaultFilename(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("openrpc.json", []byte(`{"openrpc":"1.4.0","info":{"title":"API","version":"1"},"methods":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	command := &cobra.Command{}
	command.SetOut(io.Discard)
	if err := runValidate(command, nil); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReturnsSchemaCompilationError(t *testing.T) {
	meta, err := metaschema.Latest()
	if err != nil {
		t.Fatal(err)
	}
	root := meta.Root()
	original := root["type"]
	root["type"] = 42
	t.Cleanup(func() { root["type"] = original })
	path := filepath.Join(t.TempDir(), "openrpc.json")
	if err := os.WriteFile(path, []byte(`{"openrpc":"1.4.0"}`), 0600); err != nil {
		t.Fatal(err)
	}
	command := &cobra.Command{}
	command.SetOut(io.Discard)
	if err := runValidate(command, []string{path}); err == nil || !strings.Contains(err.Error(), "Error compiling schema") {
		t.Fatalf("compile error: %v", err)
	}
}
