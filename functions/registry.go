package functions

import "github.com/open-rpc/openrpc-linter/types"

var FunctionRegistry = builtinFunctions()

func builtinFunctions() map[string]func() types.RuleFunction {
	return map[string]func() types.RuleFunction{
		"truthy": func() types.RuleFunction { return &TruthyRule{} },
		"schema": func() types.RuleFunction { return &SchemaRule{} },
		"unique": func() types.RuleFunction { return NewUniqueRule() },
	}
}

func RegisterFunctions() {
	for name, factory := range builtinFunctions() {
		FunctionRegistry[name] = factory
	}
}
