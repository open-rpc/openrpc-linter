package rules

import (
	"strings"
	"testing"

	"github.com/open-rpc/openrpc-linter/types"

	"gopkg.in/yaml.v3"
)

func TestExecuteRule(t *testing.T) {
	tests := []struct {
		name        string
		rule        *types.Rule
		document    interface{}
		context     types.RuleFunctionContext
		expectError bool
		expectedMsg string
	}{
		{
			name: "truthy rule with missing field",
			rule: &types.Rule{
				Description: "Test missing field",
				Given:       "$.info",
				Then: &types.RuleAction{
					Field:    "description",
					Function: "truthy",
				},
			},
			document: map[string]interface{}{
				"info": map[string]interface{}{
					"title":   "Test API",
					"version": "1.0.0",
					// no description field
				},
			},
			expectError: true,
			expectedMsg: "Missing required field 'description' at $.info",
		},
		{
			name: "truthy rule with present field",
			rule: &types.Rule{
				Description: "Test present field",
				Given:       "$.info",
				Then: &types.RuleAction{
					Field:    "description",
					Function: "truthy",
				},
			},
			document: map[string]interface{}{
				"info": map[string]interface{}{
					"title":       "Test API",
					"version":     "1.0.0",
					"description": "A test API",
				},
			},
			expectError: false,
		},
		{
			name: "unknown function",
			rule: &types.Rule{
				Description: "Test unknown function",
				Given:       "$.info",
				Then: &types.RuleAction{
					Field:    "title",
					Function: "unknownFunction",
				},
			},
			document: map[string]interface{}{
				"info": map[string]interface{}{
					"title": "Test API",
				},
			},
			expectError: true,
			expectedMsg: "unknown function: unknownFunction",
		},
		{
			name: "invalid jsonpath",
			rule: &types.Rule{
				Description: "Test invalid path",
				Given:       "$.nonexistent",
				Then: &types.RuleAction{
					Field:    "title",
					Function: "truthy",
				},
			},
			document: map[string]interface{}{
				"info": map[string]interface{}{
					"title": "Test API",
				},
			},
			expectError: true, // Should expect error for invalid jsonpath
			expectedMsg: "error getting JSON path: unknown key nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := types.RuleFunctionContext{
				Rule:     tt.rule,
				RuleID:   "test-rule",
				Document: tt.document,
			}
			tt.context = context

			results, err := ExecuteRule(tt.rule, tt.context)

			if tt.expectError {
				if err == nil && (len(results) == 0 || (len(results) > 0 && (results[0].Message == "" || results[0].Message == "Result: <nil>"))) {
					t.Errorf("Expected error message, but got results: %+v, err: %+v", results, err)
				}
				if tt.expectedMsg != "" && err != nil && err.Error() != tt.expectedMsg {
					t.Errorf("Expected error message %q, got %q", tt.expectedMsg, err.Error())
				}
				if tt.expectedMsg != "" && err == nil && len(results) > 0 && results[0].Message != tt.expectedMsg {
					t.Errorf("Expected result message %q, got %q", tt.expectedMsg, results[0].Message)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success, but got error: %v", err)
				}
				// For success cases, we expect either no results or results with "Result:" messages
				for _, result := range results {
					if result.Message != "" && !strings.HasPrefix(result.Message, "Result:") {
						t.Errorf("Expected success (empty or Result: message), but got: %q", result.Message)
					}
				}
			}
		})
	}
}

func TestGetFieldFromNode(t *testing.T) {
	tests := []struct {
		name     string
		node     *yaml.Node
		field    string
		expected *yaml.Node
	}{
		{
			name: "field found",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key1"},
					{Kind: yaml.ScalarNode, Value: "value1"},
					{Kind: yaml.ScalarNode, Value: "key2"},
					{Kind: yaml.ScalarNode, Value: "value2"},
				},
			},
			field:    "key2",
			expected: &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"},
		},
		{
			name: "field not found",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key1"},
					{Kind: yaml.ScalarNode, Value: "value1"},
				},
			},
			field:    "nonexistent",
			expected: nil,
		},
		{
			name: "empty node",
			node: &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{},
			},
			field:    "anykey",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFieldFromNode(tt.node, tt.field)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("GetFieldFromNode() returned %v, expected nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("GetFieldFromNode() returned nil, expected %v", tt.expected)
				} else if result.Value != tt.expected.Value {
					t.Errorf("GetFieldFromNode() returned value %q, expected %q", result.Value, tt.expected.Value)
				}
			}
		})
	}
}

// Benchmark test for ExecuteRule
func BenchmarkExecuteRule(b *testing.B) {
	rule := &types.Rule{
		Description: "Benchmark rule",
		Given:       "$.info",
		Then: &types.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
	}

	document := map[string]interface{}{
		"info": map[string]interface{}{
			"title":   "Test API",
			"version": "1.0.0",
		},
	}

	context := types.RuleFunctionContext{
		Rule:     rule,
		RuleID:   "benchmark-rule",
		Document: document,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteRule(rule, context)
	}
}
