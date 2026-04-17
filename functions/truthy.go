package functions

import (
	"fmt"
	"strings"

	"github.com/open-rpc/openrpc-linter/types"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type TruthyRule struct{}

func (r *TruthyRule) RunRule(value interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	var results []types.RuleFunctionResult

	isTruthy := true

	if value == nil {
		isTruthy = false
	} else if str, ok := value.(string); ok && str == "" {
		isTruthy = false
	} else if str, ok := value.(string); ok && str == "null" {
		isTruthy = false
	}

	if !isTruthy {
		var message string
		if context.Rule != nil && context.Rule.Then != nil && context.Rule.Then.Field != "" {
			fieldName := context.Rule.Then.Field
			jsonPath := context.Rule.Given

			if context.ArrayIndex != nil {
				jsonPath = strings.Replace(jsonPath, "[*]", fmt.Sprintf("[%d]", *context.ArrayIndex), 1)
			}

			message = "Missing required field '" + fieldName + "' at " + jsonPath
		} else {
			message = "Field must have a truthy value"
		}

		results = append(results, types.RuleFunctionResult{
			Message: message,
			Path:    []string{},
		})
	}

	return results
}

func (r *TruthyRule) GetSchema() *jsonschema.Schema {
	return &jsonschema.Schema{}
}
