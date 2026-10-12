// Package paths validates the concrete JSONPaths carried by diagnostics.
package paths

import (
	"strconv"
	"strings"

	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
)

// Path retains the source text and its concrete selectors. A failed parse has
// no selectors, so display preserves source text and location lookup stays empty.
type Path struct {
	source    string
	selectors spec.NormalizedPath
}

// Parse accepts only concrete names and nonnegative indexes. Relative bracket
// notation is accepted for existing display callers. Queries are not locations.
func Parse(path string) (Path, bool) {
	result := Path{source: path}
	if strings.HasPrefix(path, "[") {
		path = "$" + path
	}
	query, err := jsonpath.Parse(path)
	if err != nil {
		return result, false
	}
	var location spec.NormalizedPath
	for _, segment := range query.Query().Segments() {
		selectors := segment.Selectors()
		if segment.IsDescendant() || len(selectors) != 1 {
			return result, false
		}
		selector, concrete := selectors[0].(spec.NormalSelector)
		if !concrete {
			return result, false
		}
		if index, ok := selector.(spec.Index); ok && index < 0 {
			return result, false
		}
		location = append(location, selector)
	}
	result.selectors = location
	return result, true
}

// Selectors returns a read-only view used by the document label resolver.
func (path Path) Selectors() spec.NormalizedPath {
	return path.selectors
}

func (path Path) Friendly() string {
	if len(path.selectors) == 0 {
		return path.source
	}
	var out []byte
	for _, selector := range path.selectors {
		part := friendlySegment(selector)
		if name, named := selector.(spec.Name); named && name != "" && len(out) > 0 {
			part = "." + part
		}
		out = append(out, part...)
	}
	return string(out)
}

func friendlySegment(selector spec.NormalSelector) string {
	if index, indexed := selector.(spec.Index); indexed {
		return "[" + index.String() + "]"
	}
	name, _ := selector.(spec.Name)
	if name == "" {
		return "['']"
	}
	return escapedName(name)
}

// escapedName keeps decoded keys printable without adding string delimiters.
func escapedName(name spec.Name) string {
	quoted := strconv.Quote(string(name))
	return quoted[1 : len(quoted)-1]
}

func (path Path) MethodIndex() int {
	for i := 0; i+1 < len(path.selectors); i++ {
		name, named := path.selectors[i].(spec.Name)
		if !named || name != "methods" {
			continue
		}
		if index, indexed := path.selectors[i+1].(spec.Index); indexed {
			return int(index)
		}
	}
	return -1
}
