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

// cyclo-allow(side_effect_density): This explicit registration operation installs the three built-in factories in the public registry.
func RegisterFunctions() {
	for name, factory := range builtinFunctions() {
		FunctionRegistry[name] = factory
	}
}
