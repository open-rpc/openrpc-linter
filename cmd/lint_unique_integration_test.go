package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/open-rpc/openrpc-linter/types"
)

func TestRunLintUniqueBadDocumentReportsEveryRule(t *testing.T) {
	var output bytes.Buffer

	opts := LintOptions{
		OpenRPCFile: filepath.Join("testdata", "unique", "bad-document.json"),
		RulesFile:   filepath.Join("..", "rules.yml"),
		Output:      &output,
		Format:      "json",
	}

	err := RunLint(opts)
	if err == nil {
		t.Fatalf("expected linting errors for duplicate stress fixture")
	}

	var results []types.RuleFunctionResult
	if decodeErr := json.Unmarshal(output.Bytes(), &results); decodeErr != nil {
		t.Fatalf("failed to parse json reporter output: %v\noutput:\n%s", decodeErr, output.String())
	}

	if len(results) == 0 {
		t.Fatalf("expected non-empty lint results")
	}

	countByRuleID := make(map[string]int)
	for _, result := range results {
		if result.Message == "" {
			continue
		}
		countByRuleID[result.RuleID]++
	}

	requiredRuleIDs := []string{
		"unique-method-names",
		"unique-method-summary",
		"unique-param-names-per-method",
		"unique-error-codes-per-method",
		"unique-example-names-per-method",
		"unique-content-descriptor-names",
		"unique-schema-titles",
	}

	for _, ruleID := range requiredRuleIDs {
		if countByRuleID[ruleID] == 0 {
			t.Fatalf("expected at least one violation for rule %q; got counts: %+v", ruleID, countByRuleID)
		}
	}

	if countByRuleID["unique-param-names-per-method"] != 1 {
		t.Fatalf("expected exactly one per-method param-name violation; got counts: %+v", countByRuleID)
	}
}

func TestRunLintUniquePerMethodRulesDoNotLeakAcrossMethods(t *testing.T) {
	openrpcContent := `{
  "openrpc": "1.3.2",
  "info": {
    "title": "Scoped Unique Test API",
    "version": "1.0.0"
  },
  "methods": [
    {
      "name": "firstMethod",
      "params": [
        {
          "name": "accountId"
        }
      ]
    },
    {
      "name": "secondMethod",
      "params": [
        {
          "name": "accountId"
        }
      ]
    }
  ]
}`

	rulesContent := `description: "Scoped unique rules"
rules:
  unique-param-names-per-method:
    description: "Param names should be unique within each method"
    given: "$.methods[*].params[*]"
    severity: "error"
    then:
      field: "name"
      function: "unique"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err != nil {
		t.Fatalf("expected no lint failures when duplicates only exist across methods, got: %v\nresults: %+v", err, results)
	}

	if len(results) != 0 {
		t.Fatalf("expected no results when each method has unique param names, got: %+v", results)
	}
}

func TestRunLintUniqueWarnSeverityReportsViolationWithoutFailing(t *testing.T) {
	openrpcContent := `{
  "openrpc": "1.3.2",
  "info": {
    "title": "Warn Unique Test API",
    "version": "1.0.0"
  },
  "methods": [
    {
      "name": "firstMethod",
      "summary": "Shared summary"
    },
    {
      "name": "secondMethod",
      "summary": "Shared summary"
    }
  ]
}`

	rulesContent := `description: "Warning unique rules"
rules:
  unique-method-summaries:
    description: "Method summaries should be unique"
    given: "$.methods[*]"
    severity: "warn"
    then:
      field: "summary"
      function: "unique"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err != nil {
		t.Fatalf("expected warn-only duplicate to report without failing, got: %v\nresults: %+v", err, results)
	}

	if len(results) != 1 {
		t.Fatalf("expected one warning result, got: %+v", results)
	}
	if results[0].RuleID != "unique-method-summaries" {
		t.Fatalf("expected rule id to be set, got: %+v", results[0])
	}
	if results[0].Message != `Duplicate value for field 'summary': "Shared summary"` {
		t.Fatalf("unexpected warning message: %+v", results[0])
	}
	if !reflect.DeepEqual(results[0].Path, []string{"$['methods']"}) {
		t.Fatalf("expected JSON output to include collection path, got: %+v", results[0].Path)
	}
}

func TestRunLintUniqueInvalidArrayItemsReportError(t *testing.T) {
	openrpcContent := `{
  "openrpc": "1.3.2",
  "info": {
    "title": "Primitive Unique Test API",
    "version": "1.0.0"
  },
  "tags": ["alpha", "beta", "alpha"]
}`

	rulesContent := `description: "Primitive unique rules"
rules:
  unique-tags:
    description: "Tags should be unique"
    given: "$.tags[*]"
    severity: "error"
    then:
      field: "name"
      function: "unique"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err == nil {
		t.Fatalf("expected unique to report invalid primitive selections, got nil error and results: %+v", results)
	}

	if len(results) == 0 {
		t.Fatalf("expected at least one result for invalid primitive selections")
	}
}

func TestRunLintUniqueIgnoreMissingDefaultsToTrue(t *testing.T) {
	openrpcContent := `{
  "openrpc": "1.3.2",
  "info": {
    "title": "Ignore Missing Default Test API",
    "version": "1.0.0"
  },
  "methods": [
    {
      "name": "firstMethod"
    },
    {
      "name": "secondMethod"
    }
  ]
}`

	rulesContent := `description: "Ignore missing defaults to true"
rules:
  unique-method-summaries:
    description: "Method summaries should be unique"
    given: "$.methods[*]"
    severity: "error"
    then:
      field: "summary"
      function: "unique"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err != nil {
		t.Fatalf("expected missing fields to be ignored by default, got: %v\nresults: %+v", err, results)
	}

	if len(results) != 0 {
		t.Fatalf("expected no results when all compared fields are missing and ignoreMissing defaults to true, got: %+v", results)
	}
}

