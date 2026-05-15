package reporters

import (
	"fmt"
	"io"

	"github.com/open-rpc/openrpc-linter/types"
)

type TextReporter struct{}

func (r *TextReporter) Format(results []types.RuleFunctionResult, totalRules int, output io.Writer) error {
	errorCount := 0
	// keep the whole result, not just the message, so we can print the path too
	ruleErrors := make(map[string][]types.RuleFunctionResult)

	for _, result := range results {
		if result.Message != "" {
			ruleErrors[result.RuleID] = append(ruleErrors[result.RuleID], result)
			errorCount++
		}
	}

	for ruleID, results := range ruleErrors {
		for _, result := range results {
			if len(result.Path) == 0 {
				if _, err := fmt.Fprintf(output, "❌ %s: %s\n", ruleID, result.Message); err != nil {
					return err
				}
				continue
			}

			if _, err := fmt.Fprintf(output, "❌ %s at %s: %s\n", ruleID, result.Path[0], result.Message); err != nil {
				return err
			}
		}
	}

	if errorCount == 0 {
		if _, err := fmt.Fprintf(output, "\n✅ All %d rules passed!\n", totalRules); err != nil {
			return err
		}
	} else {
		rulesWithErrors := len(ruleErrors)
		if _, err := fmt.Fprintf(output, "\n❌ %d error(s) found in %d rules\n", errorCount, rulesWithErrors); err != nil {
			return err
		}
	}

	return nil
}
