package functions

import (
	"fmt"

	"github.com/open-rpc/openrpc-linter/types"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaRule owns the cache for one execution. ExecuteRule creates an instance
// per rule and invokes its targets serially; instances must not be shared.
type SchemaRule struct {
	cache map[*types.RuleAction]*jsonschema.Schema
}

func (r *SchemaRule) RunRule(value interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	// Same reasoning as UniqueRule: a missing-field Target has nothing for
	// schema validation to chew on. Skip and let truthy report absence.
	if missingField(context.Target) {
		return nil
	}

	then := context.Rule.Then

	if len(then.FunctionOptions) == 0 {
		return ruleFailure("schema function requires functionOptions", "")
	}

	schema, err := r.schemaFor(then)
	if err != nil {
		return ruleFailure(fmt.Sprintf("Invalid schema functionOptions: %v", err), "")
	}

	err = schema.Validate(value)
	if err != nil {
		return ruleFailure(fmt.Sprintf("Value does not match schema: %v", err), "")
	}

	return nil
}

// schemaFor caches one compiled schema per rule action so iterating over N
// matched nodes doesn't recompile N times.
func (r *SchemaRule) schemaFor(then *types.RuleAction) (*jsonschema.Schema, error) {
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
