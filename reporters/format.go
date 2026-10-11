package reporters

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/open-rpc/openrpc-linter/types"
)

const (
	colPath    = 48
	colMessage = 56
	colRule    = 32
)

// Group kinds, ordered by render priority (lower first).
const (
	groupKindMethod     = "method"
	groupKindSchema     = "schema"
	groupKindDescriptor = "descriptor"
	groupKindTag        = "tag"
	groupKindSection    = "section"
	groupKindGeneral    = "general"
)

func severityLabel(severity types.Severity) string {
	switch effectiveSeverity(severity) {
	case types.SeverityWarn:
		return "warning"
	case types.SeverityInfo:
		return "info"
	default:
		return "error"
	}
}

// truncate shortens s to at most max bytes, appending "…" when clipped.
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return s[:max-1] + "…"
}

// colWidth returns min(max, widest cell) for batch column padding.
func colWidth(cells []string, max int) int {
	w := 0
	for _, c := range cells {
		if n := len(c); n > w {
			w = n
		}
	}
	if w > max {
		return max
	}
	return w
}

// formatViolationLine renders one ESLint-style row from already-styled cells
// (pad + ANSI handled by styledCell / formatSeverityCol upstream). The rule
// cell is last so it needs no trailing pad.
func formatViolationLine(pathCell, severityCell, messageCell, ruleCell string) string {
	return fmt.Sprintf("  %s  %s  %s  %s\n", pathCell, severityCell, messageCell, ruleCell)
}

// formatColumnHeaderRow renders the once-at-top column header row, sized to
// the same widths as the rows so the cells line up. Header is always plain
// text (no color).
func formatColumnHeaderRow(pathW int) string {
	return fmt.Sprintf("  %-*s  %-*s  %-*s  %s\n",
		pathW, "path", severityColWidth, "level", colMessage, "message", "rule")
}

// resolveGroup picks the most specific anchor on labels and returns a
// (kind, name) pair. The kind drives both header rendering and group order.
func resolveGroup(labels types.PathLabels) (string, string) {
	if labels.Method != "" {
		return groupKindMethod, labels.Method
	}
	if labels.Section == "components" {
		if labels.Schema != "" {
			return groupKindSchema, labels.Schema
		}
		if labels.Descriptor != "" {
			return groupKindDescriptor, labels.Descriptor
		}
		if labels.Tag != "" {
			return groupKindTag, labels.Tag
		}
	}
	if labels.Section != "" {
		return groupKindSection, labels.Section
	}
	return groupKindGeneral, "general"
}

// formatGroupHeader renders the line printed above each group's rows.
//   - method/section/general → bare name
//   - schema/descriptor/tag → `<kind> "<name>"`
func formatGroupHeader(kind, name string) string {
	switch kind {
	case groupKindSchema, groupKindDescriptor, groupKindTag:
		return fmt.Sprintf("%s %q", kind, name)
	default:
		return name
	}
}

// formatSecondaryLabels returns the indented continuation lines for any
// anchor set on labels that is NOT the row's group key. Example: a row
// grouped by method "foo" whose schema title is "Bar" yields ["schema: \"Bar\""].
// Param is always shown when present (since the group is never "param").
func formatSecondaryLabels(labels types.PathLabels, groupKind string) []string {
	anchors := []struct {
		kind  string
		value string
	}{
		{"param", labels.Param},
		{groupKindDescriptor, labels.Descriptor},
		{groupKindSchema, labels.Schema},
		{groupKindTag, labels.Tag},
	}
	var out []string
	for _, anchor := range anchors {
		if anchor.value == "" || (anchor.kind != "param" && anchor.kind == groupKind) {
			continue
		}
		out = append(out, fmt.Sprintf("%s: %q", anchor.kind, anchor.value))
	}
	return out
}

// methodIndexRe extracts the methods[N] index from a normalized JSONPath so
// method groups can be ordered by document position rather than alphabetically.
var methodIndexRe = regexp.MustCompile(`\['methods'\]\[(\d+)\]`)

// methodIndexFromPath returns the array index of the first methods[N] segment
// in a normalized JSONPath, or -1 when no such segment exists.
func methodIndexFromPath(jsonPath string) int {
	m := methodIndexRe.FindStringSubmatch(jsonPath)
	if len(m) < 2 {
		return -1
	}
	idx, err := strconv.Atoi(m[1])
	if err != nil {
		return -1
	}
	return idx
}
