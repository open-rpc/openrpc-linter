package reporters

import (
	"bytes"
	"strings"
	"testing"

	"github.com/open-rpc/openrpc-linter/types"
)

func TestTextReporterIncludesPathWhenPresent(t *testing.T) {
	var output bytes.Buffer
	reporter := &TextReporter{}

	err := reporter.Format([]types.RuleFunctionResult{
		{
			RuleID:  "method-description",
			Message: "Missing required field 'description'",
			Path:    []string{"$['methods'][0]"},
		},
	}, 1, &output)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	expected := "❌ method-description at $['methods'][0]: Missing required field 'description'"
	if !strings.Contains(output.String(), expected) {
		t.Fatalf("expected output to contain %q, got:\n%s", expected, output.String())
	}
}
