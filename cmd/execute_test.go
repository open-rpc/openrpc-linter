package cmd

import (
	"io"
	"testing"
)

func TestExecuteHelp(t *testing.T) {
	old := rootCmd
	t.Cleanup(func() { rootCmd = old })
	rootCmd = newRootCommand()
	rootCmd.SetArgs([]string{"--help"})
	rootCmd.SetOut(io.Discard)
	Execute()
}

func TestExecuteRequestsFailureExit(t *testing.T) {
	old := rootCmd
	t.Cleanup(func() { rootCmd = old })
	rootCmd = newRootCommand()
	rootCmd.SetArgs([]string{"--unknown"})
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	code := 0
	execute(func(value int) { code = value })
	if code != 1 {
		t.Fatalf("exit code: %d", code)
	}
}
