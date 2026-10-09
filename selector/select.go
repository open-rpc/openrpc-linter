package selector

import (
	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
)

// Select turns a parsed JSONPath query into the uniform []Target stream
// that rule functions consume. It encapsulates the three modes the
// linter actually needs:
//
//   - value mode: terminal segment is anything but a single name (or path
//     is empty) — we just hand back the JSONPath-selected nodes.
//   - parent-field mode: terminal segment is a single .name selector —
//     we strip it, select parents, and emit a Field target per parent
//     (Exists=false reports a missing field).
//   - descendant-field mode (incl. compound) — uses the schema-aware
//     Index to find candidate parents that may contain the terminal field,
//     restricted to the scope before the .. segment.
//
// The function is intentionally recursive: compound descendants like
// $..foo.bar fall out as "do the descendant step, then for each result
// evaluate the remaining segments as a fresh query". No phase pipeline.
func Select(p *jsonpath.Path, doc any, idx *Index) []Target {
	if p == nil {
		return nil
	}
	segments := p.Query().Segments()
	return selectSegments(segments, doc, doc, spec.NormalizedPath{}, idx)
}

// selectSegments evaluates segments against current rooted at root.
// basePath is the canonical path of current within root (empty for the
// initial call). idx is the precomputed schema-aware Index.
func selectSegments(
	segments []*spec.Segment,
	current any,
	root any,
	basePath spec.NormalizedPath,
	idx *Index,
) []Target {
	if len(segments) == 0 {
		return []Target{valueTarget(basePath, current)}
	}

	// Find the *last* descendant segment. Everything up to and including
	// it is the "descendant phase"; the suffix is evaluated as a normal
	// JSONPath against each descendant-phase result. This keeps the
	// implementation a single recursion rather than a state machine.
	lastDescendant := -1
	for i, seg := range segments {
		if seg.IsDescendant() {
			lastDescendant = i
		}
	}

	last := segments[len(segments)-1]

	// Case 1: terminal single-name child segment — classic parent-field mode.
	// $.info.description, $.methods[*].description, $.methods
	if lastDescendant == -1 {
		if field, ok := terminalName(last); ok {
			parentPath := jsonpath.New(spec.Query(true, segments[:len(segments)-1]...))
			parents := parentPath.SelectLocated(current)
			out := make([]Target, 0, len(parents))
			for _, parent := range parents {
				out = append(out, fieldTarget(parent.Node, parent.Path, field, candidateTitle(idx, parent.Path)))
			}
			return out
		}
		// No terminal-name simplification possible — pure value mode.
		return valueModeTargets(jsonpath.New(spec.Query(true, segments...)), current)
	}

	// Case 2: descendant present somewhere.
	descSeg := segments[lastDescendant]
	prefix := segments[:lastDescendant]
	suffix := segments[lastDescendant+1:]

	// 2a) Pure descendant terminal: $..f or $.scope..f. The descendant
	//     selector is a single .name — use the Index so we report missing
	//     candidates, not just existing ones.
	if len(suffix) == 0 {
		if field, ok := terminalName(descSeg); ok && idx != nil {
			return descendantFieldTargets(prefix, field, current, idx)
		}
		return valueModeTargets(jsonpath.New(spec.Query(true, segments...)), current)
	}

	// 2b) Compound descendant: $.scope..f.rest. Do the descendant step
	//     first using whatever the descendant segment is (named or not),
	//     then re-run Select with the remaining segments rooted at each
	//     intermediate node. This is the recursive case
	midSegments := append(append([]*spec.Segment{}, prefix...), descSeg)
	midTargets := selectSegments(midSegments, current, root, spec.NormalizedPath{}, idx)

	out := make([]Target, 0, len(midTargets))
	for _, mid := range midTargets {
		if !mid.Exists {
			// A missing intermediate (e.g. .foo not present) has no further
			// children to walk. Surface it directly only when there is no
			// suffix; in compound queries we just skip it because we can't
			// keep descending into nothing.
			continue
		}
		// Evaluate the suffix as a standalone query rooted at mid.Node.
		// We rebase by prepending mid.Path to each emitted target's path.
		subTargets := selectSegments(suffix, mid.Node, mid.Node, mid.Path, idx)
		for _, t := range subTargets {
			out = append(out, rebase(t, mid.Path))
		}
	}
	return out
}

// descendantFieldTargets implements the schema-aware step: take the scope
// before "..f", consult Index.ByField[f] for every candidate parent the
// meta-schema says may legally contain f, keep only candidates under the
// scope, and emit a Field target per candidate. This is what makes
// $..description surface MISSING descriptions, not just present ones.
func descendantFieldTargets(prefix []*spec.Segment, field string, doc any, idx *Index) []Target {
	var scopes []*spec.LocatedNode
	if len(prefix) == 0 {
		// Root scope: every candidate qualifies, no prefix filtering needed.
		scopes = []*spec.LocatedNode{{Path: spec.NormalizedPath{}, Node: doc}}
	} else {
		scopePath := jsonpath.New(spec.Query(true, prefix...))
		scopes = scopePath.SelectLocated(doc)
	}

	candidates := idx.ByField[field]
	if len(candidates) == 0 || len(scopes) == 0 {
		return nil
	}

	out := make([]Target, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, cand := range candidates {
		for _, scope := range scopes {
			if !IsUnder(cand.Path, scope.Path) {
				continue
			}
			key := cand.Path.String() + "/" + field
			if _, dup := seen[key]; dup {
				break
			}
			seen[key] = struct{}{}
			out = append(out, fieldTarget(cand.Node, cand.Path, field, cand.SchemaTitle))
			break
		}
	}
	return out
}

// valueModeTargets evaluates path as-is and emits Targets for each
// located node — the fallback used when no terminal-field shortcut
// applies (e.g. $.methods[*], $.methods[?@.deprecated]).
func valueModeTargets(path *jsonpath.Path, doc any) []Target {
	nodes := path.SelectLocated(doc)
	out := make([]Target, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, valueTarget(n.Path, n.Node))
	}
	return out
}

// terminalName returns the single Name selector inside seg, if seg is a
// non-descendant child segment with exactly one Name. Anything else
// (wildcard, slice, filter, index, multi-selector, descendant-with-extras)
// returns false — those cases fall through to value mode.
func terminalName(seg *spec.Segment) (string, bool) {
	sels := seg.Selectors()
	if len(sels) != 1 {
		return "", false
	}
	name, ok := sels[0].(spec.Name)
	if !ok || name == "" {
		return "", false
	}
	return string(name), true
}

// candidateTitle looks up the matched schema title for a parent path so
// diagnostics can say e.g. "Missing field 'description' on
// methodObject at ...". Returns "" if the parent wasn't indexed.
func candidateTitle(idx *Index, path spec.NormalizedPath) string {
	if idx == nil {
		return ""
	}
	if c, ok := idx.ByPath[path.String()]; ok {
		return c.SchemaTitle
	}
	return ""
}

// rebase prepends prefix to a Target's Path and ParentPath. Used when
// compound-descendant recursion re-emits a sub-target whose path was
// computed relative to an intermediate node.
func rebase(t Target, prefix spec.NormalizedPath) Target {
	if len(prefix) == 0 {
		return t
	}
	t.Path = append(append(spec.NormalizedPath{}, prefix...), t.Path...)
	if t.ParentPath != nil {
		t.ParentPath = append(append(spec.NormalizedPath{}, prefix...), t.ParentPath...)
	}
	return t
}
