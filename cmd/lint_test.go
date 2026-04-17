package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRunLint(t *testing.T) {
	// Create a temporary OpenRPC file without description
	openrpcContent := map[string]interface{}{
		"info": map[string]interface{}{
			"title":   "Test API",
			"version": "1.0.0",
			// no description field
		},
	}

	openrpcData, err := json.Marshal(openrpcContent)
	if err != nil {
		t.Fatalf("Failed to create test OpenRPC content: %v", err)
	}

	tempOpenRPC, err := os.CreateTemp("", "test-openrpc-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp OpenRPC file: %v", err)
	}
	defer os.Remove(tempOpenRPC.Name())

	if _, err := tempOpenRPC.Write(openrpcData); err != nil {
		t.Fatalf("Failed to write test OpenRPC file: %v", err)
	}
	tempOpenRPC.Close()

	// Create a temporary rules file
	rulesContent := `description: "Test rules"
rules:
  info-description:
    description: "Info must have description"
    given: "$.info"
    then:
      field: "description"
      function: "truthy"
`

	tempRules, err := os.CreateTemp("", "test-rules-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp rules file: %v", err)
	}
	defer os.Remove(tempRules.Name())

	if _, err := tempRules.WriteString(rulesContent); err != nil {
		t.Fatalf("Failed to write test rules file: %v", err)
	}
	tempRules.Close()

	// Test the RunLint function directly
	var output bytes.Buffer
	opts := LintOptions{
		OpenRPCFile: tempOpenRPC.Name(),
		RulesFile:   tempRules.Name(),
		Output:      &output,
	}

	err = RunLint(opts)
	if err == nil {
		t.Fatalf("Expected RunLint to return error for linting violations, but got nil")
	}

	// Should be a linting error, not a technical error
	if !strings.Contains(err.Error(), "linting error(s)") {
		t.Fatalf("Expected linting error, but got: %v", err)
	}

	outputStr := output.String()
	t.Logf("Lint output:\n%s", outputStr)

	// Verify the output contains the expected error
	if !strings.Contains(outputStr, "❌") {
		t.Errorf("Expected error output with ❌, but got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "Missing required field 'description' at $.info") {
		t.Errorf("Expected 'Missing required field 'description' at $.info' in output, but got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "1 error(s) found") {
		t.Errorf("Expected error summary in output, but got: %s", outputStr)
	}
}

func TestRunLintSuccess(t *testing.T) {
	// Create a temporary OpenRPC file WITH description
	openrpcContent := map[string]interface{}{
		"info": map[string]interface{}{
			"title":       "Test API",
			"version":     "1.0.0",
			"description": "A test API description",
		},
	}

	openrpcData, err := json.Marshal(openrpcContent)
	if err != nil {
		t.Fatalf("Failed to create test OpenRPC content: %v", err)
	}

	tempOpenRPC, err := os.CreateTemp("", "test-openrpc-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp OpenRPC file: %v", err)
	}
	defer os.Remove(tempOpenRPC.Name())

	if _, err := tempOpenRPC.Write(openrpcData); err != nil {
		t.Fatalf("Failed to write test OpenRPC file: %v", err)
	}
	tempOpenRPC.Close()

	// Create a temporary rules file
	rulesContent := `description: "Test rules"
rules:
  info-description:
    description: "Info must have description"
    given: "$.info"
    then:
      field: "description"
      function: "truthy"
`

	tempRules, err := os.CreateTemp("", "test-rules-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp rules file: %v", err)
	}
	defer os.Remove(tempRules.Name())

	if _, err := tempRules.WriteString(rulesContent); err != nil {
		t.Fatalf("Failed to write test rules file: %v", err)
	}
	tempRules.Close()

	// Test the RunLint function directly
	var output bytes.Buffer
	opts := LintOptions{
		OpenRPCFile: tempOpenRPC.Name(),
		RulesFile:   tempRules.Name(),
		Output:      &output,
	}

	err = RunLint(opts)
	if err != nil {
		t.Fatalf("RunLint should succeed when no linting violations, but got: %v", err)
	}

	outputStr := output.String()
	t.Logf("Lint output:\n%s", outputStr)

	if !strings.Contains(outputStr, "✅") {
		t.Errorf("Expected success output with ✅, but got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "All 1 rules passed") {
		t.Errorf("Expected 'All 1 rules passed' in output, but got: %s", outputStr)
	}
}

