package functions

import (
	"github.com/open-rpc/openrpc-linter/types"
)

type TruthyRule struct{}

// RunRule evaluates one Target. The selector already decided what the
// target is — value mode (Field == "") or field mode (Field != "") — so
// truthy only has to answer "is this present and truthy?" and emit the
// right diagnostic. No JSONPath introspection lives here anymore.
func (r *TruthyRule) RunRule(value interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	path := context.Path
	t := context.Target
	if t != nil {
		path = t.PathString()
		if missingField(t) {
			return ruleFailure("missing field '"+t.Field+"'", path)
		}
	}

	if !truthyValue(value) {
		return ruleFailure("Field must have a truthy value", path)
	}

	return nil
}

func truthyValue(value interface{}) bool {
	if value == nil {
		return false
	}
	if str, ok := value.(string); ok && (str == "" || str == "null") {
		return false
	}
	return true
}
