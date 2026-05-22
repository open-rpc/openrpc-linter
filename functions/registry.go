package functions

import "github.com/open-rpc/openrpc-linter/types"

var FunctionRegistry = map[string]func() types.RuleFunction{}

func init() {
	RegisterFunctions()
}

func RegisterFunctions() {
	FunctionRegistry["truthy"] = func() types.RuleFunction { return &TruthyRule{} }
	FunctionRegistry["schema"] = func() types.RuleFunction { return &SchemaRule{} }
	FunctionRegistry["unique"] = func() types.RuleFunction { return NewUniqueRule() }
	FunctionRegistry["referenced"] = func() types.RuleFunction { return NewReferencedRule() }
}
