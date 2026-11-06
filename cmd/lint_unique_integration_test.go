package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/open-rpc/openrpc-linter/types"
)

func TestRunLintUniqueScopeWithoutMatches(t *testing.T) {
	document := `{"openrpc":"1.3.2","info":{"title":"Test","version":"1.0.0"},"methods":[{"name":"same"},{"name":"same"}],"tags":[]}`
	for _, tc := range []struct {
		name          string
		scopeOption   string
		wantDuplicate bool
	}{
		{"omitted scope is global", "", true},
		{"empty scope is global", `scope: ""`, true},
		{"root scope includes all targets", `scope: "$"`, true},
		{"missing collection skips targets", `scope: "$.missing[*]"`, false},
		{"empty collection skips targets", `scope: "$.tags[*]"`, false},
		{"unmatched filter skips targets", `scope: "$.methods[?@.name == 'absent']"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rules := fmt.Sprintf(`rules:
  scoped-unique:
    given: "$.methods[*].name"
    then:
      function: unique
      functionOptions:
        %s
`, tc.scopeOption)
			results, err := runLintJSON(t, document, rules)
			if !tc.wantDuplicate {
				if err != nil || len(results) != 0 {
					t.Fatalf("targets outside scope must be skipped: results=%+v, err=%v", results, err)
				}
				return
			}
			if err == nil || len(results) != 1 || results[0].Message != `Duplicate value "same" (first seen at $['methods'][0]['name'])` {
				t.Fatalf("expected one global duplicate: results=%+v, err=%v", results, err)
			}
		})
	}
}

func TestRunLintUniqueBadDocumentReportsEveryRule(t *testing.T) {
	var output bytes.Buffer

	opts := LintOptions{
		OpenRPCFile: filepath.Join("..", "e2e", "fixtures", "unique", "bad-document.json"),
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
    given: "$.methods[*].params[*].name"
    severity: "error"
    then:
      function: "unique"
      functionOptions:
        scope: "$.methods[*]"
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
    given: "$.methods[*].summary"
    severity: "warn"
    then:
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
	if results[0].Severity != types.SeverityWarn {
		t.Fatalf("expected warn severity, got: %+v", results[0].Severity)
	}
	if results[0].Message != `Duplicate value "Shared summary" (first seen at $['methods'][0]['summary'])` {
		t.Fatalf("unexpected warning message: %+v", results[0])
	}
	if !reflect.DeepEqual(results[0].Path, []string{"$['methods'][1]['summary']"}) {
		t.Fatalf("expected JSON output to include duplicate field path, got: %+v", results[0].Path)
	}
}

func TestRunLintUniqueDetectsPrimitiveDuplicates(t *testing.T) {
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
      function: "unique"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err == nil {
		t.Fatalf("expected duplicate primitive to produce a lint failure, got results: %+v", results)
	}

	if len(results) != 1 {
		t.Fatalf("expected one duplicate result, got: %+v", results)
	}
	if results[0].Message != `Duplicate value "alpha" (first seen at $['tags'][0])` {
		t.Fatalf("unexpected duplicate message: %+v", results[0])
	}
	if !reflect.DeepEqual(results[0].Path, []string{"$['tags'][2]"}) {
		t.Fatalf("expected duplicate path to point at the second alpha, got: %+v", results[0].Path)
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
    given: "$.methods[*].summary"
    severity: "error"
    then:
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
    given: "$.methods[*].summary"
    severity: "error"
    then:
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

	if results[0].Message != "Duplicate value null (first seen at $['methods'][0]['summary'])" {
		t.Fatalf("unexpected duplicate message for missing values: %+v", results)
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
    given: "$.methods[*].result"
    severity: "error"
    then:
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
		if result.Message != "unique value must be primitive" {
			t.Fatalf("unexpected unsupported-value message: %+v", results)
		}
		if len(result.Path) == 0 {
			t.Fatalf("expected unsupported-value result to include field path, got: %+v", results)
		}
	}
}

func runLintJSON(t *testing.T, openrpcContent string, rulesContent string) ([]types.RuleFunctionResult, error) {
	t.Helper()

	openrpcFile := writeLintFixture(t, "openrpc.json", openrpcContent)
	rulesFile := writeLintFixture(t, "rules.yml", rulesContent)

	var output bytes.Buffer
	opts := LintOptions{
		OpenRPCFile: openrpcFile,
		RulesFile:   rulesFile,
		Output:      &output,
		Format:      "json",
	}

	err := RunLint(opts)

	var results []types.RuleFunctionResult
	if decodeErr := json.Unmarshal(output.Bytes(), &results); decodeErr != nil {
		t.Fatalf("failed to parse json reporter output: %v\noutput:\n%s", decodeErr, output.String())
	}

	return results, err
}
