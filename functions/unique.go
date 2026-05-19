package functions

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/open-rpc/openrpc-linter/types"
)

type UniqueRule struct{}

func (r *UniqueRule) RunRule(value interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	if context.Rule == nil || context.Rule.Then == nil || context.Rule.Then.Field == "" {
		return []types.RuleFunctionResult{{
			Message: "unique function requires then.field",
			Path:    resultPath(context.Path),
		}}
	}
	fieldName := context.Rule.Then.Field

	items, ok := uniqueItems(value, context.Path)
	if !ok {
		return []types.RuleFunctionResult{{
			Message: "unique function requires array input",
			Path:    resultPath(context.Path),
		}}
	}

	ignoreMissing := true
	if context.Rule.Then.FunctionOptions != nil {
		rawIgnoreMissing, exists := context.Rule.Then.FunctionOptions["ignoreMissing"]
		if exists {
			parsed, ok := rawIgnoreMissing.(bool)
			if !ok {
				return []types.RuleFunctionResult{{
					Message: "unique function option ignoreMissing must be a boolean",
					Path:    resultPath(context.Path),
				}}
			}
			ignoreMissing = parsed
		}
	}

	seen := make(map[string]struct{})
	var results []types.RuleFunctionResult

	for _, item := range items {
		itemMap, ok := item.Value.(map[string]interface{})
		if !ok {
			results = append(results, types.RuleFunctionResult{
				Message: fmt.Sprintf("unique function requires object items to read field '%s'", fieldName),
				Path:    resultPath(item.Path),
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
				Path:    resultPath(fieldPath(item.Path, fieldName)),
			})
			continue
		}

		if _, found := seen[key]; found {
			results = append(results, types.RuleFunctionResult{
				Message: fmt.Sprintf("Duplicate value for field '%s': %s", fieldName, displayValue),
				Path:    resultPath(fieldPath(item.Path, fieldName)),
			})
			continue
		}

		seen[key] = struct{}{}
	}

	return results
}

type uniqueItem struct {
	Value interface{}
	Path  string
}

func uniqueItems(value interface{}, basePath string) ([]uniqueItem, bool) {
	switch v := value.(type) {
	case []interface{}:
		items := make([]uniqueItem, 0, len(v))
		for i, item := range v {
			items = append(items, uniqueItem{
				Value: item,
				Path:  fmt.Sprintf("%s[%d]", basePath, i),
			})
		}
		return items, true
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		items := make([]uniqueItem, 0, len(v))
		for _, key := range keys {
			items = append(items, uniqueItem{
				Value: v[key],
				Path:  basePath + pathNameSegment(key),
			})
		}
		return items, true
	default:
		return nil, false
	}
}

func resultPath(path string) []string {
	if path == "" {
		return []string{}
	}
	return []string{path}
}

func fieldPath(basePath string, fieldName string) string {
	if basePath == "" {
		return ""
	}
	return basePath + pathNameSegment(fieldName)
}

func pathNameSegment(name string) string {
	escaped := strings.ReplaceAll(name, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return "['" + escaped + "']"
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
