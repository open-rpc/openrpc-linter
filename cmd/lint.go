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

//lint:ignore chop Normalize and validate one severity enum while preserving accepted names and error text.
func normalizeSeverity(severity types.Severity) (types.Severity, error) {
	if severity == "" {
		return types.SeverityError, nil
	}

	switch types.Severity(strings.ToLower(string(severity))) {
	case types.SeverityIgnore:
		return types.SeverityIgnore, nil
	case types.SeverityError:
		return types.SeverityError, nil
	case types.SeverityWarn:
		return types.SeverityWarn, nil
	case types.SeverityInfo:
		return types.SeverityInfo, nil
	default:
		return "", fmt.Errorf("invalid severity %q; expected one of: error, warn, info, ignore", severity)
	}
}

// cyclo-allow(side_effect_density): This application boundary loads inputs, runs registered rules, enriches results, and writes the selected report.
//
//lint:ignore chop, thin_slice This application boundary loads inputs, runs registered rules, enriches results, and writes the selected report.
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
// cyclo-allow(side_effect_density): Read and decode the input once, reporting errors through the supplied command output.
//
//lint:ignore parameterize File reading and JSON decoding have different typed input contracts from resolved-document and schema preparation.
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

// cyclo-allow(side_effect_density): Build one resolved document and schema index; cross-package preparation calls are deliberate.
//
//lint:ignore parameterize In-memory resolution and schema preparation have different input contracts from file loading; shared guards do not unify them.
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

// cyclo-allow(side_effect_density): Load configured rules and report invalid input at the application boundary.
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

//lint:ignore thin_slice Bind one rule and normalize diagnostic IDs and severity before counting its errors.
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

// cyclo-allow(side_effect_density): This Cobra adapter reads CLI options, invokes linting, and requests the process exit on failure.
//
//lint:ignore parameterize Document linting has LintOptions and its own IO contract; the Cobra adapter preserves the existing exit behavior.
func runLintCommand(cmd *cobra.Command, args []string, exit func(int)) {
	openrpcFile := "openrpc.json"
	if len(args) > 0 {
		openrpcFile = args[0]
	}

	opts := LintOptions{
		OpenRPCFile: openrpcFile,
		RulesFile:   rulesFile,
		Output:      cmd.OutOrStdout(),
		Format:      outputFormat,
	}

	if err := RunLint(opts); err != nil {
		exit(1)
	}
}

// cyclo-allow(side_effect_density): Register lint flags and the subcommand once during package initialization.
func init() {
	lintCmd.Flags().StringVarP(&rulesFile, "rules", "r", "", "Path to rules YAML file")
	lintCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")
	rootCmd.AddCommand(lintCmd)
}

// resolveRefs resolves all $ref references in the document
// cyclo-allow(side_effect_density): JSON round-trip cloning handles unsupported input with explicit errors; encoding is the document boundary.
//
//lint:ignore parameterize JSON cloning wraps marshal errors and returns document values; its contract differs from schema-family fallback.
func resolveRefs(document interface{}) (interface{}, error) {
	// For now, we'll implement a basic $ref resolver that handles internal references
	// This is a simplified implementation that can be enhanced later

	docBytes, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	return resolveRefsJSON(docBytes, document)
}

// cyclo-allow(side_effect_density): Decode an owned document copy before reference traversal; decoder and recursive calls are intentional.
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
// cyclo-allow(side_effect_density): Traverse owned output containers and the active reference set; recursive call summaries remain conservative.
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
// cyclo-allow(side_effect_density): The active-chain set records entry and exit for cycle detection; recursive resolution requires both writes.
//
//lint:ignore chop, thin_slice The active-chain set records entry and exit for cycle detection; recursive resolution requires both writes.
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
