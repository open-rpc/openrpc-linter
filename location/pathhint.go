// Package location resolves document-position labels from JSONPath strings.
// pathhint.go implements path-derived hints; source file line/column is deferred.
package location

import (
	"regexp"
	"strconv"

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
	segs := parseJSONPath(path)
	if len(segs) == 0 {
		return types.PathLabels{}
	}

	var labels types.PathLabels
	cur := doc

	for i := 0; i < len(segs); i++ {
		seg := segs[i]
		prevKey := ""
		if i > 0 && !segs[i-1].isIdx {
			prevKey = segs[i-1].key
		}

		switch {
		case !seg.isIdx:
			switch seg.key {
			case "info", "components", "methods":
				labels.Section = seg.key
			}

			m, ok := cur.(map[string]any)
			if !ok {
				return types.PathLabels{}
			}

			if prevKey == "schemas" && labels.Section == "components" {
				labels.Schema = seg.key
			}

			val, exists := m[seg.key]
			if !exists {
				return labels
			}
			cur = val

			if seg.key == "schema" && labels.Method != "" {
				if title, ok := stringField(cur, "title"); ok && title != "" {
					labels.Schema = title
				}
			}

		case seg.isIdx:
			arr, ok := cur.([]any)
			if !ok || seg.index < 0 || seg.index >= len(arr) {
				return types.PathLabels{}
			}
			item := arr[seg.index]

			switch prevKey {
			case "methods":
				if name, ok := stringField(item, "name"); ok {
					labels.Method = name
				}
			case "params":
				if name, ok := stringField(item, "name"); ok {
					labels.Param = name
				}
			case "contentDescriptors":
				if name, ok := stringField(item, "name"); ok {
					labels.Descriptor = name
				}
			case "tags":
				if name, ok := stringField(item, "name"); ok {
					labels.Tag = name
				}
			case "schemas":
				if labels.Section == "components" {
					if title, ok := stringField(item, "title"); ok && title != "" {
						labels.Schema = title
					}
				}
			}

			cur = item
		}
	}

	return labels
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
	s := jsonPath
	if len(s) > 0 && s[0] == '$' {
		s = s[1:]
	}

	matches := jsonPathSegmentRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return jsonPath
	}

	var out []byte
	first := true
	for _, m := range matches {
		if m[1] != "" {
			if first {
				out = append(out, m[1]...)
				first = false
			} else {
				out = append(out, '.')
				out = append(out, m[1]...)
			}
			continue
		}
		if m[2] != "" {
			out = append(out, '[')
			out = append(out, m[2]...)
			out = append(out, ']')
		}
	}
	return string(out)
}

func parseJSONPath(path string) []pathSegment {
	s := path
	if len(s) > 0 && s[0] == '$' {
		s = s[1:]
	}

	matches := jsonPathSegmentRe.FindAllStringSubmatch(s, -1)
	out := make([]pathSegment, 0, len(matches))
	for _, m := range matches {
		if m[1] != "" {
			out = append(out, pathSegment{key: m[1]})
			continue
		}
		if m[2] != "" {
			idx, _ := strconv.Atoi(m[2])
			out = append(out, pathSegment{index: idx, isIdx: true})
		}
	}
	return out
}

func stringField(node any, field string) (string, bool) {
	m, ok := node.(map[string]any)
	if !ok {
		return "", false
	}
	v, ok := m[field]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}
