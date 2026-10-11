package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRunLintDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name, description, severity, wantError string
		wantOutput                             []string
	}{
		{"missing description", "", "", "linting error(s)", []string{"info.description", "missing field 'description'", "error", "1 error found"}},
		{"valid description", "A test API description", "", "", []string{"All 1 rules passed"}},
		{"warning", "", "warn", "", []string{"info.description", "missing field 'description'", "warning", "1 warning found"}},
		{"invalid severity", "", "critical", "invalid severity", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := map[string]any{"title": "Test API", "version": "1.0.0"}
			if tc.description != "" {
				info["description"] = tc.description
			}
			document := writeLintDocument(t, map[string]any{"info": info})
			rules := writeLintFixture(t, "rules.yml", fmt.Sprintf(`rules:
  info-description:
    given: "$.info.description"
    severity: %q
    then:
      function: "truthy"
`, tc.severity))
			var output bytes.Buffer
			err := RunLint(LintOptions{OpenRPCFile: document, RulesFile: rules, Output: &output})
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v; output: %s", err, &output)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("got error %v, want %q; output: %s", err, tc.wantError, &output)
			}
			for _, want := range tc.wantOutput {
				if !strings.Contains(output.String(), want) {
					t.Errorf("missing %q in output: %s", want, &output)
				}
			}
		})
	}
}

func TestRunLintSeverityIgnoreDisablesRecommendedRule(t *testing.T) {
	openrpcContent := map[string]interface{}{
		"info": map[string]interface{}{
			"title":       "Test API",
			"version":     "1.0.0",
			"description": "A test API description",
		},
		"methods": []interface{}{
			map[string]interface{}{
				"name": "test_method",
				"errors": []interface{}{
					map[string]interface{}{
						"code":    -32000,
						"message": "Test error",
					},
				},
				"examples": []interface{}{
					map[string]interface{}{
						"name":   "test example",
						"params": []interface{}{},
						"result": map[string]interface{}{},
					},
				},
			},
		},
	}

	openrpcFile := writeLintDocument(t, openrpcContent)

	rulesContent := `description: "Test rules"
extends:
  - recommended
rules:
  info-license:
    severity: "ignore"
  method-summary:
    severity: "ignore"
  method-description:
    severity: "ignore"
  example-description:
    severity: "ignore"
`

	rulesFile := writeLintFixture(t, "rules.yml", rulesContent)

	var output bytes.Buffer
	opts := LintOptions{
		OpenRPCFile: openrpcFile,
		RulesFile:   rulesFile,
		Output:      &output,
	}

	err := RunLint(opts)
	if err != nil {
		t.Fatalf("RunLint should not fail when a recommended rule is disabled, but got: %v\nOutput:\n%s", err, output.String())
	}

	outputStr := output.String()
	if strings.Contains(outputStr, "method-description") {
		t.Fatalf("Expected disabled recommended rule to be omitted from output, got:\n%s", outputStr)
	}
	if strings.Contains(outputStr, "Missing field 'description'") {
		t.Fatalf("Expected method description violation to be skipped, got:\n%s", outputStr)
	}
}

func TestRunLintHandlesCyclicSchemaRef(t *testing.T) {
	openrpcContent := `{
  "openrpc": "1.4.0",
  "info": {
    "title": "Recursive Schema API",
    "version": "1.0.0"
  },
  "methods": [
    {
      "name": "getCategory",
      "params": [],
      "result": {
        "name": "category",
        "schema": {
          "$ref": "#/components/schemas/Category"
        }
      }
    }
  ],
  "components": {
    "schemas": {
      "Category": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string"
          },
          "parent": {
            "$ref": "#/components/schemas/Category"
          }
        }
      }
    }
  }
}`

	rulesContent := `description: "Cyclic ref smoke test"
rules:
  info-title:
    description: "Info title must exist"
    given: "$.info.title"
    severity: "error"
    then:
      function: "truthy"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err != nil {
		t.Fatalf("RunLint should handle cyclic schema refs, got: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no lint results, got %+v", results)
	}
}

// TestRunLintDescendantDescriptionReportsMissingCandidates exercises the
// schema-aware descendant path that the old truthy implementation could not
// satisfy: $..description must surface MISSING descriptions on every
// OpenRPC object the meta-schema says could legally hold one, not just the
// ones that already exist. The test relies on the full lint pipeline
// (resolveRefs -> selector.Build -> ExecuteRule) so a regression in any
// stage is caught.
func TestRunLintDescendantDescriptionReportsMissingCandidates(t *testing.T) {
	openrpcContent := `{
        "openrpc": "1.4.0",
        "info": {"title": "Demo", "version": "1.0.0"},
        "methods": [
          {"name": "foo", "params": [{"name": "p"}]}
        ]
      }`

	rulesContent := `description: "Descendant description rule"
rules:
  descendant-description:
    description: "Every OpenRPC object should have description"
    given: "$..description"
    severity: "error"
    then:
      function: "truthy"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err == nil {
		t.Fatalf("expected linting errors for missing descriptions")
	}

	// Three candidate parents per the v1.4 meta-schema: info, the method,
	// and the content descriptor. None of them have description.
	wantPaths := map[string]bool{
		"$['info']['description']":                    false,
		"$['methods'][0]['description']":              false,
		"$['methods'][0]['params'][0]['description']": false,
	}
	for _, r := range results {
		if r.RuleID != "descendant-description" {
			continue
		}
		for _, p := range r.Path {
			if _, ok := wantPaths[p]; ok {
				wantPaths[p] = true
			}
		}
	}
	for path, seen := range wantPaths {
		if !seen {
			t.Errorf("expected descendant-description to report %s; results: %+v", path, results)
		}
	}
}

// TestRunLintCompoundDescendantPath exercises a path with a descendant in
// the middle ($.methods..result.schema). The selector must peel the
// descendant ..result, then evaluate .schema as a parent-field check on
// each resolved result node — and report missing schema fields.
func TestRunLintCompoundDescendantPath(t *testing.T) {
	openrpcContent := `{
        "openrpc": "1.4.0",
        "info": {"title": "Demo", "version": "1.0.0"},
        "methods": [
          {"name": "foo", "params": [], "result": {"name": "r"}}
        ]
      }`

	rulesContent := `description: "Result schemas must exist"
rules:
  result-schema:
    description: "Each result must have a schema"
    given: "$.methods..result.schema"
    severity: "error"
    then:
      function: "truthy"
`

	results, err := runLintJSON(t, openrpcContent, rulesContent)
	if err == nil {
		t.Fatalf("expected linting errors for missing result.schema")
	}

	found := false
	for _, r := range results {
		if r.RuleID != "result-schema" {
			continue
		}
		for _, p := range r.Path {
			if p == "$['methods'][0]['result']['schema']" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected missing-schema diagnostic at $['methods'][0]['result']['schema'], got: %+v", results)
	}
}
