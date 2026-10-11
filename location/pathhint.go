// Package location resolves document-position labels from JSONPath strings.
// pathhint.go implements path-derived hints; source file line/column is deferred.
package location

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/open-rpc/openrpc-linter/types"
)

var jsonPathSegmentRe = regexp.MustCompile(`\['([^']+)'\]|\[(\d+)\]`)

type pathSegment struct {
	key   string
	index int
	isIdx bool
}

// Resolve derives PathLabels by walking doc along the normalized JSONPath.
func Resolve(doc any, path string) types.PathLabels {
	segments := parseJSONPath(path)
	var labels types.PathLabels
	current := doc
	for i, segment := range segments {
		previousKey := ""
		if i > 0 && !segments[i-1].isIdx {
			previousKey = segments[i-1].key
		}
		next, present, valid := segment.selectNode(current)
		if !valid {
			return types.PathLabels{}
		}
		if segment.isIdx {
			labels = labelsAtIndex(labels, previousKey, next)
		} else {
			labels = labelsAtKey(labels, segment.key, previousKey, next)
		}
		if !present {
			return labels
		}
		current = next
	}
	return labels
}

// selectNode distinguishes an absent object field from an invalid traversal.
// Absent fields retain the labels collected so far; invalid traversal clears them.
func (s pathSegment) selectNode(current any) (node any, present, valid bool) {
	if s.isIdx {
		array, ok := current.([]any)
		if !ok || s.index < 0 || s.index >= len(array) {
			return nil, false, false
		}
		return array[s.index], true, true
	}
	object, ok := current.(map[string]any)
	if !ok {
		return nil, false, false
	}
	node, present = object[s.key]
	return node, present, true
}

func labelsAtKey(labels types.PathLabels, key, previousKey string, node any) types.PathLabels {
	switch key {
	case "info", "components", "methods":
		labels.Section = key
	}
	if previousKey == "schemas" && labels.Section == "components" {
		labels.Schema = key
	}
	if key == "schema" && labels.Method != "" {
		if title, ok := stringField(node, "title"); ok {
			labels.Schema = title
		}
	}
	return labels
}

func labelsAtIndex(labels types.PathLabels, collection string, node any) types.PathLabels {
	destination, field := indexLabel(&labels, collection)
	if destination == nil {
		return labels
	}
	if name, ok := stringField(node, field); ok {
		*destination = name
	}
	return labels
}

// indexLabel selects the field and source property for one array collection.
// Returning a field pointer avoids building a map of addresses on every step.
func indexLabel(labels *types.PathLabels, collection string) (*string, string) {
	switch collection {
	case "methods":
		return &labels.Method, "name"
	case "params":
		return &labels.Param, "name"
	case "contentDescriptors":
		return &labels.Descriptor, "name"
	case "tags":
		return &labels.Tag, "name"
	case "schemas":
		if labels.Section == "components" {
			return &labels.Schema, "title"
		}
	}
	return nil, ""
}

// Enrich fills PathLabels on each result from the resolved document.
func Enrich(results []types.RuleFunctionResult, doc any) {
	for i := range results {
		if len(results[i].Path) == 0 {
			continue
		}
		results[i].PathLabels = Resolve(doc, results[i].Path[0])
	}
}

// FriendlyPath converts a normalized JSONPath to dot/bracket form for display.
func FriendlyPath(jsonPath string) string {
	if jsonPath == "" || jsonPath == "$" {
		return jsonPath
	}
	s := strings.TrimPrefix(jsonPath, "$")

	matches := jsonPathSegmentRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return jsonPath
	}

	var out []byte
	first := true
	for _, m := range matches {
		segment := "[" + m[2] + "]"
		if m[1] != "" {
			segment = m[1]
			if !first {
				segment = "." + segment
			}
			first = false
		}
		out = append(out, segment...)
	}
	return string(out)
}

func parseJSONPath(path string) []pathSegment {
	s := strings.TrimPrefix(path, "$")

	matches := jsonPathSegmentRe.FindAllStringSubmatch(s, -1)
	out := make([]pathSegment, 0, len(matches))
	for _, match := range matches {
		segment := pathSegment{key: match[1]}
		if match[1] == "" {
			index, _ := strconv.Atoi(match[2])
			segment = pathSegment{index: index, isIdx: true}
		}
		out = append(out, segment)
	}

	return out
}

func stringField(node any, field string) (string, bool) {
	object, _ := node.(map[string]any)
	value, ok := object[field].(string)
	return value, ok && value != ""
}
