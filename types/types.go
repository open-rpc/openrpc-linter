package types

import (
	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/theory/jsonpath"
)

type Severity string

const (
	SeverityError  Severity = "error"
	SeverityWarn   Severity = "warn"
	SeverityInfo   Severity = "info"
	SeverityIgnore Severity = "ignore"
)

type RuleDefaults string

const (
	RuleExtensionRecommended RuleDefaults = "recommended"
)

type ResolvingRefs = map[string]bool

type Rule struct {
	Description string      `json:"description" yaml:"description"`
	Given       string      `json:"given,omitempty" yaml:"given,omitempty"`
	Then        *RuleAction `json:"then,omitempty" yaml:"then,omitempty"`
	Extends     interface{} `json:"extends,omitempty" yaml:"extends,omitempty"`
	Severity    Severity    `json:"severity,omitempty" yaml:"severity,omitempty"`
}

type RuleAction struct {
	Field           string                 `json:"field,omitempty" yaml:"field,omitempty"`
	Function        string                 `json:"function,omitempty" yaml:"function,omitempty"`
	FunctionOptions map[string]interface{} `json:"functionOptions,omitempty" yaml:"functionOptions,omitempty"`
}

// PathLabels holds human-readable anchors for a violation path in the document.
type PathLabels struct {
	Method     string `json:"method,omitempty"`
	Param      string `json:"param,omitempty"`
	Schema     string `json:"schema,omitempty"`
	Descriptor string `json:"descriptor,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Section    string `json:"section,omitempty"`
}

func (l PathLabels) IsEmpty() bool {
	return l == (PathLabels{})
}

type RuleFunctionResult struct {
	Message    string     `json:"message,omitempty"`
	Path       []string   `json:"path,omitempty"`
	PathLabels PathLabels `json:"pathLabels,omitempty"`
	RuleID     string     `json:"ruleId,omitempty"`
	Severity   Severity   `json:"severity,omitempty" yaml:"severity,omitempty"`
}

type RuleFunctionContext struct {
	Rule             *Rule          `json:"rule"`
	RuleID           string         `json:"ruleId"`
	Document         interface{}    `json:"document"`         // Original document with potential $refs
	ResolvedDocument interface{}    `json:"resolvedDocument"` // Document with all $refs resolved
	Path             string         `json:"path,omitempty"`   // Normalized path to the selected node.
	GivenPath        *jsonpath.Path `json:"-"`                // Parsed JSONPath from Rule.Given.

	// Index is the schema-aware document index, built once per lint run.
	// Rule functions read it via Target; they should not need it directly.
	Index *selector.Index `json:"-"`

	// Target is the per-iteration unit set by the rules executor. Each call
	// to RunRule corresponds to one Target so functions can uniformly
	// reason about value vs. field mode, presence vs. absence, etc.
	Target *selector.Target `json:"-"`
}

type RuleFunction interface {
	RunRule(value interface{}, context RuleFunctionContext) []RuleFunctionResult
}
