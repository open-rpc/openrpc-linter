package functions

import (
	"reflect"
	"testing"

	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/open-rpc/openrpc-linter/types"
	"github.com/theory/jsonpath/spec"
)

func TestReferencedRuleReportsUnreferencedComponentSchema(t *testing.T) {
	document := referencedDoc(map[string]interface{}{})
	target := componentSchemaTarget("Pet")

	results := NewReferencedRule().RunRule(target.Node, types.RuleFunctionContext{
		Document: document,
		Target:   &target,
	})

	if len(results) != 1 {
		t.Fatalf("expected one result, got %+v", results)
	}
	if results[0].Message != `unused component "schemas.Pet"` {
		t.Fatalf("unexpected message: %+v", results)
	}
	if !reflect.DeepEqual(results[0].Path, []string{"$['components']['schemas']['Pet']"}) {
		t.Fatalf("unexpected path: %+v", results[0].Path)
	}
}

func TestReferencedRuleDoesNotReportDirectlyReferencedSchema(t *testing.T) {
	document := referencedDoc(map[string]interface{}{
		"methods": []interface{}{
			map[string]interface{}{
				"result": map[string]interface{}{
					"schema": map[string]interface{}{"$ref": "#/components/schemas/Pet"},
				},
			},
		},
	})
	target := componentSchemaTarget("Pet")

	results := NewReferencedRule().RunRule(target.Node, types.RuleFunctionContext{
		Document: document,
		Target:   &target,
	})

	if len(results) != 0 {
		t.Fatalf("expected no result, got %+v", results)
	}
}

func TestReferencedRuleUsesOriginalDocumentInsteadOfResolvedDocument(t *testing.T) {
	document := referencedDoc(map[string]interface{}{
		"methods": []interface{}{
			map[string]interface{}{
				"result": map[string]interface{}{
					"schema": map[string]interface{}{"$ref": "#/components/schemas/Pet"},
				},
			},
		},
	})
	resolvedDocument := referencedDoc(map[string]interface{}{
		"methods": []interface{}{
			map[string]interface{}{
				"result": map[string]interface{}{
					"schema": map[string]interface{}{"type": "object"},
				},
			},
		},
	})
	target := componentSchemaTarget("Pet")

	results := NewReferencedRule().RunRule(target.Node, types.RuleFunctionContext{
		Document:         document,
		ResolvedDocument: resolvedDocument,
		Target:           &target,
	})

	if len(results) != 0 {
		t.Fatalf("expected original document refs to drive reachability, got %+v", results)
	}
}

func TestReferencedRuleDoesNotReportTransitivelyReferencedSchema(t *testing.T) {
	document := referencedDoc(map[string]interface{}{
		"methods": []interface{}{
			map[string]interface{}{
				"result": map[string]interface{}{
					"schema": map[string]interface{}{"$ref": "#/components/schemas/A"},
				},
			},
		},
	})
	schemas := document["components"].(map[string]interface{})["schemas"].(map[string]interface{})
	schemas["A"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"b": map[string]interface{}{"$ref": "#/components/schemas/B"},
		},
	}
	schemas["B"] = map[string]interface{}{"type": "string"}

	rule := NewReferencedRule()
	ctx := types.RuleFunctionContext{Document: document}
	for _, name := range []string{"A", "B"} {
		target := componentSchemaTarget(name)
		ctx.Target = &target
		if results := rule.RunRule(target.Node, ctx); len(results) != 0 {
			t.Fatalf("expected %s to be reachable, got %+v", name, results)
		}
	}
}

func TestReferencedRuleDoesNotReportSchemaReferencedOnlyByAnotherComponent(t *testing.T) {
	document := referencedDoc(map[string]interface{}{})
	schemas := document["components"].(map[string]interface{})["schemas"].(map[string]interface{})
	schemas["UnusedA"] = map[string]interface{}{"$ref": "#/components/schemas/UnusedB"}
	schemas["UnusedB"] = map[string]interface{}{"type": "string"}

	rule := NewReferencedRule()
	ctx := types.RuleFunctionContext{Document: document}
	var messages []string
	for _, name := range []string{"UnusedA", "UnusedB"} {
		target := componentSchemaTarget(name)
		ctx.Target = &target
		messages = append(messages, resultMessages(rule.RunRule(target.Node, ctx))...)
	}

	expected := []string{`unused component "schemas.UnusedA"`}
	if !reflect.DeepEqual(messages, expected) {
		t.Fatalf("expected %+v, got %+v", expected, messages)
	}
}

func TestReferencedRuleReportsSelfReferencedComponent(t *testing.T) {
	document := referencedDoc(map[string]interface{}{})
	schemas := document["components"].(map[string]interface{})["schemas"].(map[string]interface{})
	schemas["Pet"] = map[string]interface{}{"$ref": "#/components/schemas/Pet"}
	target := componentSchemaTarget("Pet")

	results := NewReferencedRule().RunRule(target.Node, types.RuleFunctionContext{
		Document: document,
		Target:   &target,
	})

	if len(results) != 1 {
		t.Fatalf("expected self-reference not to count as usage, got %+v", results)
	}
}

func TestReferencedRuleHandlesEscapedJSONPointerNames(t *testing.T) {
	document := referencedDoc(map[string]interface{}{
		"methods": []interface{}{
			map[string]interface{}{
				"result": map[string]interface{}{
					"schema": map[string]interface{}{"$ref": "#/components/schemas/Pet~1Owner~0Draft"},
				},
			},
		},
	})
	schemas := document["components"].(map[string]interface{})["schemas"].(map[string]interface{})
	schemas["Pet/Owner~Draft"] = map[string]interface{}{"type": "object"}
	target := componentSchemaTarget("Pet/Owner~Draft")

	results := NewReferencedRule().RunRule(target.Node, types.RuleFunctionContext{
		Document: document,
		Target:   &target,
	})

	if len(results) != 0 {
		t.Fatalf("expected escaped component name to be reachable, got %+v", results)
	}
}

func TestReferencedRuleIgnoresExternalRefs(t *testing.T) {
	document := referencedDoc(map[string]interface{}{
		"methods": []interface{}{
			map[string]interface{}{
				"result": map[string]interface{}{
					"schema": map[string]interface{}{"$ref": "https://example.com/schemas.json#/Pet"},
				},
			},
		},
	})
	target := componentSchemaTarget("Pet")

	results := NewReferencedRule().RunRule(target.Node, types.RuleFunctionContext{
		Document: document,
		Target:   &target,
	})

	if len(results) != 1 {
		t.Fatalf("expected external ref not to mark component reachable, got %+v", results)
	}
}

func referencedDoc(extra map[string]interface{}) map[string]interface{} {
	document := map[string]interface{}{
		"openrpc": "1.3.2",
		"info":    map[string]interface{}{"title": "Test", "version": "1.0.0"},
		"methods": []interface{}{},
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{
				"Pet": map[string]interface{}{"type": "object"},
			},
		},
	}
	for key, value := range extra {
		document[key] = value
	}
	return document
}

func componentSchemaTarget(name string) selector.Target {
	return selector.Target{
		Path: spec.NormalizedPath{
			spec.Name("components"),
			spec.Name("schemas"),
			spec.Name(name),
		},
		Node:   map[string]interface{}{"type": "object"},
		Exists: true,
	}
}
