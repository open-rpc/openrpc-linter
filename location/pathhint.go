// Package location resolves document-position labels from JSONPath strings.
// pathhint.go implements path-derived hints; source file line/column is deferred.
package location

import (
	"github.com/open-rpc/openrpc-linter/internal/paths"
	"github.com/open-rpc/openrpc-linter/types"
	"github.com/theory/jsonpath/spec"
)

// Resolve derives PathLabels by walking doc along the normalized JSONPath.
func Resolve(doc any, path string) types.PathLabels {
	position, _ := paths.Parse(path)
	segments := position.Selectors()
	var labels types.PathLabels
	current := doc
	for i, segment := range segments {
		previousKey := ""
		if i > 0 {
			name, _ := segments[i-1].(spec.Name)
			previousKey = string(name)
		}
		next, present, valid := selectNode(segment, current)
		if !valid {
			return types.PathLabels{}
		}
		if name, named := segment.(spec.Name); named {
			labels = labelsAtKey(labels, string(name), previousKey, next)
		} else {
			labels = labelsAtIndex(labels, previousKey, next)
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
func selectNode(selector spec.NormalSelector, current any) (node any, present, valid bool) {
	if index, indexed := selector.(spec.Index); indexed {
		array, ok := current.([]any)
		if !ok || int(index) < 0 || int(index) >= len(array) {
			return nil, false, false
		}
		return array[int(index)], true, true
	}
	object, ok := current.(map[string]any)
	if !ok {
		return nil, false, false
	}
	node, present = object[string(selector.(spec.Name))]
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
		if title := stringField(node, "title"); title != "" {
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
	if name := stringField(node, field); name != "" {
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
	path, _ := paths.Parse(jsonPath)
	return path.Friendly()
}

func stringField(node any, field string) string {
	object, _ := node.(map[string]any)
	value, _ := object[field].(string)
	return value
}
