package reporters

import (
	"reflect"
	"strings"
	"testing"

	"github.com/open-rpc/openrpc-linter/internal/paths"
	"github.com/open-rpc/openrpc-linter/types"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"empty", "", 10, ""},
		{"exact fit", "hello", 5, "hello"},
		{"under limit", "hi", 10, "hi"},
		{"over limit", "hello world", 8, "hello w…"},
		{"max one", "hello", 1, "…"},
		{"max zero passthrough", "hello", 0, "hello"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := truncate(tc.s, tc.max); got != tc.want {
				t.Fatalf("truncate(%q, %d) = %q, want %q", tc.s, tc.max, got, tc.want)
			}
		})
	}
}

func TestColWidth(t *testing.T) {
	if got := colWidth([]string{"a", "abc", "ab"}, 48); got != 3 {
		t.Fatalf("colWidth = %d, want 3", got)
	}
	if got := colWidth([]string{strings.Repeat("x", 60)}, 48); got != 48 {
		t.Fatalf("colWidth capped = %d, want 48", got)
	}
}

func TestResolveGroup(t *testing.T) {
	cases := []struct {
		name     string
		labels   types.PathLabels
		wantKind groupKind
		wantName string
	}{
		{"method wins over nested schema", types.PathLabels{Method: "foo", Schema: "Bar"}, groupKindMethod, "foo"},
		{"components schema", types.PathLabels{Section: "components", Schema: "Pet"}, groupKindSchema, "Pet"},
		{"components descriptor", types.PathLabels{Section: "components", Descriptor: "Block"}, groupKindDescriptor, "Block"},
		{"components tag", types.PathLabels{Section: "components", Tag: "eth"}, groupKindTag, "eth"},
		{"components section without an anchor", types.PathLabels{Section: "components"}, groupKindSection, "components"},
		{"info section", types.PathLabels{Section: "info"}, groupKindSection, "info"},
		{"empty falls back to general", types.PathLabels{}, groupKindGeneral, "general"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, name := resolveGroup(tc.labels)
			if kind != tc.wantKind || name != tc.wantName {
				t.Fatalf("resolveGroup(%+v) = (%v,%q), want (%v,%q)", tc.labels, kind, name, tc.wantKind, tc.wantName)
			}
		})
	}
}

func TestFormatGroupHeader(t *testing.T) {
	cases := []struct {
		kind       groupKind
		name, want string
	}{
		{groupKindMethod, "eth_getLogs", "eth_getLogs"},
		{groupKindSchema, "Pet", `schema "Pet"`},
		{groupKindDescriptor, "Block", `descriptor "Block"`},
		{groupKindTag, "eth", `tag "eth"`},
		{groupKindSection, "info", "info"},
		{groupKindGeneral, "general", "general"},
	}
	for _, tc := range cases {
		if got := formatGroupHeader(tc.kind, tc.name); got != tc.want {
			t.Errorf("formatGroupHeader(%v,%q) = %q, want %q", tc.kind, tc.name, got, tc.want)
		}
	}
}

func TestFormatSecondaryLabels(t *testing.T) {
	cases := []struct {
		name      string
		labels    types.PathLabels
		groupKind groupKind
		want      []string
	}{
		{
			name:      "method group surfaces schema + param",
			labels:    types.PathLabels{Method: "m", Param: "p", Schema: "s"},
			groupKind: groupKindMethod,
			want:      []string{`param: "p"`, `schema: "s"`},
		},
		{
			name:      "schema group hides matching schema",
			labels:    types.PathLabels{Section: "components", Schema: "Pet"},
			groupKind: groupKindSchema,
			want:      nil,
		},
		{
			name:      "descriptor group hides matching descriptor",
			labels:    types.PathLabels{Section: "components", Descriptor: "Block"},
			groupKind: groupKindDescriptor,
			want:      nil,
		},
		{
			name:      "tag group hides matching tag",
			labels:    types.PathLabels{Section: "components", Tag: "eth"},
			groupKind: groupKindTag,
			want:      nil,
		},
		{
			name:      "no anchors",
			labels:    types.PathLabels{},
			groupKind: groupKindGeneral,
			want:      nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSecondaryLabels(tc.labels, tc.groupKind)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMethodIndexFromPath(t *testing.T) {
	cases := []struct {
		path string
		want int
	}{
		{"$['methods'][0]['description']", 0},
		{"$['methods'][12]['params'][3]", 12},
		{"$['info']['license']", -1},
		{"", -1},
	}
	for _, tc := range cases {
		position, _ := paths.Parse(tc.path)
		if got := position.MethodIndex(); got != tc.want {
			t.Errorf("methodIndexFromPath(%q) = %d, want %d", tc.path, got, tc.want)
		}
	}
}
