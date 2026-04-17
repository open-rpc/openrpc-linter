package rules

import (
	"fmt"

	"github.com/open-rpc/openrpc-linter/functions"
	"github.com/open-rpc/openrpc-linter/types"

	"github.com/PaesslerAG/jsonpath"
	"gopkg.in/yaml.v3"
)

func ExecuteRule(rule *types.Rule, context types.RuleFunctionContext) ([]types.RuleFunctionResult, error) {

	documentToUse := context.Document
	if context.ResolvedDocument != nil {
		documentToUse = context.ResolvedDocument
	}

	res, err := jsonpath.Get(rule.Given, documentToUse)
	if err != nil {
		return nil, fmt.Errorf("error getting JSON path: %w", err)
	}

	if rule.Then != nil {
		ruleFunc := functions.FunctionRegistry[rule.Then.Function]
		if ruleFunc == nil {
			return nil, fmt.Errorf("unknown function: %s", rule.Then.Function)
		}

		if resArray, ok := res.([]interface{}); ok {
			var allResults []types.RuleFunctionResult

			for i, item := range resArray {
				var valueToValidate interface{}

				if rule.Then.Field != "" {
					if itemMap, ok := item.(map[string]interface{}); ok {
						valueToValidate = itemMap[rule.Then.Field]
					}
				} else {
					valueToValidate = item
				}

				itemContext := context
				itemContext.ArrayIndex = &i

				results := ruleFunc.RunRule(valueToValidate, itemContext)

				for _, result := range results {
					if result.Message != "" {
						allResults = append(allResults, result)
					}
				}
			}

			return allResults, nil
		} else {
			var valueToValidate interface{}

			if rule.Then.Field != "" {
				if resMap, ok := res.(map[string]interface{}); ok {
					valueToValidate = resMap[rule.Then.Field]
				}
			} else {
				valueToValidate = res
			}

			results := ruleFunc.RunRule(valueToValidate, context)

			return results, nil
		}
	}

	return []types.RuleFunctionResult{}, nil
}

func GetFieldFromNode(node *yaml.Node, field string) *yaml.Node {
	for i, n := range node.Content {
		if n.Value == field {
			return node.Content[i+1]
		}
	}
	return nil
}
