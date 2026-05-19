package functions

import (
	"reflect"
	"testing"

	"github.com/open-rpc/openrpc-linter/types"
)

func TestUniqueRule(t *testing.T) {
	tests := []struct {
		name            string
		value           interface{}
		field           string
		functionOptions map[string]interface{}
		expected        []string
	}{
		{
			name:  "reports duplicate strings with quoted display value",
			field: "name",
			value: []interface{}{
				map[string]interface{}{"name": "ping"},
				map[string]interface{}{"name": "ping"},
			},
			expected: []string{`Duplicate value for field 'name': "ping"`},
		},
		{
			name:  "keeps primitive types distinct",
			field: "value",
			value: []interface{}{
				map[string]interface{}{"value": "1"},
				map[string]interface{}{"value": float64(1)},
				map[string]interface{}{"value": true},
				map[string]interface{}{"value": false},
				map[string]interface{}{"value": nil},
			},
		},
		{
			name:  "reports duplicate booleans",
			field: "deprecated",
			value: []interface{}{
				map[string]interface{}{"deprecated": true},
				map[string]interface{}{"deprecated": true},
			},
			expected: []string{"Duplicate value for field 'deprecated': true"},
		},
		{
			name:  "ignores missing fields by default",
			field: "summary",
			value: []interface{}{
				map[string]interface{}{"name": "first"},
				map[string]interface{}{"name": "second"},
			},
		},
		{
			name:            "treats missing fields as null when ignoreMissing is false",
			field:           "summary",
			functionOptions: map[string]interface{}{"ignoreMissing": false},
			value: []interface{}{
				map[string]interface{}{"name": "first"},
				map[string]interface{}{"name": "second"},
			},
			expected: []string{"Duplicate value for field 'summary': null"},
		},
		{
			name:            "rejects non boolean ignoreMissing option",
			field:           "summary",
			functionOptions: map[string]interface{}{"ignoreMissing": "false"},
			value: []interface{}{
				map[string]interface{}{"name": "first"},
			},
			expected: []string{"unique function option ignoreMissing must be a boolean"},
		},
		{
			name:     "requires then field",
			value:    []interface{}{},
			expected: []string{"unique function requires then.field"},
		},
		{
			name:     "requires array input",
			field:    "name",
			value:    "ping",
			expected: []string{"unique function requires array input"},
		},
		{
			name:  "requires object items",
			field: "name",
			value: []interface{}{"ping"},
			expected: []string{
				"unique function requires object items to read field 'name'",
			},
		},
		{
			name:  "rejects non primitive field values",
			field: "schema",
			value: []interface{}{
				map[string]interface{}{"schema": map[string]interface{}{"type": "string"}},
			},
			expected: []string{"unique function does not support non-primitive value for field 'schema'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &types.Rule{
				Then: &types.RuleAction{
					Field:           tt.field,
					Function:        "unique",
					FunctionOptions: tt.functionOptions,
				},
			}
			results := (&UniqueRule{}).RunRule(tt.value, types.RuleFunctionContext{Rule: rule})
			got := resultMessages(results)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("expected messages %+v, got %+v", tt.expected, got)
			}
		})
	}
}

func resultMessages(results []types.RuleFunctionResult) []string {
	if len(results) == 0 {
		return nil
	}

	messages := make([]string, 0, len(results))
	for _, result := range results {
		messages = append(messages, result.Message)
	}
	return messages
}
