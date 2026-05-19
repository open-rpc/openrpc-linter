package functions

import (
	"fmt"
	"sync"

	"github.com/open-rpc/openrpc-linter/types"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type SchemaRule struct {
	cache sync.Map // keyed by *types.RuleAction
}

func (r *SchemaRule) RunRule(value interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	then := context.Rule.Then

	if len(then.FunctionOptions) == 0 {
		return []types.RuleFunctionResult{{Message: "schema function requires functionOptions"}}
	}

	schema, err := r.schemaFor(then)
	if err != nil {
		return []types.RuleFunctionResult{{Message: fmt.Sprintf("Invalid schema functionOptions: %v", err)}}
	}

	err = schema.Validate(value)
	if err != nil {
		return []types.RuleFunctionResult{{Message: fmt.Sprintf("Value does not match schema: %v", err)}}
	}

	return nil
}

// schemaFor caches one compiled schema per rule action so iterating over N
// matched nodes doesn't recompile N times.
func (r *SchemaRule) schemaFor(then *types.RuleAction) (*jsonschema.Schema, error) {
	if cached, ok := r.cache.Load(then); ok {
		return cached.(*jsonschema.Schema), nil
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("rule-schema.json", then.FunctionOptions); err != nil {
		return nil, err
	}

	schema, err := compiler.Compile("rule-schema.json")
	if err != nil {
		return nil, err
	}

	actual, _ := r.cache.LoadOrStore(then, schema)
	return actual.(*jsonschema.Schema), nil
}
