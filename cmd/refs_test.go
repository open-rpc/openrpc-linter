package cmd

import (
	"github.com/open-rpc/openrpc-linter/types"
	"reflect"
	"testing"
)

func TestReferenceBoundaryCases(t *testing.T) {
	if _, err := resolveRefs(make(chan int)); err == nil {
		t.Fatal("accepted non-JSON value")
	}
	doc := map[string]any{"value": "ping", "arr": []any{"pong"}}
	if got := resolveJSONPointer("", doc); !reflect.DeepEqual(got, doc) {
		t.Fatalf("root pointer: %v", got)
	}
	for _, path := range []string{"missing", "arr/0", "value/child"} {
		if got := resolveJSONPointer(path, doc); got != nil {
			t.Fatalf("unsupported pointer %q: %v", path, got)
		}
	}
	ref := map[string]any{"$ref": "#/missing"}
	if got := resolveRefsRecursive(ref, doc, types.ResolvingRefs{}); !reflect.DeepEqual(got, ref) {
		t.Fatalf("unresolved reference: %v", got)
	}
}

func TestResolveRefsJSONRejectsMalformedInput(t *testing.T) {
	if _, err := resolveRefsJSON([]byte("{"), nil); err == nil {
		t.Fatal("expected JSON decoding error")
	}
}
