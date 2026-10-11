package reporters

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/open-rpc/openrpc-linter/internal/paths"
	"github.com/open-rpc/openrpc-linter/types"
)

// TextReporter renders violations grouped by their most-specific document
// anchor (method, components schema/descriptor/tag, etc.). SourceFile is
// optional and, when set, prints once as a header before the rows.
type TextReporter struct {
	SourceFile string
}

func effectiveSeverity(severity types.Severity) types.Severity {
	if severity == "" {
		return types.SeverityError
	}
	return severity
}

// row holds the precomputed cells for one violation so we can size columns
// before printing.
type row struct {
	path       string
	severity   types.Severity
	message    string
	ruleID     string
	pathLabels types.PathLabels
	groupKind  groupKind
	groupName  string
}

type groupKey struct {
	kind groupKind
	name string
}

type bucket struct {
	kind     groupKind
	name     string
	rows     []row
	methodIx int // smallest methods[N] index across the rows (groupKindMethod only)
}

type pending struct {
	row       row
	methodIdx int
}

// textLayout carries the shared column width and color policy for one report.
type textLayout struct {
	pathWidth int
	color     bool
}

func (r *TextReporter) Format(results []types.RuleFunctionResult, totalRules int, output io.Writer) error {
	pendings, summary := collectRows(results)
	if len(pendings) == 0 {
		return writeChunks(output, []string{fmt.Sprintf("All %d rules passed\n", totalRules)})
	}
	chunks := violationReport(pendings, summary, r.SourceFile, supportsColor(output))
	return writeChunks(output, chunks)
}

// violationReport builds report text without touching the destination writer.
// Each chunk preserves one existing output boundary and its byte order.
func violationReport(pendings []pending, summary, source string, colorEnabled bool) []string {
	layout := textLayout{pathWidth: pathColumnWidth(pendings), color: colorEnabled}
	var body []string
	for _, group := range groupRows(pendings) {
		body = append(body, group.render(layout)...)
	}
	return slices.Concat(reportHeader(source, summary, layout.pathWidth), body, []string{"\n" + summary + "\n"})
}

func reportHeader(source, summary string, pathW int) []string {
	header := []string{summary + "\n\n", formatColumnHeaderRow(pathW)}
	if source == "" {
		return header
	}
	return slices.Concat([]string{source + "\n\n"}, header)
}

// writeChunks is the report's single output boundary; failures stop all writes.
func writeChunks(output io.Writer, chunks []string) error {
	for _, chunk := range chunks {
		written, err := io.WriteString(output, chunk)
		if err != nil {
			return err
		}
		if written != len(chunk) {
			return io.ErrShortWrite
		}
	}
	return nil
}

// severityIndex shares the default-to-error policy used by report labels.
func severityIndex(severity types.Severity) int {
	switch effectiveSeverity(severity) {
	case types.SeverityWarn:
		return 1
	case types.SeverityInfo:
		return 2
	default:
		return 0
	}
}

func collectRows(results []types.RuleFunctionResult) ([]pending, string) {
	pendings := make([]pending, 0, len(results))
	counts := [3]int{}
	rules := make(map[string]struct{})
	for _, result := range results {
		if result.Message == "" {
			continue
		}
		counts[severityIndex(result.Severity)]++
		rules[result.RuleID] = struct{}{}
		pendings = append(pendings, pendingRow(result))
	}
	return pendings, formatViolationSummary(counts[0], counts[1], counts[2], len(rules))
}

func pendingRow(result types.RuleFunctionResult) pending {
	path := ""
	methodIndex := -1
	if len(result.Path) > 0 {
		position, _ := paths.Parse(result.Path[0])
		path = truncate(position.Friendly(), colPath)
		methodIndex = position.MethodIndex()
	}
	kind, name := resolveGroup(result.PathLabels)
	return pending{
		row: row{
			path:       path,
			severity:   result.Severity,
			message:    truncate(result.Message, colMessage),
			ruleID:     truncate(result.RuleID, colRule),
			pathLabels: result.PathLabels,
			groupKind:  kind,
			groupName:  name,
		},
		methodIdx: methodIndex,
	}
}

