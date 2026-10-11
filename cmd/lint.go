package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/open-rpc/openrpc-linter/location"
	"github.com/open-rpc/openrpc-linter/metaschema"
	"github.com/open-rpc/openrpc-linter/reporters"
	"github.com/open-rpc/openrpc-linter/rules"
	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/open-rpc/openrpc-linter/types"

	"github.com/spf13/cobra"
)

var (
	rulesFile    string
	outputFormat string
)

type LintOptions struct {
	OpenRPCFile string
	RulesFile   string
	Output      io.Writer
	Format      string
}

func GetReporter(format string) reporters.Reporter {
	switch format {
	case "json":
		return &reporters.JSONReporter{}
	case "text":
		return &reporters.TextReporter{}
	default:
		return &reporters.TextReporter{}
	}
}

func normalizeSeverity(severity types.Severity) (types.Severity, error) {
	if severity == "" {
		return types.SeverityError, nil
	}

	normalized := types.Severity(strings.ToLower(string(severity)))
	switch normalized {
	case types.SeverityIgnore, types.SeverityError, types.SeverityWarn, types.SeverityInfo:
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid severity %q; expected one of: error, warn, info, ignore", severity)
	}
}

func RunLint(opts LintOptions) error {
	context, err := loadLintDocument(opts.OpenRPCFile, opts.Output)
	if err != nil {
		return err
	}
	ruleSet, err := loadLintRules(opts.RulesFile, opts.Output)
	if err != nil {
		return err
	}
	results, errorCount, err := evaluateLintRules(ruleSet, context, opts.Output)
	if err != nil {
		return err
	}
	location.Enrich(results, context.ResolvedDocument)

	reporter := GetReporter(opts.Format)
	if tr, ok := reporter.(*reporters.TextReporter); ok {
		tr.SourceFile = opts.OpenRPCFile
	}
	if err := reporter.Format(results, len(ruleSet), opts.Output); err != nil {
		return err
	}
	if errorCount > 0 {
		return fmt.Errorf("found %d linting error(s)", errorCount)
	}
	return nil
}

// loadLintDocument prepares the shared document and index once per lint run.
func loadLintDocument(path string, output io.Writer) (types.RuleFunctionContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(output, "Error reading OpenRPC file: %v\n", err)
		return types.RuleFunctionContext{}, err
	}
	var document interface{}
	if err := json.Unmarshal(data, &document); err != nil {
		fmt.Fprintf(output, "Error parsing OpenRPC file: %v\n", err)
		return types.RuleFunctionContext{}, err
	}
	return prepareLintDocument(document, output)
}

func prepareLintDocument(document interface{}, output io.Writer) (types.RuleFunctionContext, error) {
	resolved, err := resolveRefs(document)
	if err != nil {
		fmt.Fprintf(output, "Error resolving $refs in OpenRPC file: %v\n", err)
		return types.RuleFunctionContext{}, err
	}
	meta, err := metaschema.For(document)
	if err != nil {
		fmt.Fprintf(output, "Error selecting OpenRPC meta-schema: %v\n", err)
		return types.RuleFunctionContext{}, err
	}
	return types.RuleFunctionContext{
		Document:         document,
		ResolvedDocument: resolved,
		Index:            selector.Build(resolved, meta),
	}, nil
}

func loadLintRules(path string, output io.Writer) (map[string]types.Rule, error) {
	wrapper, err := rules.LoadRulesFileFromPath(path)
	if err != nil {
		return nil, err
	}
	if err := wrapper.CheckRules(); err != nil {
		fmt.Fprintf(output, "Error checking rules file: %v\n", err)
		return nil, err
	}
	return wrapper.ResolvedRules()
}

func evaluateLintRules(ruleSet map[string]types.Rule, context types.RuleFunctionContext, output io.Writer) ([]types.RuleFunctionResult, int, error) {
	var allResults []types.RuleFunctionResult
	errorCount := 0
	for ruleID, rule := range ruleSet {
		severity, err := normalizeSeverity(rule.Severity)
		if err != nil {
			fmt.Fprintf(output, "Error validating rules file: rule %q %v\n", ruleID, err)
			return nil, 0, err
		}
		if severity == types.SeverityIgnore {
			continue
		}
		rule.Severity = severity
		results, count := evaluateLintRule(ruleID, rule, context)
		allResults = append(allResults, results...)
		errorCount += count
	}
	return allResults, errorCount, nil
}

