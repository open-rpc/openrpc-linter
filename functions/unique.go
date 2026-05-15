package functions

import (
	"fmt"
	"strconv"

	"github.com/open-rpc/openrpc-linter/types"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type UniqueRule struct{}

func (r *UniqueRule) RunRule(value interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	if context.Rule == nil || context.Rule.Then == nil || context.Rule.Then.Field == "" {
		return []types.RuleFunctionResult{{
			Message: "unique function requires then.field",
			Path:    []string{},
		}}
	}
	fieldName := context.Rule.Then.Field

	items, ok := value.([]interface{})
	if !ok {
		return []types.RuleFunctionResult{{
			Message: "unique function requires array input",
			Path:    []string{},
		}}
	}

	ignoreMissing := true
	if context.Rule.Then.FunctionOptions != nil {
		if rawIgnoreMissing, exists := context.Rule.Then.FunctionOptions["ignoreMissing"]; exists {
			if parsed, ok := rawIgnoreMissing.(bool); ok {
				ignoreMissing = parsed
			}
		}
	}

	seen := make(map[string]struct{})
	var results []types.RuleFunctionResult

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			results = append(results, types.RuleFunctionResult{
				Message: fmt.Sprintf("unique function requires object items to read field '%s'", fieldName),
				Path:    []string{},
			})
			continue
		}

		fieldValue, exists := itemMap[fieldName]
		if !exists && ignoreMissing {
			continue
		}
		if !exists {
			fieldValue = nil
		}

		key, displayValue, supported := comparableKey(fieldValue)
		if !supported {
			results = append(results, types.RuleFunctionResult{
				Message: fmt.Sprintf("unique function does not support non-primitive value for field '%s'", fieldName),
				Path:    []string{},
			})
			continue
		}

		if _, found := seen[key]; found {
			results = append(results, types.RuleFunctionResult{
				Message: fmt.Sprintf("Duplicate value for field '%s': %s", fieldName, displayValue),
				Path:    []string{},
			})
			continue
		}

		seen[key] = struct{}{}
	}

	return results
}

func (r *UniqueRule) GetSchema() *jsonschema.Schema {
	return &jsonschema.Schema{}
}

func comparableKey(value interface{}) (key string, displayValue string, supported bool) {
	switch v := value.(type) {
	case nil:
		return "null", "null", true
	case string:
		return "string:" + v, strconv.Quote(v), true
	case bool:
		if v {
			return "bool:true", "true", true
		}
		return "bool:false", "false", true
	case float64:
		return fmt.Sprintf("number:%g", v), fmt.Sprintf("%g", v), true
	default:
		return "", "", false
	}
}
