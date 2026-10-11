package selector

import (
	"github.com/open-rpc/openrpc-linter/metaschema"
	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
	"testing"
)

func TestSelectionBoundaryCases(t *testing.T) {
	if got := Select(nil, nil, nil); len(got) != 0 {
		t.Fatalf("nil query: %v", got)
	}
	p, err := jsonpath.Parse("$")
	if err != nil {
		t.Fatal(err)
	}
	if got := Select(p, "root", nil); len(got) != 1 || got[0].Node != "root" {
		t.Fatalf("root query: %v", got)
	}
	if got := rebase(Target{Node: "value"}, nil); got.Node != "value" {
		t.Fatalf("empty rebase: %v", got)
	}
	target := Target{Field: "name", Exists: false}
	if target.PathString() != "$" {
		t.Fatalf("root path: %q", target.PathString())
	}
	idx := &Index{ByField: map[string][]*Candidate{"name": {{Node: map[string]any{}, Fields: map[string]struct{}{"name": {}}}}}}
	p, err = jsonpath.Parse("$..name.value")
	if err != nil {
		t.Fatal(err)
	}
	if got := Select(p, map[string]any{}, idx); len(got) != 0 {
		t.Fatalf("missing intermediate: %v", got)
	}
	cand := &Candidate{Path: spec.NormalizedPath{}, Node: map[string]any{"name": "ping"}}
	idx.ByField["name"] = []*Candidate{cand, cand}
	if got := descendantFieldTargets(nil, "name", cand.Node, idx); len(got) != 1 {
		t.Fatalf("duplicate candidates: %v", got)
	}
}

func TestIndexSchemaBoundaries(t *testing.T) {
	meta, err := metaschema.Latest()
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{meta: meta, idx: &Index{ByField: map[string][]*Candidate{}, ByPath: map[string]*Candidate{}}}
	b.walk(map[string]any{}, nil, nil)
	b.walk(map[string]any{}, map[string]any{"$ref": "https://external.example/schema"}, nil)
	b.indexArray([]any{1}, map[string]any{}, nil)
	child := map[string]any{"type": "string"}
	if got := b.schemaForKey(map[string]any{"additionalProperties": child}, "extra"); got["type"] != "string" {
		t.Fatalf("additional property: %v", got)
	}
	for _, schema := range []map[string]any{
		{"oneOf": []any{1, map[string]any{"$ref": "https://external.example/schema"}}},
		{"oneOf": []any{map[string]any{"title": "referenceObject"}}},
	} {
		if got := b.chooseOneOf(schema); got != nil {
			t.Fatalf("unsupported alternatives: %v", got)
		}
	}
	if matchesPattern("unknown", "x-field") {
		t.Fatal("unknown pattern must not match")
	}
}

func TestUnknownCandidateTitleAndEmptyScopes(t *testing.T) {
	idx := &Index{ByPath: map[string]*Candidate{}, ByField: map[string][]*Candidate{}}
	if got := candidateTitle(idx, nil); got != "" {
		t.Fatalf("unknown candidate: %q", got)
	}
	if got := descendantFieldTargets(nil, "name", nil, idx); len(got) != 0 {
		t.Fatalf("empty candidates: %v", got)
	}
}
