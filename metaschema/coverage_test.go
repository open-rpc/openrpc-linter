package metaschema

import "testing"

func TestMalformedSchemaBoundaries(t *testing.T) {
	for _, raw := range []string{"{", "[]"} {
		if _, err := parse("test", raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	if _, err := Version(nil); err == nil {
		t.Fatal("accepted non-document")
	}
	if m, err := For(nil); err != nil || m.VersionFamily != "1.4.x" {
		t.Fatalf("fallback: %+v %v", m, err)
	}
	m := &MetaSchema{root: map[string]any{}}
	if got := m.Resolve("#/definitions/missing"); got != nil {
		t.Fatalf("missing definitions: %v", got)
	}
	m.root = map[string]any{"type": 42}
	if _, err := m.Compile(); err == nil {
		t.Fatal("accepted invalid schema")
	}
}
