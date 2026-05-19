package types

type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
	SeverityInfo  Severity = "info"
)

type RuleDefaults string

const (
	RuleExtensionRecommended RuleDefaults = "recommended"
)

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

type RuleFunctionResult struct {
	Message string   `json:"message,omitempty"`
	Path    []string `json:"path,omitempty"`
	RuleID  string   `json:"ruleId,omitempty"`
}

type RuleFunctionContext struct {
	Rule             *Rule       `json:"rule"`
	RuleID           string      `json:"ruleId"`
	Document         interface{} `json:"document"`         // Original document with potential $refs
	ResolvedDocument interface{} `json:"resolvedDocument"` // Document with all $refs resolved
	ArrayIndex       *int        `json:"arrayIndex,omitempty"`
	Path             string      `json:"path,omitempty"` // Normalized path to the selected node.
}

type RuleFunction interface {
	RunRule(value interface{}, context RuleFunctionContext) []RuleFunctionResult
}
