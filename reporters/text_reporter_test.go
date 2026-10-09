package reporters

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/open-rpc/openrpc-linter/types"
)

// helper: split lines, drop trailing blanks.
func outLines(s string) []string {
	parts := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return parts
}

func TestGroupHeader_Method(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     "method-description",
			Message:    "missing field 'description'",
			Path:       []string{"$['methods'][0]['description']"},
			PathLabels: types.PathLabels{Method: "eth_getLogs"},
			Severity:   types.SeverityError,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "\neth_getLogs\n") {
		t.Fatalf("expected bare method group header 'eth_getLogs', got:\n%s", outStr)
	}
	if strings.Contains(outStr, `method "eth_getLogs"`) {
		t.Fatalf("expected method label dropped from rows, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "methods[0].description") {
		t.Fatalf("expected friendly path in row, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "error") {
		t.Fatalf("expected severity label, got:\n%s", outStr)
	}
}

func TestGroupHeader_ComponentsSchema(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     "schema-title",
			Message:    "missing field 'title'",
			Path:       []string{"$['components']['schemas']['Pet']['title']"},
			PathLabels: types.PathLabels{Section: "components", Schema: "Pet"},
			Severity:   types.SeverityWarn,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "\nschema \"Pet\"\n") {
		t.Fatalf("expected `schema \"Pet\"` group header, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "components.schemas.Pet.title") {
		t.Fatalf("expected friendly path, got:\n%s", outStr)
	}
}

func TestGroupHeader_InfoSection(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     "info-license",
			Message:    "missing field 'license'",
			Path:       []string{"$['info']['license']"},
			PathLabels: types.PathLabels{Section: "info"},
			Severity:   types.SeverityWarn,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	if !strings.Contains(output.String(), "\ninfo\n") {
		t.Fatalf("expected bare `info` group header, got:\n%s", output.String())
	}
}

func TestGroupHeader_General(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:   "openrpc-version",
			Message:  "missing field 'openrpc'",
			Path:     []string{"$['openrpc']"},
			Severity: types.SeverityError,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	if !strings.Contains(output.String(), "\ngeneral\n") {
		t.Fatalf("expected bare `general` group header for unlabeled row, got:\n%s", output.String())
	}
}

func TestSecondaryLabel_AppendsContinuationLine(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     "schema-description",
			Message:    "missing field 'description'",
			Path:       []string{"$['methods'][0]['result']['schema']['description']"},
			PathLabels: types.PathLabels{Method: "debug_getBadBlocks", Schema: "Bad block"},
			Severity:   types.SeverityWarn,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "\n    schema: \"Bad block\"\n") {
		t.Fatalf("expected indented `schema:` continuation line, got:\n%s", outStr)
	}
}

func TestSecondaryLabel_OmittedWhenSameAsGroup(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     "schema-title",
			Message:    "missing field 'title'",
			Path:       []string{"$['components']['schemas']['Pet']['title']"},
			PathLabels: types.PathLabels{Section: "components", Schema: "Pet"},
			Severity:   types.SeverityWarn,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	if strings.Contains(output.String(), "schema: \"Pet\"") {
		t.Fatalf("schema-grouped row must not repeat schema as a continuation line, got:\n%s", output.String())
	}
}

func TestSecondaryLabel_Param(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     "schema-description",
			Message:    "missing field 'description'",
			Path:       []string{"$['methods'][0]['params'][0]['schema']['description']"},
			PathLabels: types.PathLabels{Method: "debug_getRawBlock", Param: "Block", Schema: "Block"},
			Severity:   types.SeverityWarn,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "\n    param: \"Block\"\n") {
		t.Fatalf("expected indented `param:` continuation line, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "\n    schema: \"Block\"\n") {
		t.Fatalf("expected indented `schema:` continuation line, got:\n%s", outStr)
	}
}

func TestGroupOrder_MethodsFirstByDocOrder(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		// Intentionally provide methods[1] before methods[0] and a schema row.
		{
			RuleID:     "schema-title",
			Message:    "missing title",
			Path:       []string{"$['components']['schemas']['Pet']['title']"},
			PathLabels: types.PathLabels{Section: "components", Schema: "Pet"},
			Severity:   types.SeverityWarn,
		},
		{
			RuleID:     "method-description",
			Message:    "missing desc on second method",
			Path:       []string{"$['methods'][1]['description']"},
			PathLabels: types.PathLabels{Method: "second_method"},
			Severity:   types.SeverityError,
		},
		{
			RuleID:     "method-description",
			Message:    "missing desc on first method",
			Path:       []string{"$['methods'][0]['description']"},
			PathLabels: types.PathLabels{Method: "first_method"},
			Severity:   types.SeverityError,
		},
		{
			RuleID:   "info-license",
			Message:  "missing license",
			Path:     []string{"$['info']['license']"},
			Severity: types.SeverityWarn,
			PathLabels: types.PathLabels{
				Section: "info",
			},
		},
	}, 4, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	outStr := output.String()
	firstIdx := strings.Index(outStr, "\nfirst_method\n")
	secondIdx := strings.Index(outStr, "\nsecond_method\n")
	schemaIdx := strings.Index(outStr, "\nschema \"Pet\"\n")
	infoIdx := strings.Index(outStr, "\ninfo\n")

	if firstIdx < 0 || secondIdx < 0 || schemaIdx < 0 || infoIdx < 0 {
		t.Fatalf("expected all group headers present, got:\n%s", outStr)
	}
	if !(firstIdx < secondIdx && secondIdx < schemaIdx && schemaIdx < infoIdx) {
		t.Fatalf("expected order first_method < second_method < schema \"Pet\" < info, indexes %d %d %d %d:\n%s",
			firstIdx, secondIdx, schemaIdx, infoIdx, outStr)
	}
}