func evaluateLintRule(ruleID string, rule types.Rule, context types.RuleFunctionContext) ([]types.RuleFunctionResult, int) {
	context.Rule = &rule
	context.RuleID = ruleID
	results, err := rules.ExecuteRule(&rule, context)
	if err != nil {
		return []types.RuleFunctionResult{{
			RuleID:   ruleID,
			Message:  err.Error(),
			Severity: types.SeverityError,
		}}, 1
	}
	errorCount := 0
	for i := range results {
		if results[i].RuleID == "" {
			results[i].RuleID = ruleID
		}
		results[i].Severity = rule.Severity
		if rule.Severity == types.SeverityError {
			errorCount++
		}
	}
	return results, errorCount
}

var lintCmd = &cobra.Command{
	Use:   "lint [openrpc-file]",
	Short: "Lint an OpenRPC document",
	Long:  "Lint an OpenRPC document for compliance with OpenRPC specification",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runLintCommand(cmd, args, os.Exit)
	},
}

func runLintCommand(cmd *cobra.Command, args []string, exit func(int)) {
	opts := LintOptions{
		OpenRPCFile: commandArgument(args, "openrpc.json"),
		RulesFile:   rulesFile,
		Output:      cmd.OutOrStdout(),
		Format:      outputFormat,
	}

	execute(func() error { return RunLint(opts) }, exit)
}

func init() {
	lintCmd.Flags().StringVarP(&rulesFile, "rules", "r", "", "Path to rules YAML file")
	lintCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")
	rootCmd.AddCommand(lintCmd)
}

// resolveRefs resolves all $ref references in the document
func resolveRefs(document interface{}) (interface{}, error) {
	// For now, we'll implement a basic $ref resolver that handles internal references
	// This is a simplified implementation that can be enhanced later

	docBytes, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	return resolveRefsJSON(docBytes, document)
}

func resolveRefsJSON(docBytes []byte, document interface{}) (interface{}, error) {
	var resolved interface{}
	err := json.Unmarshal(docBytes, &resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// Recursively resolve $refs within the document
	resolved = resolveRefsRecursive(resolved, document, types.ResolvingRefs{})

	return resolved, nil
}

// resolveRefsRecursive recursively resolves $ref references in the document
func resolveRefsRecursive(current interface{}, root interface{}, resolving types.ResolvingRefs) interface{} {
	switch v := current.(type) {
	case map[string]interface{}:
		if _, exists := v["$ref"]; exists {
			return resolveReference(v, root, resolving)
		}

		// Recursively process all values in the map
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = resolveRefsRecursive(value, root, resolving)
		}
		return result

	case []interface{}:
		// Recursively process all items in the array
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = resolveRefsRecursive(item, root, resolving)
		}
		return result

	default:
		// For primitive types, return as-is
		return v
	}
}

// resolveReference follows a local reference while tracking the active chain.
// Unresolved and circular references retain their original object.
func resolveReference(node map[string]interface{}, root interface{}, resolving types.ResolvingRefs) interface{} {
	ref, ok := node["$ref"].(string)
	if !ok || !strings.HasPrefix(ref, "#/") || resolving[ref] {
		return node
	}
	target := resolveJSONPointer(ref[2:], root)
	if target == nil {
		return node
	}
	resolving[ref] = true
	result := resolveRefsRecursive(target, root, resolving)
	delete(resolving, ref)
	return result
}

// resolveJSONPointer resolves a JSON pointer path in the document
func resolveJSONPointer(path string, document interface{}) interface{} {
	if path == "" {
		return document
	}

	parts := strings.Split(path, "/")
	current := document

	for _, part := range parts {
		// Unescape JSON pointer characters
		part = strings.ReplaceAll(part, "~1", "/")
		part = strings.ReplaceAll(part, "~0", "~")

		switch v := current.(type) {
		case map[string]interface{}:
			if val, exists := v[part]; exists {
				current = val
			} else {
				return nil // Path not found
			}
		case []interface{}:
			// Handle array indices (not commonly used in OpenRPC, but for completeness)
			return nil
		default:
			return nil // Can't traverse further
		}
	}

	return current
}
