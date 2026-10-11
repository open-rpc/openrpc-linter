package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitCreatesBasicRulesFile(t *testing.T) {
	dir := t.TempDir()
	rulesFile := filepath.Join(dir, "rules.yml")

	var output bytes.Buffer
	err := RunInit(InitOptions{
		RulesFile: rulesFile,
		Output:    &output,
	})
	if err != nil {
		t.Fatalf("RunInit should create rules file: %v", err)
	}

	content, err := os.ReadFile(rulesFile)
	if err != nil {
		t.Fatalf("read rules file: %v", err)
	}
	if string(content) != basicRulesYAML {
		t.Fatalf("expected basic rules YAML %q, got %q", basicRulesYAML, string(content))
	}
	if !strings.Contains(output.String(), "Created "+rulesFile) {
		t.Fatalf("expected created output, got %q", output.String())
	}
}

func TestRunInitDoesNotOverwriteExistingRulesFile(t *testing.T) {
	dir := t.TempDir()
	rulesFile := filepath.Join(dir, "rules.yml")
	original := []byte("rules:\n  custom: {}\n")
	if err := os.WriteFile(rulesFile, original, 0644); err != nil {
		t.Fatalf("write existing rules file: %v", err)
	}

	var output bytes.Buffer
	err := RunInit(InitOptions{
		RulesFile: rulesFile,
		Output:    &output,
	})
	if err == nil {
		t.Fatal("expected RunInit to fail for existing rules file")
	}

	content, err := os.ReadFile(rulesFile)
	if err != nil {
		t.Fatalf("read rules file: %v", err)
	}
	if string(content) != string(original) {
		t.Fatalf("expected existing content to remain %q, got %q", string(original), string(content))
	}
	if !strings.Contains(output.String(), "already exists; use --force to overwrite") {
		t.Fatalf("expected overwrite guidance, got %q", output.String())
	}
}

func TestRunInitForceOverwritesExistingRulesFile(t *testing.T) {
	dir := t.TempDir()
	rulesFile := filepath.Join(dir, "rules.yml")
	if err := os.WriteFile(rulesFile, []byte("rules:\n  custom: {}\n"), 0644); err != nil {
		t.Fatalf("write existing rules file: %v", err)
	}

	err := RunInit(InitOptions{
		RulesFile: rulesFile,
		Output:    &bytes.Buffer{},
		Force:     true,
	})
	if err != nil {
		t.Fatalf("RunInit should overwrite with force: %v", err)
	}

	content, err := os.ReadFile(rulesFile)
	if err != nil {
		t.Fatalf("read rules file: %v", err)
	}
	if string(content) != basicRulesYAML {
		t.Fatalf("expected basic rules YAML %q, got %q", basicRulesYAML, string(content))
	}
}

func TestRunInitReturnsOutputFailure(t *testing.T) {
	rulesFile := filepath.Join(t.TempDir(), "rules.yml")
	want := fmt.Errorf("output unavailable")
	if err := RunInit(InitOptions{RulesFile: rulesFile, Output: failingWriter{err: want}}); err != want {
		t.Fatalf("expected write error %v, got %v", want, err)
	}
	content, err := os.ReadFile(rulesFile)
	if err != nil || string(content) != basicRulesYAML {
		t.Fatalf("rules file must be written before the status output: %q, %v", content, err)
	}
}
