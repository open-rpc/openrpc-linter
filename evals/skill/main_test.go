package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func evalWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
}

func evalFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	workspace := t.TempDir()
	fixtures := filepath.Join(root, "evals", "skill", "fixtures")
	evalWrite(t, filepath.Join(fixtures, "openrpc.json"), `{"methods":[{"name":"ping"}]}`)
	evalWrite(t, filepath.Join(fixtures, "rules.yml"), "rules: unchanged\n")
	if err := prepare(root, workspace); err != nil {
		t.Fatal(err)
	}
	return root, workspace
}

func TestEvalDocumentContract(t *testing.T) {
	for _, tc := range []struct{ name, updated, message string }{
		{"success", `{"methods":[{"name":"ping","description":"Returns PONG."}]}`, ""},
		{"missing method", `{"methods":[]}`, "original ping"},
		{"missing description", `{"methods":[{"name":"ping"}]}`, "mentioning pong"},
		{"unrelated change", `{"methods":[{"name":"other","description":"pong"}]}`, "unrelated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, workspace := evalFixture(t)
			evalWrite(t, filepath.Join(workspace, "openrpc.json"), tc.updated)
			err := check(root, workspace)
			if tc.message == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("got %v, want %q", err, tc.message)
			}
		})
	}
}

func TestEvalFileFailures(t *testing.T) {
	root, workspace := evalFixture(t)
	fixtures := filepath.Join(root, "evals", "skill", "fixtures")
	if _, err := document("missing.json"); err == nil {
		t.Fatal("missing document")
	}
	malformed := filepath.Join(t.TempDir(), "bad.json")
	evalWrite(t, malformed, "{")
	if _, err := document(malformed); err == nil {
		t.Fatal("malformed document")
	}
	if err := check("missing", workspace); err == nil {
		t.Fatal("missing original")
	}
	if err := check(root, "missing"); err == nil {
		t.Fatal("missing updated")
	}
	if err := checkRules("missing", workspace); err == nil {
		t.Fatal("missing original rules")
	}
	if err := checkRules(fixtures, "missing"); err == nil {
		t.Fatal("missing workspace rules")
	}
	evalWrite(t, filepath.Join(workspace, "rules.yml"), "changed")
	if err := checkRules(fixtures, workspace); err == nil || !strings.Contains(err.Error(), "rules.yml changed") {
		t.Fatalf("changed rules: %v", err)
	}
	if err := prepare("missing", workspace); err == nil {
		t.Fatal("missing fixtures")
	}
	blocked := filepath.Join(t.TempDir(), "blocked")
	evalWrite(t, blocked, "file")
	if err := prepare(root, blocked); err == nil {
		t.Fatal("blocked workspace")
	}
}

func TestEvalCommandsAndLogs(t *testing.T) {
	workspace := t.TempDir()
	dir := t.TempDir()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := loggedCommand(workspace, dir, "", filepath.Join(dir, "success.log"), "help", binary, "-test.list=TestEvalDocumentContract"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "success.log"))
	if err != nil || !strings.Contains(string(data), "TestEvalDocumentContract") {
		t.Fatalf("log: %q %v", data, err)
	}
	if err := loggedCommand(workspace, dir, "", filepath.Join(dir, "failure.log"), "bad", binary, "-invalid-flag"); err == nil || !strings.Contains(err.Error(), "failure.log") {
		t.Fatalf("failed command: %v", err)
	}
	if err := loggedCommand(workspace, dir, "", dir, "write", binary, "-test.list=none"); err == nil {
		t.Fatal("log write failure")
	}
	if _, err := preparePrompt(workspace, dir, dir, "missing-command"); err == nil {
		t.Fatal("missing skill command")
	}
}

func TestEvalExplicitAgentAndSetupFailure(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"eval", binary, "-test.list=none"}
	if err := runAgent(t.TempDir(), t.TempDir(), t.TempDir(), "input"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	if _, _, _, err := setup(); err == nil {
		t.Fatal("expected temp directory failure")
	}
}
