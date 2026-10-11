package functions

import "github.com/open-rpc/openrpc-linter/types"

func resultPath(path string) []string {
	if path == "" {
		return []string{}
	}
	return []string{path}
}

// ruleFailure creates one diagnostic with the same optional-path policy for all
// built-in rules. The executor supplies a target path when the path is absent.
func ruleFailure(message, path string) []types.RuleFunctionResult {
	return []types.RuleFunctionResult{{Message: message, Path: resultPath(path)}}
}

// functionOptions treats a missing rule or action as absent options.
func functionOptions(context types.RuleFunctionContext) map[string]any {
	if context.Rule == nil || context.Rule.Then == nil {
		return nil
	}
	return context.Rule.Then.FunctionOptions
}