func TestRunLintWarningSeverityDoesNotFail(t *testing.T) {
	// Create a temporary OpenRPC file without description to trigger a truthy violation.
	openrpcContent := map[string]interface{}{
		"info": map[string]interface{}{
			"title":   "Test API",
			"version": "1.0.0",
		},
	}

	openrpcData, err := json.Marshal(openrpcContent)
	if err != nil {
		t.Fatalf("Failed to create test OpenRPC content: %v", err)
	}

	tempOpenRPC, err := os.CreateTemp("", "test-openrpc-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp OpenRPC file: %v", err)
	}
	defer os.Remove(tempOpenRPC.Name())

	if _, err := tempOpenRPC.Write(openrpcData); err != nil {
		t.Fatalf("Failed to write test OpenRPC file: %v", err)
	}
	tempOpenRPC.Close()

	// severity: warn should not cause RunLint to return an error.
	rulesContent := `description: "Test rules"
rules:
  info-description:
    description: "Info must have description"
    given: "$.info"
    severity: "warn"
    then:
      field: "description"
      function: "truthy"
`

	tempRules, err := os.CreateTemp("", "test-rules-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp rules file: %v", err)
	}
	defer os.Remove(tempRules.Name())

	if _, err := tempRules.WriteString(rulesContent); err != nil {
		t.Fatalf("Failed to write test rules file: %v", err)
	}
	tempRules.Close()

	var output bytes.Buffer
	opts := LintOptions{
		OpenRPCFile: tempOpenRPC.Name(),
		RulesFile:   tempRules.Name(),
		Output:      &output,
	}

	err = RunLint(opts)
	if err != nil {
		t.Fatalf("RunLint should not fail for warn severity violations, but got: %v\nOutput:\n%s", err, output.String())
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Missing required field 'description' at $.info") {
		t.Fatalf("Expected warning violation output, got:\n%s", outputStr)
	}
}

func TestRunLintInvalidSeverityFailsFast(t *testing.T) {
	openrpcContent := map[string]interface{}{
		"info": map[string]interface{}{
			"title":   "Test API",
			"version": "1.0.0",
		},
	}

	openrpcData, err := json.Marshal(openrpcContent)
	if err != nil {
		t.Fatalf("Failed to create test OpenRPC content: %v", err)
	}

	tempOpenRPC, err := os.CreateTemp("", "test-openrpc-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp OpenRPC file: %v", err)
	}
	defer os.Remove(tempOpenRPC.Name())

	if _, err := tempOpenRPC.Write(openrpcData); err != nil {
		t.Fatalf("Failed to write test OpenRPC file: %v", err)
	}
	tempOpenRPC.Close()

	rulesContent := `description: "Test rules"
rules:
  info-description:
    description: "Info must have description"
    given: "$.info"
    severity: "critical"
    then:
      field: "description"
      function: "truthy"
`

	tempRules, err := os.CreateTemp("", "test-rules-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp rules file: %v", err)
	}
	defer os.Remove(tempRules.Name())

	if _, err := tempRules.WriteString(rulesContent); err != nil {
		t.Fatalf("Failed to write test rules file: %v", err)
	}
	tempRules.Close()

	var output bytes.Buffer
	opts := LintOptions{
		OpenRPCFile: tempOpenRPC.Name(),
		RulesFile:   tempRules.Name(),
		Output:      &output,
	}

	err = RunLint(opts)
	if err == nil {
		t.Fatalf("Expected RunLint to fail for invalid severity")
	}

	if !strings.Contains(err.Error(), "invalid severity") {
		t.Fatalf("Expected invalid severity error, got: %v", err)
	}
}
