package cmd

import (
	"bytes"
	"errors"
	"github.com/open-rpc/openrpc-linter/reporters"
	"github.com/open-rpc/openrpc-linter/types"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLintFixture(t *testing.T, name, content string) string {
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
			_, err := loadLintDocument(writeLintFixture(t, "openrpc.json", tc.content), &output)
			if err == nil || !strings.Contains(output.String(), tc.message) {
				t.Fatalf("got %v, %q", err, output.String())
			}
		})
	}
	var output bytes.Buffer
	if err := RunLint(LintOptions{OpenRPCFile: "missing.json", Output: &output}); err == nil {
		t.Fatal("accepted missing document")
	}
	doc := writeLintFixture(t, "openrpc.json", `{"info":{"title":"API"}}`)
	if err := RunLint(LintOptions{OpenRPCFile: doc, RulesFile: "missing.yml", Output: &output}); err == nil {
		t.Fatal("accepted missing rules")
	}
	empty := writeLintFixture(t, "rules.yml", "rules: {}")
	if _, err := loadLintRules(empty, &output); err == nil {
		t.Fatal("accepted empty rules")
	}
	unknown := writeLintFixture(t, "rules.yml", "extends: [unknown]")
	if _, err := loadLintRules(unknown, &output); err == nil {
		t.Fatal("accepted unknown extension")
	}
	invalid := writeLintFixture(t, "rules.yml", "rules:\n  r:\n    severity: invalid\n")
	if err := RunLint(LintOptions{OpenRPCFile: doc, RulesFile: invalid, Output: &output}); err == nil {
		t.Fatal("accepted invalid severity")
	}
	valid := writeLintFixture(t, "rules.yml", "rules:\n  r:\n    given: '$'\n")
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

func TestLintCommandDefaultsAndExplicitFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("openrpc.json", []byte(`{"info":{"title":"API"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("rules.yml", []byte("rules:\n  r:\n    given: '$'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	oldRules, oldFormat := rulesFile, outputFormat
	t.Cleanup(func() { rulesFile = oldRules; outputFormat = oldFormat })
	rulesFile = "rules.yml"
	outputFormat = "text"
	command := &cobra.Command{}
	command.SetOut(io.Discard)
	lintCmd.Run(command, nil)
	lintCmd.Run(command, []string{"openrpc.json"})
	if _, ok := GetReporter("text").(*reporters.TextReporter); !ok {
		t.Fatal("text reporter")
	}
}

func TestPrepareLintDocumentRejectsNonJSONValue(t *testing.T) {
	var output bytes.Buffer
	if _, err := prepareLintDocument(make(chan int), &output); err == nil || !strings.Contains(output.String(), "Error resolving $refs") {
		t.Fatalf("resolution error: %v %q", err, output.String())
	}
}

func TestLintCommandRequestsFailureExit(t *testing.T) {
	command := &cobra.Command{}
	command.SetOut(io.Discard)
	code := 0
	runLintCommand(command, []string{filepath.Join(t.TempDir(), "missing.json")}, func(value int) { code = value })
	if code != 1 {
		t.Fatalf("exit code: %d", code)
	}
}
