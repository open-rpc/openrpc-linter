package functions

import (
	"fmt"
	"strings"
	"testing"

	"github.com/open-rpc/openrpc-linter/types"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestSchemaRule(t *testing.T) {
	tests := []struct {
		name           string
		value          interface{}
		functionOpts   map[string]interface{}
		expectedResult string
	}{
		{
			name: "valid methods length",
			value: []interface{}{
				map[string]interface{}{"name": "ping"},
			},
			functionOpts: map[string]interface{}{
				"type":     "array",
				"minItems": 1,
			},
		},
		{
			name:  "invalid methods length",
			value: []interface{}{},
			functionOpts: map[string]interface{}{
				"type":     "array",
				"minItems": 1,
			},
			expectedResult: "Value does not match schema:",
		},
		{
			name:  "invalid function options",
			value: []interface{}{},
			functionOpts: map[string]interface{}{
				"type": 12,
			},
			expectedResult: "Invalid schema functionOptions:",
		},
		{
			name:           "missing function options",
			value:          []interface{}{},
			functionOpts:   nil,
			expectedResult: "schema function requires functionOptions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &types.Rule{
				Given: "$.methods",
				Then: &types.RuleAction{
					Function:        "schema",
					FunctionOptions: tt.functionOpts,
				},
			}

			schemaRule := &SchemaRule{}
			results := schemaRule.RunRule(tt.value, types.RuleFunctionContext{Rule: rule})

			if tt.expectedResult == "" {
				if len(results) != 0 {
					t.Fatalf("expected no results, got %+v", results)
				}
				return
			}

			if len(results) != 1 {
				t.Fatalf("expected one result, got %+v", results)
			}

			if !strings.HasPrefix(results[0].Message, tt.expectedResult) {
				t.Fatalf("expected message prefix %q, got %q", tt.expectedResult, results[0].Message)
			}
		})
	}
}

func TestSchemaCacheIndependentExecutions(t *testing.T) {
	action := &types.RuleAction{FunctionOptions: map[string]interface{}{"type": "string"}}
	type compiled struct {
		schema *jsonschema.Schema
		err    error
	}
	results := make(chan compiled, 16)
	for range cap(results) {
		go func() {
			rule := &SchemaRule{}
			schema, err := rule.schemaFor(action)
			if err != nil {
				results <- compiled{err: err}
				return
			}
			cached, err := rule.schemaFor(action)
			if cached != schema {
				results <- compiled{err: fmt.Errorf("execution did not reuse its schema")}
				return
			}
			results <- compiled{schema: cached, err: err}
		}()
	}
	first := <-results
	if first.err != nil || first.schema == nil {
		t.Fatalf("first compilation: %+v", first)
	}
	for range cap(results) - 1 {
		result := <-results
		if result.err != nil || result.schema == nil || result.schema == first.schema {
			t.Fatalf("executions shared a compiled schema: %+v", result)
		}
	}
}