func TestSummary_AppearsTopAndBottom(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:   "method-description",
			Message:  "Missing description",
			Severity: types.SeverityError,
		},
		{
			RuleID:   "info-license",
			Message:  "Missing license",
			Severity: types.SeverityWarn,
		},
	}, 2, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	want := "1 error, 1 warning found in 2 rules"
	count := strings.Count(output.String(), want)
	if count != 2 {
		t.Fatalf("expected summary %q twice (top + bottom), saw %d:\n%s", want, count, output.String())
	}
}

func TestColumnHeaderRow_AlignedWithRows(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     "method-description",
			Message:    "missing",
			Path:       []string{"$['methods'][0]['description']"},
			PathLabels: types.PathLabels{Method: "m"},
			Severity:   types.SeverityError,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	lines := outLines(output.String())
	var headerLine, rowLine string
	for _, ln := range lines {
		if strings.HasPrefix(ln, "  path") {
			headerLine = ln
		}
		if strings.Contains(ln, "methods[0].description") {
			rowLine = ln
		}
	}
	if headerLine == "" || rowLine == "" {
		t.Fatalf("expected header + row line, got:\n%s", output.String())
	}
	if strings.Index(headerLine, "level") != strings.Index(rowLine, "error") {
		t.Fatalf("level column not aligned with severity cell:\nheader: %q\nrow:    %q", headerLine, rowLine)
	}
}

func TestColumnsAlign_AcrossGroups(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     "a-rule",
			Message:    "first",
			Path:       []string{"$['methods'][0]['description']"},
			PathLabels: types.PathLabels{Method: "m1"},
			Severity:   types.SeverityError,
		},
		{
			RuleID:     "b-rule",
			Message:    "second",
			Path:       []string{"$['components']['schemas']['VeryLongSchemaName']['title']"},
			PathLabels: types.PathLabels{Section: "components", Schema: "VeryLongSchemaName"},
			Severity:   types.SeverityWarn,
		},
	}, 2, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	var errLine, warnLine string
	for _, ln := range outLines(output.String()) {
		if strings.Contains(ln, "found in") {
			continue
		}
		if errLine == "" && strings.Contains(ln, "error") && strings.Contains(ln, "methods[0]") {
			errLine = ln
		}
		if warnLine == "" && strings.Contains(ln, "warning") && strings.Contains(ln, "components.schemas") {
			warnLine = ln
		}
	}
	if errLine == "" || warnLine == "" {
		t.Fatalf("expected one error and one warning row, got:\n%s", output.String())
	}
	if strings.Index(errLine, "error") != strings.Index(warnLine, "warning") {
		t.Fatalf("severity column not aligned across groups:\nerror line: %q\nwarn  line: %q", errLine, warnLine)
	}
}

func TestTextReporterInfoLabel(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:   "hint-rule",
			Message:  "Consider adding examples",
			Severity: types.SeverityInfo,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "info") {
		t.Fatalf("expected info label, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Consider adding examples") {
		t.Fatalf("expected message, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "1 info found in 1 rule") {
		t.Fatalf("expected info summary, got:\n%s", outStr)
	}
}

func TestTextReporterDefaultsMissingSeverityToError(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:  "legacy-rule",
			Message: "Something failed",
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "error") {
		t.Fatalf("expected default error label, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "legacy-rule") {
		t.Fatalf("expected rule id, got:\n%s", outStr)
	}
}

func TestTextReporterSuccessLine(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}
	if err := reporter.Format(nil, 5, &output); err != nil {
		t.Fatalf("Format returned error: %v", err)
	}
	if !strings.Contains(output.String(), "All 5 rules passed") {
		t.Fatalf("expected success line, got:\n%s", output.String())
	}
}

func TestTextReporterPrintsSourceFileHeader(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{SourceFile: "openrpc.json"}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:   "method-description",
			Message:  "missing",
			Path:     []string{"$['methods'][0]['description']"},
			Severity: types.SeverityError,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	if !strings.HasPrefix(output.String(), "openrpc.json\n") {
		t.Fatalf("expected source file header, got:\n%s", output.String())
	}
}

