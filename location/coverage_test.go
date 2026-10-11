package location

import (
	"github.com/open-rpc/openrpc-linter/types"
	"testing"
)

func TestPathBoundaryFallbacks(t *testing.T) {
	for _, tc := range []struct {
		doc  any
		path string
	}{
		{nil, "$"}, {map[string]any{"info": "text"}, "$['info']['title']"},
	} {
		if got := Resolve(tc.doc, tc.path); !got.IsEmpty() {
			t.Fatalf("invalid walk: %+v", got)
		}
	}
	if got := FriendlyPath("['info']['title']"); got != "info.title" {
		t.Fatalf("relative friendly path: %q", got)
	}
	doc := map[string]any{"components": map[string]any{"schemas": []any{map[string]any{"title": "Item"}}}}
	got := Resolve(doc, "$['components']['schemas'][0]")
	if got != (types.PathLabels{Section: "components", Schema: "Item"}) {
		t.Fatalf("array schema labels: %+v", got)
	}
}

func TestNamedSchemaAndTagLabels(t *testing.T) {
	doc := map[string]any{"methods": []any{map[string]any{"name": "ping", "result": map[string]any{"schema": map[string]any{"title": "Pong"}}, "tags": []any{map[string]any{"name": "rpc"}}}}}
	if got := Resolve(doc, "$['methods'][0]['result']['schema']"); got.Schema != "Pong" {
		t.Fatalf("schema title: %+v", got)
	}
	if got := Resolve(doc, "$['methods'][0]['tags'][0]"); got.Tag != "rpc" {
		t.Fatalf("tag: %+v", got)
	}
	if _, ok := stringField("text", "name"); ok {
		t.Fatal("non-object field")
	}
	if got := FriendlyPath("unknown"); got != "unknown" {
		t.Fatalf("unknown path: %q", got)
	}
}
