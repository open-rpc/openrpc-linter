// Package selector turns a JSONPath rule.Given plus a (pre-$ref-resolved)
// OpenRPC document into a uniform []Target that every rule function can
// consume. It uses the OpenRPC meta-schema to answer "where could field X
// validly exist?" so descendant queries (e.g. $..description) can report
// missing fields, not just existing ones.
package selector

import (
	"github.com/theory/jsonpath/spec"
)

// Target is the uniform unit that rule functions evaluate.//
//   - Value mode  : Field == ""        ; Exists == true; Node is the value.
//   - Field mode  : Field != ""        ; Exists indicates presence on Parent.
type Target struct {
	Path        spec.NormalizedPath
	Node        any
	Exists      bool
	Field       string
	Parent      any
	ParentPath  spec.NormalizedPath
	SchemaTitle string
}

func (t Target) PathString() string {
	if len(t.Path) == 0 {
		return "$"
	}
	return t.Path.String()
}

// IsUnder reports whether child has scope as a structural prefix. Used to
// confine descendant queries (e.g. $.methods..description) to candidates
// that actually live inside the selected scope.
func IsUnder(child, scope spec.NormalizedPath) bool {
	if len(child) < len(scope) {
		return false
	}
	return child[:len(scope)].Compare(scope) == 0
}

// fieldTarget builds a Target for a named field on a parent map.
func fieldTarget(parent any, parentPath spec.NormalizedPath, field string, schemaTitle string) Target {
	value, exists := mapField(parent, field)
	childPath := append(append(spec.NormalizedPath{}, parentPath...), spec.Name(field))
	return Target{
		Path:        childPath,
		Node:        value,
		Exists:      exists,
		Field:       field,
		Parent:      parent,
		ParentPath:  parentPath,
		SchemaTitle: schemaTitle,
	}
}

// valueTarget builds a Target for an existing JSONPath-selected value.
func valueTarget(path spec.NormalizedPath, node any) Target {
	return Target{
		Path:   append(spec.NormalizedPath{}, path...),
		Node:   node,
		Exists: true,
	}
}

func mapField(value any, field string) (any, bool) {
	if obj, ok := value.(map[string]any); ok {
		v, ok := obj[field]
		return v, ok
	}
	return nil, false
}
