package reporters

import (
	"encoding/json"
	"io"

	"github.com/open-rpc/openrpc-linter/types"
)

type JSONReporter struct{}

func (r *JSONReporter) Format(results []types.RuleFunctionResult, totalRules int, output io.Writer) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}
