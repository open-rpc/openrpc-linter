package rules

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/open-rpc/openrpc-linter/functions"
	"github.com/open-rpc/openrpc-linter/types"
)

func TestRuleOverrideFields(t *testing.T) {
	base := types.Rule{Description: "base", Given: "$.base", Then: &types.RuleAction{Function: "truthy"}, Severity: types.SeverityError}
	override := types.Rule{Description: "override", Given: "$.override", Then: &types.RuleAction{Function: "unique"}, Extends: []types.RuleDefaults{types.RuleExtensionRecommended}, Severity: types.SeverityWarn}
	if got := mergeRule(base, override); !reflect.DeepEqual(got, override) {
		t.Fatalf("override: %+v", got)
	}
	if got := mergeRule(base, types.Rule{}); !reflect.DeepEqual(got, base) {
		t.Fatalf("empty override: %+v", got)
	}
	if got := mergeRules(map[string]types.Rule{"r": base}, map[string]types.Rule{"r": override}); !reflect.DeepEqual(got["r"], override) {
		t.Fatalf("merge: %+v", got)
	}
}

func TestRuleFileBoundaries(t *testing.T) {
	for _, files := range []fstest.MapFS{{}, {"rules.yml": {Data: []byte("rules: [")}}} {
		if _, err := LoadRulesFile(files, "rules.yml"); err == nil {
			t.Fatal("expected file error")
		}
	}
	path := filepath.Join(t.TempDir(), "rules.yml")
	if err := os.WriteFile(path, []byte("rules:\n  r:\n    given: '$'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadRulesFileFromPath(path); err != nil || got.Rules["r"].Given != "$" {
		t.Fatalf("file rule: %+v %v", got, err)
	}
	if _, err := (&RulesWrapper{Extends: []types.RuleDefaults{"unknown"}}).ResolvedRules(); err == nil {
		t.Fatal("accepted unknown extension")
	}
}

type blankResultRule struct{}

func (blankResultRule) RunRule(any, types.RuleFunctionContext) []types.RuleFunctionResult {
	return []types.RuleFunctionResult{{}, {Message: "failure"}}
}

func TestExecutorEmptyResultsAndPaths(t *testing.T) {
	if got, err := ExecuteRule(&types.Rule{}, types.RuleFunctionContext{}); err != nil || len(got) != 0 {
		t.Fatalf("missing action: %v %v", got, err)
	}
	const name = "coverage-result"
	functions.FunctionRegistry[name] = func() types.RuleFunction { return blankResultRule{} }
	t.Cleanup(func() { delete(functions.FunctionRegistry, name) })
	rule := &types.Rule{Given: "$", Then: &types.RuleAction{Function: name}}
	got, err := ExecuteRule(rule, types.RuleFunctionContext{Document: map[string]any{}})
	if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0].Path, []string{"$"}) {
		t.Fatalf("result fallback: %v %v", got, err)
	}
}
