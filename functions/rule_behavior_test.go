package functions

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/open-rpc/openrpc-linter/types"
	"github.com/theory/jsonpath/spec"
)

func TestTruthyValuesAndPaths(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  any
		truthy bool
	}{
		{"nil", nil, false}, {"empty", "", false}, {"null", "null", false}, {"text", "ping", true}, {"false", false, true}, {"zero", float64(0), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, withTarget := range []bool{false, true} {
				ctx := types.RuleFunctionContext{Path: "$.fallback"}
				path := "$.fallback"
				if withTarget {
					ctx.Target = &selector.Target{Path: spec.NormalizedPath{spec.Name("name")}, Exists: true}
					path = "$['name']"
				}
				got := (&TruthyRule{}).RunRule(tc.value, ctx)
				if tc.truthy {
					if len(got) != 0 {
						t.Fatalf("unexpected diagnostic: %+v", got)
					}
					continue
				}
				want := []types.RuleFunctionResult{{Message: "Field must have a truthy value", Path: []string{path}}}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %+v, want %+v", got, want)
				}
			}
		})
	}
	if got := (&TruthyRule{}).RunRule(nil, types.RuleFunctionContext{}); len(got) != 1 || len(got[0].Path) != 0 {
		t.Fatalf("empty path: %+v", got)
	}
}

func TestRegisteredFunctions(t *testing.T) {
	for _, name := range []string{"truthy", "schema", "unique"} {
		if factory := FunctionRegistry[name]; factory == nil || factory() == nil {
			t.Fatalf("missing factory %q", name)
		}
	}
}

func TestSchemaMissingFieldAndCache(t *testing.T) {
	r := &SchemaRule{}
	if got := r.RunRule(nil, types.RuleFunctionContext{Target: &selector.Target{Field: "name"}}); len(got) != 0 {
		t.Fatalf("missing field: %+v", got)
	}
	action := &types.RuleAction{FunctionOptions: map[string]any{"type": "string"}}
	ctx := types.RuleFunctionContext{Rule: &types.Rule{Then: action}}
	for range 2 {
		if got := r.RunRule("ping", ctx); len(got) != 0 {
			t.Fatalf("cached validation: %+v", got)
		}
	}
	invalid := &types.RuleAction{FunctionOptions: map[string]any{"$id": false}}
	if _, err := r.schemaFor(invalid); err == nil {
		t.Fatal("expected resource error")
	}
}

func TestUniquePrimitiveIdentity(t *testing.T) {
	r := &UniqueRule{}
	ctx := types.RuleFunctionContext{}
	if got := r.RunRule(nil, ctx); len(got) != 0 {
		t.Fatalf("nil target: %+v", got)
	}
	for _, v := range []any{true, false, float64(1), "1", nil} {
		ctx.Target = &selector.Target{Path: spec.NormalizedPath{spec.Name("first")}, Exists: true}
		if got := r.RunRule(v, ctx); len(got) != 0 {
			t.Fatalf("first %v: %+v", v, got)
		}
		ctx.Target = &selector.Target{Path: spec.NormalizedPath{spec.Name("second")}, Exists: true}
		got := r.RunRule(v, ctx)
		if len(got) != 1 || !strings.Contains(got[0].Message, "first seen at $['first']") {
			t.Fatalf("duplicate %v: %+v", v, got)
		}
	}
}

func TestUniqueMissingActionDefaults(t *testing.T) {
	for _, rule := range []*types.Rule{nil, {}} {
		target := selector.Target{Field: "name"}
		got := NewUniqueRule().RunRule(nil, types.RuleFunctionContext{Rule: rule, Target: &target})
		if len(got) != 0 {
			t.Fatalf("missing options: %+v", got)
		}
	}
}

func TestUniqueScopeUsesResolvedDocument(t *testing.T) {
	r := NewUniqueRule()
	target := selector.Target{Path: spec.NormalizedPath{spec.Name("methods"), spec.Index(0), spec.Name("name")}, Exists: true}
	ctx := types.RuleFunctionContext{
		Rule:     &types.Rule{Then: &types.RuleAction{FunctionOptions: map[string]any{"scope": "$.methods[*]"}}},
		Document: map[string]any{}, ResolvedDocument: map[string]any{"methods": []any{map[string]any{"name": "ping"}}}, Target: &target,
	}
	if got := r.RunRule("ping", ctx); len(got) != 0 {
		t.Fatalf("first: %+v", got)
	}
	if len(r.scopePaths) != 1 {
		t.Fatalf("expected resolved scope, got %v", r.scopePaths)
	}
	if got := r.RunRule("ping", ctx); len(got) != 1 {
		t.Fatalf("expected duplicate in resolved scope: %+v", got)
	}
}

// Relative schema resources need a working directory. Check the error boundary
// when a workspace is removed while the process still has it open.
func TestSchemaRemovedWorkspace(t *testing.T) {
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
	action := &types.RuleAction{FunctionOptions: map[string]any{"type": "string"}}
	if _, err := (&SchemaRule{}).schemaFor(action); err == nil {
		t.Fatal("expected relative resource error")
	}
}
