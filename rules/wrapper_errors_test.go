package rules

import (
	"embed"
	"runtime"

	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/open-rpc/openrpc-linter/functions"
	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/open-rpc/openrpc-linter/types"
	"github.com/theory/jsonpath/spec"
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
	base.Extends = []types.RuleDefaults{types.RuleExtensionRecommended}
	partial := types.Rule{Given: "$.override", Extends: []types.RuleDefaults{}}
	want := base
	want.Given = partial.Given
	want.Extends = partial.Extends
	if got := mergeRule(base, partial); !reflect.DeepEqual(got, want) {
		t.Fatalf("partial override: %+v, want %+v", got, want)
	}
	if got := mergeRule(base, types.Rule{}); !reflect.DeepEqual(got, base) {
		t.Fatalf("inherited extensions: %+v, want %+v", got, base)
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

func TestMissingEmbeddedRuleExtension(t *testing.T) {
	original := ruleExtensionsFS
	ruleExtensionsFS = embed.FS{}
	t.Cleanup(func() { ruleExtensionsFS = original })
	if _, err := getExtendedRules([]types.RuleDefaults{types.RuleExtensionRecommended}); err == nil {
		t.Fatal("expected missing extension error")
	}
}

func TestRulesPathAfterWorkspaceRemoval(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not permit removing the working directory")
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Error(err)
		}
	})
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRulesFileFromPath("rules.yml"); err == nil {
		t.Fatal("expected relative path error")
	}
}

type movingTargetRule struct{}

func (movingTargetRule) RunRule(_ any, context types.RuleFunctionContext) []types.RuleFunctionResult {
	context.Target.Path = spec.NormalizedPath{spec.Name("changed")}
	return []types.RuleFunctionResult{{Message: "fallback"}, {Message: "explicit", Path: []string{"$.explicit"}}}
}

func TestResultPathsFollowCallback(t *testing.T) {
	target := selector.Target{}
	got := executeTarget(movingTargetRule{}, &target, types.RuleFunctionContext{}, nil)
	want := []types.RuleFunctionResult{{Message: "fallback", Path: []string{"$['changed']"}}, {Message: "explicit", Path: []string{"$.explicit"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("callback paths: %+v, want %+v", got, want)
	}
}
