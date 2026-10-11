package functions

import (
	"fmt"
	"sync"

	"github.com/open-rpc/openrpc-linter/types"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type SchemaRule struct {
	mu    sync.Mutex
	cache map[*types.RuleAction]*jsonschema.Schema
}

//lint:ignore chop, thin_slice Missing fields skip schema validation; option and conformance errors preserve distinct ordered diagnostics.
func (r *SchemaRule) RunRule(value interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	// Same reasoning as UniqueRule: a missing-field Target has nothing for
	// schema validation to chew on. Skip and let truthy report absence.
	if t := context.Target; t != nil && t.Field != "" && !t.Exists {
		return nil
	}

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
	r.mu.Lock()
	defer r.mu.Unlock()
	if cached, ok := r.cache[then]; ok {
		return cached, nil
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("rule-schema.json", then.FunctionOptions); err != nil {
		return nil, err
	}

	schema, err := compiler.Compile("rule-schema.json")
	if err != nil {
		return nil, err
	}

	if r.cache == nil {
		r.cache = make(map[*types.RuleAction]*jsonschema.Schema)
	}
	r.cache[then] = schema
	return schema, nil
}