// groupRows owns the grouping table. Buckets are complete before publication.
func groupRows(pendings []pending) []*bucket {
	members := make(map[groupKey][]pending)
	for _, p := range pendings {
		key := groupKey{kind: p.row.groupKind, name: p.row.groupName}
		members[key] = append(members[key], p)
	}
	groups := make([]*bucket, 0, len(members))
	for key, rows := range members {
		groups = append(groups, bucketFor(key, rows))
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].less(groups[j]) })
	return groups
}

// bucketFor derives method position and row order from owned members.
func bucketFor(key groupKey, members []pending) *bucket {
	rows := make([]row, len(members))
	methodIndex := -1
	for i, member := range members {
		rows[i] = member.row
		if member.methodIdx >= 0 && (methodIndex < 0 || member.methodIdx < methodIndex) {
			methodIndex = member.methodIdx
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].less(rows[j]) })
	return &bucket{kind: key.kind, name: key.name, rows: rows, methodIx: methodIndex}
}

func (r row) less(other row) bool {
	if r.path != other.path {
		return r.path < other.path
	}
	if r.ruleID != other.ruleID {
		return r.ruleID < other.ruleID
	}
	return r.message < other.message
}

func (b *bucket) less(other *bucket) bool {
	if b.kind != other.kind {
		return b.kind < other.kind
	}
	if b.kind != groupKindMethod {
		return b.name < other.name
	}
	return b.methodLess(other)
}

// methodLess puts methods with a document index before methods without one.
func (b *bucket) methodLess(other *bucket) bool {
	if b.methodIx == other.methodIx {
		return b.name < other.name
	}
	if b.methodIx < 0 {
		return false
	}
	if other.methodIx < 0 {
		return true
	}
	return b.methodIx < other.methodIx
}

func pathColumnWidth(pendings []pending) int {
	// Compute path column width across ALL rows so every group lines up.
	allPaths := make([]string, 0, len(pendings))
	for _, p := range pendings {
		allPaths = append(allPaths, p.row.path)
	}
	pathW := colWidth(allPaths, colPath)
	// Header word "path" is 4 chars; never shrink below that or the header
	// row would overflow its cell.
	if pathW < len("path") {
		pathW = len("path")
	}

	return pathW
}

func (b *bucket) render(layout textLayout) []string {
	chunks := []string{"\n" + formatGroupHeader(b.kind, b.name) + "\n"}
	for _, row := range b.rows {
		chunks = append(chunks, row.render(layout)...)
	}
	return chunks
}

// render keeps a row and its continuation labels together.
func (r row) render(layout textLayout) []string {
	secondary := formatSecondaryLabels(r.pathLabels, r.groupKind)
	chunks := make([]string, len(secondary)+1)
	chunks[0] = r.formattedLine(layout)
	for i, label := range secondary {
		styled := colorize(truncate(label, colMessage), ansiDarkGrey, layout.color)
		chunks[i+1] = "    " + styled + "\n"
	}
	return chunks
}

// formattedLine owns column styling; the bucket owns row order; the writer owns output errors.
func (r row) formattedLine(layout textLayout) string {
	pathCell := styledCell(r.path, layout.pathWidth, ansiDarkGrey, layout.color)
	severityCell := formatSeverityCol(r.severity, layout.color)
	messageCell := styledCell(r.message, colMessage, ansiLightGrey, layout.color)
	ruleCell := colorize(r.ruleID, ansiDarkGrey, layout.color)
	return formatViolationLine(pathCell, severityCell, messageCell, ruleCell)
}

func formatViolationSummary(errorCount, warnCount, infoCount, rulesWithViolations int) string {
	counts := []struct {
		count int
		label string
	}{
		{errorCount, plural(errorCount, "error", "errors")},
		{warnCount, plural(warnCount, "warning", "warnings")},
		{infoCount, "info"},
	}
	var parts []string
	for _, severity := range counts {
		if severity.count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", severity.count, severity.label))
		}
	}

	ruleLabel := "rules"
	if rulesWithViolations == 1 {
		ruleLabel = "rule"
	}
	return fmt.Sprintf("%s found in %d %s", strings.Join(parts, ", "), rulesWithViolations, ruleLabel)
}

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return singular
	}
	return pluralForm
}
