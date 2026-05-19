package reporters

import (
	"fmt"
	"io"

	"github.com/open-rpc/openrpc-linter/types"
)

type TextReporter struct{}

func (r *TextReporter) Format(results []types.RuleFunctionResult, totalRules int, output io.Writer) error {
	lintErrorCount := 0
	warnCount := 0
	infoCount := 0
	ruleErrorCount := 0
	ruleResults := make(map[string][]types.RuleFunctionResult)

	for _, result := range results {
		if result.Message == "" {
			continue
		}

		ruleResults[result.RuleID] = append(ruleResults[result.RuleID], result)
		if result.Kind == types.ResultKindRuleError {
			ruleErrorCount++
			continue
		}

		switch result.Severity {
		case types.SeverityWarn:
			warnCount++
		case types.SeverityInfo:
			infoCount++
		default:
			lintErrorCount++
		}
	}

	for ruleID, results := range ruleResults {
		for _, result := range results {
			icon := "❌"
			label := ""
			if result.Kind == types.ResultKindRuleError {
				label = "rule error"
			} else {
				switch result.Severity {
				case types.SeverityWarn:
					icon = "⚠️"
					label = "warning"
				case types.SeverityInfo:
					icon = "ℹ️"
					label = "info"
				}
			}

			if len(result.Path) == 0 {
				if label == "" {
					if _, err := fmt.Fprintf(output, "%s %s: %s\n", icon, ruleID, result.Message); err != nil {
						return err
					}
					continue
				}
				if _, err := fmt.Fprintf(output, "%s %s %s: %s\n", icon, label, ruleID, result.Message); err != nil {
					return err
				}
				continue
			}

			if label == "" {
				if _, err := fmt.Fprintf(output, "%s %s at %s: %s\n", icon, ruleID, result.Path[0], result.Message); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(output, "%s %s %s at %s: %s\n", icon, label, ruleID, result.Path[0], result.Message); err != nil {
				return err
			}
		}
	}

	totalFindings := lintErrorCount + warnCount + infoCount + ruleErrorCount
	if totalFindings == 0 {
		if _, err := fmt.Fprintf(output, "\n✅ All %d rules passed!\n", totalRules); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(
			output,
			"\n❌ %d error(s) found, ⚠️ %d warning(s), ℹ️ %d info(s), %d rule error(s) in %d rules\n",
			lintErrorCount,
			warnCount,
			infoCount,
			ruleErrorCount,
			len(ruleResults),
		); err != nil {
			return err
		}
	}

	return nil
}