func TestRunLintUniqueIgnoreMissingFalseTreatsMissingAsDuplicates(t *testing.T) {
	openrpcContent := `{
  "openrpc": "1.3.2",
  "info": {
    "title": "Ignore Missing False Test API",
    "version": "1.0.0"
  },
  "methods": [
    {
      "name": "firstMethod"
    },
    {
      "name": "secondMethod"
    }
  ]
}`

	rulesContent := `description: "Ignore missing false"
rules:
  unique-method-summaries:
    description: "Method summaries should be unique"
    given: "$.methods[*]"
    severity: "error"
    then:
      field: "summary"
      function: "unique"
      functionOptions:
        ignoreMissing: false
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err == nil {
		t.Fatalf("expected missing fields to become duplicate violations when ignoreMissing is false")
	}

	if len(results) != 1 {
		t.Fatalf("expected exactly one duplicate result for repeated missing values, got: %+v", results)
	}

	if results[0].Message != "Duplicate value for field 'summary': null" {
		t.Fatalf("unexpected duplicate message for missing values: %+v", results)
	}
}

func TestRunLintUniqueRequiresThenField(t *testing.T) {
	openrpcContent := `{
  "openrpc": "1.3.2",
  "info": {
    "title": "Missing Field Config Test API",
    "version": "1.0.0"
  },
  "methods": [
    {
      "name": "firstMethod"
    }
  ]
}`

	rulesContent := `description: "Missing then.field"
rules:
  unique-methods:
    description: "Methods should be unique somehow"
    given: "$.methods[*]"
    severity: "error"
    then:
      function: "unique"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err == nil {
		t.Fatalf("expected missing then.field to produce a lint failure")
	}

	if len(results) != 1 {
		t.Fatalf("expected a single configuration error result, got: %+v", results)
	}

	if results[0].Message != "unique function requires then.field" {
		t.Fatalf("unexpected configuration error message: %+v", results)
	}
}

func TestRunLintUniqueRejectsNonPrimitiveFieldValues(t *testing.T) {
	openrpcContent := `{
  "openrpc": "1.3.2",
  "info": {
    "title": "Non Primitive Unique Test API",
    "version": "1.0.0"
  },
  "methods": [
    {
      "name": "firstMethod",
      "result": {
        "name": "balance"
      }
    },
    {
      "name": "secondMethod",
      "result": {
        "name": "balance"
      }
    }
  ]
}`

	rulesContent := `description: "Non primitive field values"
rules:
  unique-method-result:
    description: "Method results should be unique"
    given: "$.methods[*]"
    severity: "error"
    then:
      field: "result"
      function: "unique"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err == nil {
		t.Fatalf("expected non-primitive compared values to produce a lint failure")
	}

	if len(results) != 2 {
		t.Fatalf("expected one unsupported-value result per method item, got: %+v", results)
	}

	for _, result := range results {
		if result.Message != "unique function does not support non-primitive value for field 'result'" {
			t.Fatalf("unexpected unsupported-value message: %+v", results)
		}
	}
}

func runLintJSON(t *testing.T, openrpcContent string, rulesContent string) ([]types.RuleFunctionResult, error) {
	t.Helper()

	tempOpenRPC, err := os.CreateTemp("", "unique-openrpc-*.json")
	if err != nil {
		t.Fatalf("failed to create temp OpenRPC file: %v", err)
	}
	defer os.Remove(tempOpenRPC.Name())

	if _, err := tempOpenRPC.WriteString(openrpcContent); err != nil {
		t.Fatalf("failed to write temp OpenRPC file: %v", err)
	}
	if err := tempOpenRPC.Close(); err != nil {
		t.Fatalf("failed to close temp OpenRPC file: %v", err)
	}

	tempRules, err := os.CreateTemp("", "unique-rules-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp rules file: %v", err)
	}
	defer os.Remove(tempRules.Name())

	if _, err := tempRules.WriteString(rulesContent); err != nil {
		t.Fatalf("failed to write temp rules file: %v", err)
	}
	if err := tempRules.Close(); err != nil {
		t.Fatalf("failed to close temp rules file: %v", err)
	}

	var output bytes.Buffer
	opts := LintOptions{
		OpenRPCFile: tempOpenRPC.Name(),
		RulesFile:   tempRules.Name(),
		Output:      &output,
		Format:      "json",
	}

	err = RunLint(opts)

	var results []types.RuleFunctionResult
	if decodeErr := json.Unmarshal(output.Bytes(), &results); decodeErr != nil {
		t.Fatalf("failed to parse json reporter output: %v\noutput:\n%s", decodeErr, output.String())
	}

	return results, err
}
