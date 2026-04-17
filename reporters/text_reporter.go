package reporters

import (
	"fmt"
	"io"

	"github.com/open-rpc/openrpc-linter/types"
)

type TextReporter struct{}

func (r *TextReporter) Format(results []types.RuleFunctionResult, totalRules int, output io.Writer) error {
	errorCount := 0
	ruleErrors := make(map[string][]string)

	for _, result := range results {
		if result.Message != "" {
			ruleErrors[result.RuleID] = append(ruleErrors[result.RuleID], result.Message)
			errorCount++
		}
	}

	for ruleID, messages := range ruleErrors {
		for _, message := range messages {
			if _, err := fmt.Fprintf(output, "❌ %s: %s\n", ruleID, message); err != nil {
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
