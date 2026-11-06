package cmd

import (
	"bytes"
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"io"
	"os"
	"testing"
)

func TestInitCommandReportsFailures(t *testing.T) {
	wantOutputErr := errors.New("status output unavailable")
	for _, tc := range []struct {
		name   string
		path   string
		output io.Writer
		want   string
	}{
		{"missing parent", filepath.Join(t.TempDir(), "missing", "rules.yml"), io.Discard, "rules.yml"},
		{"status output", filepath.Join(t.TempDir(), "rules.yml"), failingWriter{err: wantOutputErr}, wantOutputErr.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var diagnostics bytes.Buffer
			command := &cobra.Command{}
			command.SetOut(tc.output)
			command.SetErr(&diagnostics)
			code := 0
			runInitCommand(command, []string{tc.path}, func(value int) { code = value })
			if code != 1 || !strings.Contains(diagnostics.String(), tc.want) {
				t.Fatalf("expected failure diagnostic containing %q: exit=%d, stderr=%q", tc.want, code, diagnostics.String())
			}
		})
	}
}

func TestInitDefaultsAndCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := RunInit(InitOptions{}); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile("rules.yml"); err != nil || string(data) != basicRulesYAML {
		t.Fatalf("default rules: %q %v", data, err)
	}
	oldForce := initForce
	t.Cleanup(func() { initForce = oldForce })
	initForce = false
	command := &cobra.Command{}
	command.SetOut(io.Discard)
	initCmd.Run(command, []string{"custom.yml"})
	if _, err := os.Stat("custom.yml"); err != nil {
		t.Fatal(err)
	}
	initForce = true
	initCmd.Run(command, nil)
}

func TestInitCommandRequestsFailureExit(t *testing.T) {
	command := &cobra.Command{}
	command.SetOut(io.Discard)
	code := 0
	runInitCommand(command, []string{t.TempDir()}, func(value int) { code = value })
	if code != 1 {
		t.Fatalf("exit code: %d", code)
	}
}

func TestInitReturnsWriteError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires /dev/full")
	}
	if err := RunInit(InitOptions{RulesFile: "/dev/full", Force: true}); err == nil {
		t.Fatal("expected write error")
	}
}
