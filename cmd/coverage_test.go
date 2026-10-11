package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/open-rpc/openrpc-linter/types"
	"github.com/spf13/cobra"
)

func writeCoverageFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLintInputFailures(t *testing.T) {
	for _, tc := range []struct{ name, content, message string }{
		{"malformed", "{", "Error parsing"}, {"unsupported", `{"openrpc":"9.0.0"}`, "Error selecting"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			_, err := loadLintDocument(writeCoverageFile(t, "openrpc.json", tc.content), &output)
			if err == nil || !strings.Contains(output.String(), tc.message) {
				t.Fatalf("got %v, %q", err, output.String())
			}
		})
	}
	var output bytes.Buffer
	if err := RunLint(LintOptions{OpenRPCFile: "missing.json", Output: &output}); err == nil {
		t.Fatal("accepted missing document")
	}
	doc := writeCoverageFile(t, "openrpc.json", `{"info":{"title":"API"}}`)
	if err := RunLint(LintOptions{OpenRPCFile: doc, RulesFile: "missing.yml", Output: &output}); err == nil {
		t.Fatal("accepted missing rules")
	}
	empty := writeCoverageFile(t, "rules.yml", "rules: {}")
	if _, err := loadLintRules(empty, &output); err == nil {
		t.Fatal("accepted empty rules")
	}
	unknown := writeCoverageFile(t, "rules.yml", "extends: [unknown]")
	if _, err := loadLintRules(unknown, &output); err == nil {
		t.Fatal("accepted unknown extension")
	}
	invalid := writeCoverageFile(t, "rules.yml", "rules:\n  r:\n    severity: invalid\n")
	if err := RunLint(LintOptions{OpenRPCFile: doc, RulesFile: invalid, Output: &output}); err == nil {
		t.Fatal("accepted invalid severity")
	}
	valid := writeCoverageFile(t, "rules.yml", "rules:\n  r:\n    given: '$'\n")
	want := errors.New("writer failed")
	if err := RunLint(LintOptions{OpenRPCFile: doc, RulesFile: valid, Output: failingWriter{want}}); !errors.Is(err, want) {
		t.Fatalf("write error: %v", err)
	}
}

func TestRuleExecutionErrorDiagnostic(t *testing.T) {
	got, count := evaluateLintRule("r", types.Rule{Given: "$", Then: &types.RuleAction{Function: "unknown"}}, types.RuleFunctionContext{})
	if count != 1 || len(got) != 1 || got[0].RuleID != "r" || got[0].Severity != types.SeverityError {
		t.Fatalf("execution diagnostic: %+v count %d", got, count)
	}
	if severity, err := normalizeSeverity(types.SeverityInfo); err != nil || severity != types.SeverityInfo {
		t.Fatalf("info: %v %v", severity, err)
	}
}

func TestReferenceBoundaryCases(t *testing.T) {
	if _, err := resolveRefs(make(chan int)); err == nil {
		t.Fatal("accepted non-JSON value")
	}
	doc := map[string]any{"value": "ping", "arr": []any{"pong"}}
	if got := resolveJSONPointer("", doc); !reflect.DeepEqual(got, doc) {
		t.Fatalf("root pointer: %v", got)
	}
	for _, path := range []string{"missing", "arr/0", "value/child"} {
		if got := resolveJSONPointer(path, doc); got != nil {
			t.Fatalf("unsupported pointer %q: %v", path, got)
		}
	}
	ref := map[string]any{"$ref": "#/missing"}
	if got := resolveRefsRecursive(ref, doc, types.ResolvingRefs{}); !reflect.DeepEqual(got, ref) {
		t.Fatalf("unresolved reference: %v", got)
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

func TestExecuteHelp(t *testing.T) {
	old := rootCmd
	t.Cleanup(func() { rootCmd = old })
	rootCmd = newRootCommand()
	rootCmd.SetArgs([]string{"--help"})
	rootCmd.SetOut(io.Discard)
	Execute()
}
