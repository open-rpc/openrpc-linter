package reporters

import (
	"fmt"
	"io"
	"strings"

	"github.com/open-rpc/openrpc-linter/types"
)

type TextReporter struct{}

func (r *TextReporter) Format(results []types.RuleFunctionResult, totalRules int, output io.Writer) error {
	errorCount := 0
	warnCount := 0
	infoCount := 0

	for _, result := range results {
		if result.Message != "" {
			switch result.Severity {
			case types.SeverityWarn:
				warnCount++
			case types.SeverityInfo:
				infoCount++
			default:
				errorCount++
			}

			ruleID := result.RuleID
			if ruleID == "" {
				ruleID = "-"
			}

			severity := result.Severity
			if severity == "" {
				severity = types.SeverityError
			}

			icon := "❌"
			switch severity {
			case types.SeverityWarn:
				icon = "⚠️"
			case types.SeverityInfo:
				icon = "ℹ️"
			}

			if _, err := fmt.Fprintf(
				output,
				"%s [%s] %s: %s\n",
				icon,
				strings.ToUpper(string(severity)),
				ruleID,
				result.Message,
			); err != nil {
				return err
			}
		}
	}

	totalFindings := errorCount + warnCount + infoCount
	if totalFindings == 0 {
		if _, err := fmt.Fprintf(output, "\n✅ All %d rules passed!\n", totalRules); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(
			output,
			"\nSummary: %d error(s), %d warning(s), %d info finding(s)\n",
			errorCount,
			warnCount,
			infoCount,
		); err != nil {
			return err
		}
	}

	return nil
}
