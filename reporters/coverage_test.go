package reporters

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/open-rpc/openrpc-linter/types"
)

func TestJSONReporterRoundTrip(t *testing.T) {
	want := []types.RuleFunctionResult{{Message: "missing", Path: []string{"$"}, RuleID: "r", Severity: types.SeverityWarn}}
	var buf bytes.Buffer
	if err := (&JSONReporter{}).Format(want, 1, &buf); err != nil {
		t.Fatal(err)
	}
	var got []types.RuleFunctionResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestColorWriterHeuristics(t *testing.T) {
	for _, key := range []string{"NO_COLOR", "FORCE_COLOR", "CLICOLOR_FORCE", "TERM"} {
		t.Setenv(key, "")
	}
	t.Setenv("TERM", "dumb")
	if supportsColor(&bytes.Buffer{}) {
		t.Fatal("dumb terminal has color")
	}
	t.Setenv("TERM", "xterm")
	if supportsColor(&bytes.Buffer{}) {
		t.Fatal("buffer has color")
	}
	f, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	if supportsColor(f) {
		t.Fatal("regular file has color")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if supportsColor(f) {
		t.Fatal("closed file has color")
	}
}

func TestSecondaryDescriptorAndOverflowIndex(t *testing.T) {
	if got := formatSecondaryLabels(types.PathLabels{Descriptor: "Item"}, groupKindMethod); !reflect.DeepEqual(got, []string{`descriptor: "Item"`}) {
		t.Fatalf("secondary labels: %v", got)
	}
	if got := methodIndexFromPath("$['methods'][" + strings.Repeat("9", 100) + "]"); got != -1 {
		t.Fatalf("overflow index: %d", got)
	}
}

func TestStableGroupedRows(t *testing.T) {
	rows := []pending{
		{row: row{groupKind: groupKindMethod, groupName: "b", path: "same", ruleID: "r", message: "z"}, methodIdx: 2},
		{row: row{groupKind: groupKindMethod, groupName: "b", path: "same", ruleID: "r", message: "a"}, methodIdx: 1},
		{row: row{groupKind: groupKindMethod, groupName: "b", path: "same", ruleID: "a", message: "x"}, methodIdx: 1},
	}
	groups := groupRows(rows)
	if len(groups) != 1 || groups[0].methodIx != 1 {
		t.Fatalf("groups: %+v", groups)
	}
	if got := groups[0].rows; got[0].ruleID != "a" || got[1].message != "a" || got[2].message != "z" {
		t.Fatalf("row order: %+v", got)
	}
	known := &bucket{kind: groupKindMethod, name: "z", methodIx: 1}
	unknown := &bucket{kind: groupKindMethod, name: "a", methodIx: -1}
	if !known.less(unknown) || unknown.less(known) {
		t.Fatal("indexed methods must precede unknown methods")
	}
	equal := &bucket{kind: groupKindMethod, name: "a", methodIx: 1}
	if !equal.less(known) {
		t.Fatal("equal method indexes must sort by name")
	}
	if rows, _ := collectRows([]types.RuleFunctionResult{{}}); len(rows) != 0 {
		t.Fatalf("empty diagnostic: %+v", rows)
	}
}

func TestForcedColorAndSecondarySchema(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("FORCE_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	if !supportsColor(&bytes.Buffer{}) {
		t.Fatal("explicit color force ignored")
	}
	if got := formatSecondaryLabels(types.PathLabels{Schema: "Result"}, groupKindMethod); !reflect.DeepEqual(got, []string{`schema: "Result"`}) {
		t.Fatalf("schema label: %v", got)
	}
	groups := groupRows([]pending{
		{row: row{groupKind: groupKindGeneral, groupName: "general", path: "same", ruleID: "z"}, methodIdx: -1},
		{row: row{groupKind: groupKindGeneral, groupName: "general", path: "same", ruleID: "a"}, methodIdx: -1},
	})
	if groups[0].rows[0].ruleID != "a" {
		t.Fatal("rule order")
	}
}
