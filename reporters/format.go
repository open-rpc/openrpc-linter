package reporters

import (
	"fmt"
	"unicode/utf8"

	"github.com/open-rpc/openrpc-linter/types"
)

const (
	colPath    = 48
	colMessage = 56
	colRule    = 32
)

// groupKind fixes render priority in the type rather than a mutable map.
type groupKind uint8

const (
	groupKindMethod groupKind = iota
	groupKindSchema
	groupKindDescriptor
	groupKindTag
	groupKindSection
	groupKindGeneral
	groupKindParam // secondary label only
)

func (kind groupKind) label() string {
	return [...]string{"method", "schema", "descriptor", "tag", "section", "general", "param"}[kind]
}

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

// truncate preserves UTF-8 and reserves the byte width of the ellipsis.
// Limits smaller than the ellipsis itself still show the complete ellipsis.
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	limit := max - len("…")
	if limit < 0 {
		return "…"
	}
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return s[:limit] + "…"
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
func resolveGroup(labels types.PathLabels) (groupKind, string) {
	switch {
	case labels.Method != "":
		return groupKindMethod, labels.Method
	case labels.Section == "components":
		return componentGroup(labels)
	case labels.Section != "":
		return groupKindSection, labels.Section
	default:
		return groupKindGeneral, "general"
	}
}

func componentGroup(labels types.PathLabels) (groupKind, string) {
	switch {
	case labels.Schema != "":
		return groupKindSchema, labels.Schema
	case labels.Descriptor != "":
		return groupKindDescriptor, labels.Descriptor
	case labels.Tag != "":
		return groupKindTag, labels.Tag
	default:
		return groupKindSection, "components"
	}
}

// formatGroupHeader renders the line printed above each group's rows.
//   - method/section/general → bare name
//   - schema/descriptor/tag → `<kind> "<name>"`
func formatGroupHeader(kind groupKind, name string) string {
	switch kind {
	case groupKindSchema, groupKindDescriptor, groupKindTag:
		return fmt.Sprintf("%s %q", kind.label(), name)
	default:
		return name
	}
}

// formatSecondaryLabels returns the indented continuation lines for any
// anchor set on labels that is NOT the row's group key. Example: a row
// grouped by method "foo" whose schema title is "Bar" yields ["schema: \"Bar\""].
// Param is always shown when present (since the group is never "param").
func formatSecondaryLabels(labels types.PathLabels, selectedKind groupKind) []string {
	anchors := []struct {
		kind  groupKind
		value string
	}{
		{groupKindParam, labels.Param},
		{groupKindDescriptor, labels.Descriptor},
		{groupKindSchema, labels.Schema},
		{groupKindTag, labels.Tag},
	}
	var out []string
	for _, anchor := range anchors {
		if anchor.value == "" || anchor.kind == selectedKind {
			continue
		}
		out = append(out, fmt.Sprintf("%s: %q", anchor.kind.label(), anchor.value))
	}
	return out
}