func TestTextReporterTruncatesLongColumns(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	longPath := "$['components']['schemas']['" + strings.Repeat("S", 80) + "']['title']"
	longMessage := strings.Repeat("x", 80)
	longRule := strings.Repeat("r", 40)

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     longRule,
			Message:    longMessage,
			Path:       []string{longPath},
			PathLabels: types.PathLabels{Section: "components", Schema: "Pet"},
			Severity:   types.SeverityError,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "…") {
		t.Fatalf("expected truncation ellipsis in output, got:\n%s", outStr)
	}
	if strings.Contains(outStr, longMessage) {
		t.Fatalf("expected long message truncated, got:\n%s", outStr)
	}
	if strings.Contains(outStr, longRule) {
		t.Fatalf("expected long rule id truncated, got:\n%s", outStr)
	}

	for _, line := range outLines(outStr) {
		if strings.Contains(line, "found in") {
			continue
		}
		if strings.Contains(line, "error") && len(line) > 220 {
			t.Fatalf("expected bounded row width, got len %d:\n%s", len(line), line)
		}
	}
}

func TestTextReporterAppliesColorsToPathMessageAndRule(t *testing.T) {
	// FORCE_COLOR=1 makes supportsColor return true for any io.Writer,
	// including bytes.Buffer, so we can assert on the emitted ANSI codes.
	t.Setenv("NO_COLOR", "")
	t.Setenv("FORCE_COLOR", "1")

	var output bytes.Buffer
	reporter := &TextReporter{}
	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:     "method-description",
			Message:    "missing description",
			Path:       []string{"$['methods'][0]['description']"},
			PathLabels: types.PathLabels{Method: "eth_getLogs", Schema: "Bad block"},
			Severity:   types.SeverityError,
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	outStr := output.String()
	wantPath := ansiDarkGrey + "methods[0].description" + ansiReset
	if !strings.Contains(outStr, wantPath) {
		t.Fatalf("expected dark-grey path %q in output, got:\n%s", wantPath, outStr)
	}
	wantMsg := ansiLightGrey + "missing description" + ansiReset
	if !strings.Contains(outStr, wantMsg) {
		t.Fatalf("expected light-grey message %q in output, got:\n%s", wantMsg, outStr)
	}
	wantRule := ansiDarkGrey + "method-description" + ansiReset
	if !strings.Contains(outStr, wantRule) {
		t.Fatalf("expected dark-grey rule %q in output, got:\n%s", wantRule, outStr)
	}
	// Continuation lines (e.g. `schema: "Bad block"`) should also use dark grey.
	wantSec := ansiDarkGrey + `schema: "Bad block"` + ansiReset
	if !strings.Contains(outStr, wantSec) {
		t.Fatalf("expected dark-grey secondary label %q in output, got:\n%s", wantSec, outStr)
	}
	// Column header row stays plain — no color codes immediately around "path".
	headerIdx := strings.Index(outStr, "  path  ")
	if headerIdx < 0 {
		t.Fatalf("expected plain `path` header cell, got:\n%s", outStr)
	}
}

func TestTextReporterOmitsSourceFileHeaderWhenNoViolations(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{SourceFile: "openrpc.json"}
	if err := reporter.Format(nil, 2, &output); err != nil {
		t.Fatalf("Format returned error: %v", err)
	}
	if strings.Contains(output.String(), "openrpc.json") {
		t.Fatalf("expected no source file header on clean runs, got:\n%s", output.String())
	}
}

// writeProbe fails at a chosen write so every output boundary is exercised.
type writeProbe struct {
	calls  int
	failAt int
	err    error
}

func (w *writeProbe) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		return 0, w.err
	}
	return len(p), nil
}

func TestTextReporterReturnsWriteErrors(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("FORCE_COLOR", "")
	reporter := TextReporter{SourceFile: "openrpc.json"}
	cases := [][]types.RuleFunctionResult{
		nil,
		{{Message: "missing description", RuleID: "description", Path: []string{"$['methods'][0]['params'][0]['description']"},
			PathLabels: types.PathLabels{Method: "ping", Param: "id"}}},
	}
	for caseIndex, results := range cases {
		probe := &writeProbe{}
		if err := reporter.Format(results, 1, probe); err != nil {
			t.Fatal(err)
		}
		for failAt := 1; failAt <= probe.calls; failAt++ {
			t.Run(fmt.Sprintf("case=%d/write=%d", caseIndex, failAt), func(t *testing.T) {
				want := errors.New("write failed")
				writer := &writeProbe{failAt: failAt, err: want}
				if err := reporter.Format(results, 1, writer); !errors.Is(err, want) {
					t.Fatalf("got %v, want %v", err, want)
				}
				if writer.calls != failAt {
					t.Fatalf("writes continue after error: got %d, want %d", writer.calls, failAt)
				}
			})
		}
	}
}
