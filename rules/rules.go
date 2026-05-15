package rules

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/open-rpc/openrpc-linter/functions"
	"github.com/open-rpc/openrpc-linter/types"

	"github.com/theory/jsonpath"
	"gopkg.in/yaml.v3"
)

var normalizedArrayIndexPattern = regexp.MustCompile(`\[(\d+)\]$`)

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
		if rule.Then.Field != "" {
			if itemMap, ok := node.Node.(map[string]interface{}); ok {
				valueToValidate = itemMap[rule.Then.Field]
			}
		}

		itemContext := context
		if arrayIndex, ok := arrayIndexFromNormalizedPath(node.Path.String()); ok {
			itemContext.ArrayIndex = &arrayIndex
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

func arrayIndexFromNormalizedPath(path string) (int, bool) {
	matches := normalizedArrayIndexPattern.FindStringSubmatch(path)
	if len(matches) != 2 {
		return 0, false
	}

	index, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}

	return index, true
}

func GetFieldFromNode(node *yaml.Node, field string) *yaml.Node {
	for i, n := range node.Content {
		if n.Value == field {
			return node.Content[i+1]
		}
	}
	return nil
}
