package rules

import (
	"fmt"

	"github.com/open-rpc/openrpc-linter/functions"
	"github.com/open-rpc/openrpc-linter/types"

	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
	"gopkg.in/yaml.v3"
)

func ExecuteRule(rule *types.Rule, context types.RuleFunctionContext) ([]types.RuleFunctionResult, error) {
	if rule.Then == nil {
		return []types.RuleFunctionResult{}, nil
	}

	ruleFunc := functions.FunctionRegistry[rule.Then.Function]
	if ruleFunc == nil {
		return nil, fmt.Errorf("unknown function: %s", rule.Then.Function)
	}

	path, err := jsonpath.Parse(rule.Given)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON path: %w", err)
	}

	document := context.Document
	if context.ResolvedDocument != nil {
		document = context.ResolvedDocument
	}

	var allResults []types.RuleFunctionResult
	for _, node := range path.SelectLocated(document) {
		valueToValidate := node.Node
		if rule.Then.Function != "unique" && rule.Then.Field != "" {
			if itemMap, ok := node.Node.(map[string]interface{}); ok {
				valueToValidate = itemMap[rule.Then.Field]
			}
		}

		itemContext := context
		if segs := node.Path; len(segs) > 0 {
			if idx, ok := segs[len(segs)-1].(spec.Index); ok {
				i := int(idx)
				itemContext.ArrayIndex = &i
			}
		}

		for _, result := range ruleFunc.RunRule(valueToValidate, itemContext) {
			if result.Message == "" {
				continue
			}
			if len(result.Path) == 0 {
				result.Path = []string{node.Path.String()}
			}
			allResults = append(allResults, result)
		}
	}

	return allResults, nil
}

func GetFieldFromNode(node *yaml.Node, field string) *yaml.Node {
	for i, n := range node.Content {
		if n.Value == field {
			return node.Content[i+1]
		}
	}
	return nil
}
