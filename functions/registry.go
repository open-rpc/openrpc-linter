package functions

import "github.com/open-rpc/openrpc-linter/types"

var FunctionRegistry = make(map[string]types.RuleFunction)

func init() {
	RegisterFunctions()
}

func RegisterFunctions() {
	FunctionRegistry["truthy"] = &TruthyRule{}
}
