package rules

import (
	"fmt"
	"strings"

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
	if rule.Then.Function == "unique" {
		return executeUniqueRule(rule, context, document, ruleFunc)
	}

	for _, node := range path.SelectLocated(document) {
		valueToValidate := node.Node
		if rule.Then.Field != "" {
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

func executeUniqueRule(
	rule *types.Rule,
	context types.RuleFunctionContext,
	document interface{},
	ruleFunc types.RuleFunction,
) ([]types.RuleFunctionResult, error) {
	collections, err := getUniqueCollections(rule.Given, document)
	if err != nil {
		return nil, err
	}

	var allResults []types.RuleFunctionResult
	for _, collection := range collections {
		results := ruleFunc.RunRule(collection.Items, context)
		for _, result := range results {
			if result.Message != "" {
				if len(result.Path) == 0 {
					result.Path = []string{collection.Path}
				}
				allResults = append(allResults, result)
			}
		}
	}

	return allResults, nil
}

type uniqueCollection struct {
	Items []interface{}
	Path  string
}

func getUniqueCollections(given string, document interface{}) ([]uniqueCollection, error) {
	parentPath := strings.TrimSpace(given)
	if strings.HasSuffix(parentPath, "[*]") {
		parentPath = strings.TrimSuffix(parentPath, "[*]")
	}

	path, err := jsonpath.Parse(parentPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing unique parent JSON path: %w", err)
	}

	var collections []uniqueCollection
	for _, node := range path.SelectLocated(document) {
		switch v := node.Node.(type) {
		case []interface{}:
			collections = append(collections, uniqueCollection{
				Items: v,
				Path:  node.Path.String(),
			})
		case map[string]interface{}:
			collection := make([]interface{}, 0, len(v))
			for _, item := range v {
				collection = append(collection, item)
			}
			collections = append(collections, uniqueCollection{
				Items: collection,
				Path:  node.Path.String(),
			})
		default:
			return nil, fmt.Errorf("unique function requires array-like JSONPath selection")
		}
	}

	return collections, nil
}

func GetFieldFromNode(node *yaml.Node, field string) *yaml.Node {
	for i, n := range node.Content {
		if n.Value == field {
			return node.Content[i+1]
		}
	}
	return nil
}
