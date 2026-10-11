package rules

import (
	"fmt"

	"github.com/open-rpc/openrpc-linter/functions"
	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/open-rpc/openrpc-linter/types"

	"github.com/theory/jsonpath"
	"gopkg.in/yaml.v3"
)

// ExecuteRule runs a single rule against the document by:
//
//  1. parsing rule.Given into a JSONPath,
//  2. asking the selector for a uniform []Target (value mode for paths
//     that point at concrete nodes; field mode for paths whose terminal
//     segment is a field name; descendant-field mode for $..f / $.scope..f
//     using the schema-aware Index),
//  3. invoking the registered function once per Target.
//
// Every function — truthy, unique, schema, future ones — sees the same
// Target shape, so there is no per-function special-casing here.
//
//lint:ignore chop One rule owns a function instance, parsed path, and target stream; state cannot be shared between rule executions.
func ExecuteRule(rule *types.Rule, context types.RuleFunctionContext) ([]types.RuleFunctionResult, error) {
	if rule.Then == nil {
		return []types.RuleFunctionResult{}, nil
	}

	factory := functions.FunctionRegistry[rule.Then.Function]
	if factory == nil {
		return nil, fmt.Errorf("unknown function: %s", rule.Then.Function)
	}
	ruleFunc := factory()

	path, err := jsonpath.Parse(rule.Given)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON path: %w", err)
	}

	document := context.Document
	if context.ResolvedDocument != nil {
		document = context.ResolvedDocument
	}
	context.GivenPath = path

	targets := selector.Select(path, document, context.Index)

	var allResults []types.RuleFunctionResult
	for i := range targets {
		allResults = executeTarget(ruleFunc, &targets[i], context, allResults)
	}

	return allResults, nil
}

// executeTarget binds one selected node to the shared rule context and
// normalizes diagnostics before returning them to the rule executor.
// cyclo-allow(aggregate): Context and diagnostic values are transient DTOs, not independently identified domain aggregates.
func executeTarget(ruleFunc types.RuleFunction, target *selector.Target, context types.RuleFunctionContext, results []types.RuleFunctionResult) []types.RuleFunctionResult {
	context.Path = target.PathString()
	context.Target = target
	for _, result := range ruleFunc.RunRule(target.Node, context) {
		if result.Message == "" {
			continue
		}
		if len(result.Path) == 0 {
			result.Path = []string{target.PathString()}
		}
		results = append(results, result)
	}
	return results
}

func GetFieldFromNode(node *yaml.Node, field string) *yaml.Node {
	for i, n := range node.Content {
		if n.Value == field {
			return node.Content[i+1]
		}
	}
	return nil
}
